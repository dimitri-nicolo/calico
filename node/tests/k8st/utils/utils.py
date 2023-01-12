# Copyright (c) 2018 Tigera, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import datetime
import functools
import json
import logging
import os
import random
import string
import subprocess
import time

from kubernetes import client, config

_log = logging.getLogger(__name__)

ROUTER_IMAGE = os.getenv("ROUTER_IMAGE", "calico/bird:latest")

# Helps with printing diags after a test.
class DiagsCollector(object):
    def __enter__(self):
        pass

    def __exit__(self, exc_type, exc_value, traceback):
        if exc_type is not None:
            # Print out diagnostics for the test. These will go to screen
            # on test failure.
            _log.info("===================================================")
            _log.info("============= COLLECTING DIAGS FOR TEST ===========")
            _log.info("===================================================")
            kubectl("get deployments,pods,svc,endpoints --all-namespaces -o wide")
            for resource in ["node", "bgpconfig", "bgppeer", "gnp", "felixconfig"]:
                _log.info("")
                calicoctl("get " + resource + " -o yaml")
            nodes, _, _ = node_info()
            for node in nodes:
                _log.info("")
                run("docker exec " + node + " ip r")
                run("docker exec " + node + " ip l")
            for pod_name in calico_node_pod_names():
                kubectl("exec -n kube-system %s -- cat /etc/calico/confd/config/bird_aggr.cfg" % pod_name,
                        allow_fail=True)
                kubectl("exec -n kube-system %s -- cat /etc/calico/confd/config/bird_ipam.cfg" % pod_name,
                        allow_fail=True)
                kubectl("logs -n kube-system %s" % pod_name,
                        allow_fail=True)
            _log.info("===================================================")
            _log.info("============= COLLECTED DIAGS FOR TEST ============")
            _log.info("===================================================")
        else:
            _log.info("===================================================")
            _log.info("========= TEST COMPLETED WITHOUT EXCEPTION ========")
            _log.info("===================================================")

def log_calico_node(node_ip):
    pod_name = run(" kubectl get pod -n kube-system -o wide | grep calico-node | grep %s | awk '{print $1}'" % node_ip)
    kubectl("logs %s -n kube-system " % pod_name.strip())

def start_external_node_with_bgp(name, bird_peer_config=None, bird6_peer_config=None):
    # Check how much disk space we have.
    run("df -h")

    # Setup external node: use privileged mode for setting routes.
    run("docker run -d --privileged --net=kind --name %s %s" % (name, ROUTER_IMAGE))

    # Check how much space there is inside the container.  We may need
    # to retry this, as it may take a while for the image to download
    # and the container to start running.
    while True:
        try:
            run("docker exec %s df -h" % name)
            break
        except subprocess.CalledProcessError:
            _log.exception("Container not ready yet")
            time.sleep(20)

    # Install curl and iproute2.
    run("docker exec %s apk add --no-cache curl iproute2" % name)

    # Set ECMP hash algrithm to L4 for a proper load balancing between nodes.
    run("docker exec %s sysctl -w net.ipv4.fib_multipath_hash_policy=1" % name)

    # Add "merge paths on" to the BIRD config.
    run("docker exec %s sed -i '/protocol kernel {/a merge paths on;' /etc/bird.conf" % name)
    run("docker exec %s sed -i '/protocol kernel {/a merge paths on;' /etc/bird6.conf" % name)

    if bird_peer_config:
        # Install desired peer config.
        output = run("docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' %s" % name)
        birdy_ip = output.strip()
        with open('peers.conf', 'w') as peerconfig:
            peerconfig.write(bird_peer_config.replace("ip@local", birdy_ip))
        run("docker cp peers.conf %s:/etc/bird/peers.conf" % name)
        run("rm peers.conf")
        run("docker exec %s birdcl configure" % name)

    elif bird6_peer_config:
        # Install desired peer config.
        output = run("docker inspect -f '{{range .NetworkSettings.Networks}}{{.GlobalIPv6Address}}{{end}}' %s" % name)
        birdy_ip = output.strip()
        run("docker exec %s sysctl -w net.ipv6.conf.all.disable_ipv6=0" % name)
        run("docker exec %s sysctl -w net.ipv6.conf.all.forwarding=1" % name)

        # Try to set net.ipv6.fib_multipath_hash_policy to get IPv6
        # ECMP load balancing by 5-tuple, but allow it to fail as
        # older kernels (e.g. Semaphore v2) don't have that setting.
        # It doesn't actually matter as we aren't currently testing
        # IPv6 ECMP behaviour in detail.
        run("docker exec %s sysctl -w net.ipv6.fib_multipath_hash_policy=1" % name,
            allow_fail=True)

        with open('peers.conf', 'w') as peerconfig:
            peerconfig.write(bird6_peer_config.replace("ip@local", birdy_ip))
        run("docker cp peers.conf %s:/etc/bird6/peers.conf" % name)
        run("rm peers.conf")
        run("docker exec %s birdcl6 configure" % name)

    return birdy_ip

def retry_until_success(fun,
                        retries=10,
                        wait_time=1,
                        ex_class=None,
                        log_exception=True,
                        function_args=None,
                        function_kwargs=None):
    """
    Retries function until no exception is thrown. If exception continues,
    it is reraised.
    :param fun: the function to be repeatedly called
    :param retries: the maximum number of times to retry the function.  A value
    of 0 will run the function once with no retries.
    :param wait_time: the time to wait between retries (in s)
    :param ex_class: The class of expected exceptions.
    :param log_exception: By default this function logs the exception if the
    function is still failing after max retries.   This log can sometimes be
    superfluous -- if e.g. the calling code is going to make a better log --
    and can be suppressed by setting this parameter to False.
    :param function_args: A list of arguments to pass to function
    :param function_kwargs: A dictionary of keyword arguments to pass to
                            function
    :returns: the value returned by function
    """
    if function_args is None:
        function_args = []
    if function_kwargs is None:
        function_kwargs = {}
    for retry in range(retries + 1):
        try:
            result = fun(*function_args, **function_kwargs)
        except Exception as e:
            if ex_class and e.__class__ is not ex_class:
                _log.exception("Hit unexpected exception in function - "
                               "not retrying.")
                stop_for_debug()
                raise
            if retry < retries:
                _log.debug("Hit exception in function - retrying: %s", e)
                time.sleep(wait_time)
            else:
                if log_exception:
                    _log.exception("Function %s did not succeed before "
                                   "timeout.", fun)
                stop_for_debug()
                raise
        else:
            # Successfully ran the function
            return result

def function_name(f):
    """
    A function that returns the name of the provided function as a string.
    This primarily exists to handle the fact that functools.partial is an
    imperfect wrapper.
    """
    if isinstance(f, functools.partial):
        f = f.func

    try:
        return f.__name__
    except Exception:
        return "<unknown function>"


def run(command, logerr=True, allow_fail=False, allow_codes=[], returnerr=False):
    out = ""
    _log.info("[%s] %s", datetime.datetime.now(), command)
    try:
        out = subprocess.check_output(command,
                                      shell=True,
                                      stderr=subprocess.STDOUT)
        _log.info("Output:\n%s", out)
    except subprocess.CalledProcessError as e:
        if logerr:
            _log.info("Failure output:\n%s", e.output)
        if not (allow_fail or e.returncode in allow_codes):
            raise
        if returnerr:
            return e.output
    return out


def curl(hostname, container="kube-node-extra"):
    if ':' in hostname:
        # It's an IPv6 address.
        hostname = '[' + hostname + ']'

    cmd = "docker exec %s curl --connect-timeout 2 -m 3 %s" % (container,
                                                               hostname)
    return run(cmd)


def kubectl(args, logerr=True, allow_fail=False, allow_codes=[], returnerr=False):
    if timeout == 0:
        cmd = "kubectl "
    else:
        cmd = "timeout -s %d kubectl " % timeout
    return run(cmd + args,
               logerr=logerr,
               allow_fail=allow_fail,
               allow_codes=allow_codes,
               returnerr=returnerr)

def calicoctl(args, allow_fail=False):
    return kubectl("exec -i -n kube-system calicoctl -- /calicoctl --allow-version-mismatch " + args,
                   allow_fail=allow_fail)


def calicoctl_apply_dict(object_dict):
    calicoctl("""apply -f - << EOF
%s
EOF
""" % json.dumps(object_dict))


def generate_unique_id(length, prefix=""):
    random_string = ''.join(random.choice(string.ascii_lowercase + string.digits) for _ in range(length))
    return "%s-%s" % (prefix, random_string)


# We have to define and use this static map, from each node name to
# its IPv6 address, because Kubernetes does not yet allow for an IPv6
# address field in its host resource.  The mappings here must match
# the code in tests/k8st/deploy_resources_on_kind_cluster.sh that assigns an IPv6
# address to each node.
ipv6_map = {
    "kind-control-plane": "2001:20::8",
    "kind-worker": "2001:20::1",
    "kind-worker2": "2001:20::2",
    "kind-worker3": "2001:20::3",
}


def node_info():
    nodes = []
    ips = []
    ip6s = []

    master_node = kubectl("get node --selector='node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].metadata.name}'")
    nodes.append(master_node)
    master_ip6 = kubectl("get node --selector='node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].status.addresses[1].address}'")
    ip6s.append(master_ip6)
    master_ip = kubectl("get node --selector='node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].status.addresses[0].address}'")
    ips.append(master_ip)

    for i in range(3):
        node = kubectl("get node --selector='!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[%d].metadata.name}'" % i)
        nodes.append(node)
        node_ip6 = kubectl("get node --selector='!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[%d].status.addresses[1].address}'" % i)
        ip6s.append(node_ip6)
        node_ip = kubectl("get node --selector='!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[%d].status.addresses[0].address}'" % i)
        ips.append(node_ip)
    return nodes, ips, ip6s

def stop_for_debug():
    _log.info("stop on file /code/stop")
    while os.path.isfile('/code/stop'):
        os.system("sleep 3")


def calico_node_pod_names():
    return kubectl("get po -n kube-system -l k8s-app=calico-node" +
                   " -o jsonpath='{.items[*].metadata.name}'").split()

def update_ds_env(ds, ns, env_vars):
        config.load_kube_config(os.environ.get('KUBECONFIG'))
        api = client.AppsV1Api(client.ApiClient())
        node_ds = api.read_namespaced_daemon_set(ds, ns, exact=True, export=False)
        for container in node_ds.spec.template.spec.containers:
            if container.name == ds:
                for k, v in env_vars.items():
                    _log.info("Set %s=%s", k, v)
                    env_present = False
                    for env in container.env:
                        if env.name == k:
                            if env.value == v:
                                env_present = True
                            else:
                                container.env.remove(env)

                    if not env_present:
                        v1_ev = client.V1EnvVar(name=k, value=v, value_from=None)
                        container.env.append(v1_ev)
        api.replace_namespaced_daemon_set(ds, ns, node_ds)

        # Wait until the DaemonSet reports that all nodes have been updated.
        # In the past we've seen that the calico-node on kind-control-plane can
        # hang, in a not Ready state, for about 15 minutes.  Here we want to
        # detect in case that happens again, and fail the test case if so.  We
        # do that by querying the number of nodes that have been updated, every
        # 10s, and failing the test if that number does not change for 4 cycles
        # i.e. for 40s.
        last_number = 0
        iterations_with_no_change = 0
        while True:
            time.sleep(10)
            node_ds = api.read_namespaced_daemon_set_status("calico-node", "kube-system")
            _log.info("%d/%d nodes updated",
                      node_ds.status.updated_number_scheduled,
                      node_ds.status.desired_number_scheduled)
            if node_ds.status.updated_number_scheduled == node_ds.status.desired_number_scheduled:
                break
            if node_ds.status.updated_number_scheduled == last_number:
                iterations_with_no_change += 1
                if iterations_with_no_change == 4:
                    run("docker exec kind-control-plane conntrack -L", allow_fail=True)
                    raise Exception("calico-node DaemonSet update failed to make progress for 40s")
            else:
                last_number = node_ds.status.updated_number_scheduled
                iterations_with_no_change = 0

        # Wait until all calico-node pods are ready.
        kubectl("wait pod --for=condition=Ready -l k8s-app=calico-node -n kube-system --timeout=300s")

