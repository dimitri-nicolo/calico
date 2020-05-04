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
import json
import logging
import os
import subprocess
import time
from pprint import pformat
from unittest import TestCase

from deepdiff import DeepDiff
from kubernetes import client, config

from utils.utils import retry_until_success, run, kubectl

logger = logging.getLogger(__name__)


first_log_time = None


class TestBase(TestCase):

    """
    Base class for test-wide methods.
    """

    def setUp(self):
        """
        Set up before every test.
        """
        super(TestBase, self).setUp()
        self.cluster = self.k8s_client()
        self.cleanups = []

        # Log a newline to ensure that the first log appears on its own line.
        logger.info("")

    def tearDown(self):
        for cleanup in reversed(self.cleanups):
            cleanup()
        super(TestBase, self).tearDown()

    def add_cleanup(self, cleanup):
        self.cleanups.append(cleanup)

    @staticmethod
    def assert_same(thing1, thing2):
        """
        Compares two things.  Debug logs the differences between them before
        asserting that they are the same.
        """
        assert cmp(thing1, thing2) == 0, \
            "Items are not the same.  Difference is:\n %s" % \
            pformat(DeepDiff(thing1, thing2), indent=2)

    @staticmethod
    def writejson(filename, data):
        """
        Converts a python dict to json and outputs to a file.
        :param filename: filename to write
        :param data: dictionary to write out as json
        """
        with open(filename, 'w') as f:
            text = json.dumps(data,
                              sort_keys=True,
                              indent=2,
                              separators=(',', ': '))
            logger.debug("Writing %s: \n%s" % (filename, text))
            f.write(text)

    @staticmethod
    def log_banner(msg, *args, **kwargs):
        global first_log_time
        time_now = time.time()
        if first_log_time is None:
            first_log_time = time_now
        time_now -= first_log_time
        elapsed_hms = "%02d:%02d:%02d " % (time_now / 3600,
                                           (time_now % 3600) / 60,
                                           time_now % 60)

        level = kwargs.pop("level", logging.INFO)
        msg = elapsed_hms + str(msg) % args
        banner = "+" + ("-" * (len(msg) + 2)) + "+"
        logger.log(level, "\n" +
                   banner + "\n"
                            "| " + msg + " |\n" +
                   banner)

    @staticmethod
    def k8s_client():
        config.load_kube_config(os.environ.get('KUBECONFIG'))
        return client.CoreV1Api()

    def check_pod_status(self, ns):
        pods = self.cluster.list_namespaced_pod(ns)

        for pod in pods.items:
            logger.info("%s\t%s\t%s", pod.metadata.name, pod.metadata.namespace, pod.status.phase)
            if pod.status.phase != 'Running':
                kubectl("describe po %s -n %s" % (pod.metadata.name, pod.metadata.namespace))
            assert pod.status.phase == 'Running'

    def create_namespace(self, ns_name):
        self.cluster.create_namespace(client.V1Namespace(metadata=client.V1ObjectMeta(name=ns_name)))

    def deploy(self, image, name, ns, port, replicas=1, svc_type="NodePort", traffic_policy="Local", cluster_ip=None, ipv6=False):
        """
        Creates a deployment and corresponding service with the given
        parameters.
        """
        # Run a deployment with <replicas> copies of <image>, with the
        # pods labelled with "app": <name>.
        deployment = client.V1Deployment(
            api_version="apps/v1",
            kind="Deployment",
            metadata=client.V1ObjectMeta(name=name),
            spec=client.V1DeploymentSpec(
                replicas=replicas,
                selector={'matchLabels': {'app': name}},
                template=client.V1PodTemplateSpec(
                    metadata=client.V1ObjectMeta(labels={"app": name}),
                    spec=client.V1PodSpec(containers=[
                        client.V1Container(name=name,
                                           image=image,
                                           ports=[client.V1ContainerPort(container_port=port)]),
                    ]))))
        api_response = client.AppsV1Api().create_namespaced_deployment(
            body=deployment,
            namespace=ns)
        logger.debug("Deployment created. status='%s'" % str(api_response.status))

        # Create a service called <name> whose endpoints are the pods
        # with "app": <name>; i.e. those just created above.
        self.create_service(name, name, ns, port, svc_type, traffic_policy, ipv6=ipv6)

    def wait_for_deployment(self, name, ns):
        """
        Waits for the given deployment to have the desired number of replicas.
        """
        logger.info("Checking status for deployment %s/%s" % (ns, name))
        kubectl("-n %s rollout status deployment/%s" % (ns, name))
        kubectl("get pods -n %s -o wide" % ns)

    def create_service(self, name, app, ns, port, svc_type="NodePort", traffic_policy="Local", cluster_ip=None, ipv6=False):
        service = client.V1Service(
            metadata=client.V1ObjectMeta(
                name=name,
                labels={"name": name},
            ),
            spec={
                "ports": [{"port": port}],
                "selector": {"app": app},
                "type": svc_type,
            }
        )
        if traffic_policy:
            service.spec["externalTrafficPolicy"] = traffic_policy
        if cluster_ip:
          service.spec["clusterIP"] = cluster_ip
        if ipv6:
          service.spec["ipFamily"] = "IPv6"

        api_response = self.cluster.create_namespaced_service(
            body=service,
            namespace=ns,
        )
        logger.debug("service created, status='%s'" % str(api_response.status))

    def wait_until_exists(self, name, resource_type, ns="default"):
        retry_until_success(kubectl, function_args=["get %s %s -n%s" %
                                                    (resource_type, name, ns)])

    def delete_and_confirm(self, name, resource_type, ns="default"):
        try:
            kubectl("delete %s %s -n%s" % (resource_type, name, ns))
        except subprocess.CalledProcessError:
            pass

        def is_it_gone_yet(res_name, res_type):
            try:
                kubectl("get %s %s -n%s" % (res_type, res_name, ns),
                        logerr=False)
                raise self.StillThere
            except subprocess.CalledProcessError:
                # Success
                pass

        retry_until_success(is_it_gone_yet, retries=10, wait_time=10, function_args=[name, resource_type])

    class StillThere(Exception):
        pass

    def get_routes(self):
        return run("docker exec kube-node-extra ip r")

    def annotate_resource(self, res_type, res_name, ns, k, v):
        return run("kubectl annotate %s %s -n %s %s=%s" % (res_type, res_name, ns, k, v)).strip()

    def get_node_ips_with_local_pods(self, ns, label_selector):
        config.load_kube_config(os.environ.get('KUBECONFIG'))
        api = client.CoreV1Api(client.ApiClient())
        pods = api.list_namespaced_pod(ns, label_selector=label_selector)
        node_names = map(lambda x: x.spec.node_name, pods.items)
        node_ips = []
        for n in node_names:
            addrs = api.read_node(n).status.addresses
            for a in addrs:
                if a.type == 'InternalIP':
                    node_ips.append(a.address)
        return node_ips

    def update_ds_env(self, ds, ns, env_vars):
        config.load_kube_config(os.environ.get('KUBECONFIG'))
        api = client.AppsV1Api(client.ApiClient())
        node_ds = api.read_namespaced_daemon_set(ds, ns, exact=True, export=True)
        for container in node_ds.spec.template.spec.containers:
            if container.name == ds:
                for k, v in env_vars.items():
                    logger.info("Set %s=%s", k, v)
                    env_present = False
                    for env in container.env:
                        if env.name == k:
                            env_present = True
                    if not env_present:
                        v1_ev = client.V1EnvVar(name=k, value=v, value_from=None)
                        container.env.append(v1_ev)
        api.replace_namespaced_daemon_set(ds, ns, node_ds)

        # Wait until the DaemonSet reports that all nodes have been updated.
        while True:
            time.sleep(10)
            node_ds = api.read_namespaced_daemon_set_status("calico-node", "kube-system")
            logger.info("%d/%d nodes updated",
                      node_ds.status.updated_number_scheduled,
                      node_ds.status.desired_number_scheduled)
            if node_ds.status.updated_number_scheduled == node_ds.status.desired_number_scheduled:
                break

    def scale_deployment(self, deployment, ns, replicas):
        return kubectl("scale deployment %s -n %s --replicas %s" %
                       (deployment, ns, replicas)).strip()

    def create_namespace(self, ns):
        kubectl("create ns " + ns)
        self.add_cleanup(lambda: self.delete_and_confirm(ns, "ns"))


# Default is for K8ST tests to run only in the dual_stack rig by
# default.  Individual test classes can override this.
TestBase.vanilla = False
TestBase.dual_stack = True
TestBase.dual_tor = False


class Container(object):

    def __init__(self, image, args, flags=""):
        self.id = run("docker run --rm -d %s %s %s" % (flags, image, args)).strip().split("\n")[-1].strip()
        self._ip = None

    def kill(self):
        run("docker rm -f %s" % self.id)

    def inspect(self, template):
        return run("docker inspect -f '%s' %s" % (template, self.id))

    def running(self):
        return self.inspect("{{.State.Running}}").strip()

    def assert_running(self):
        assert self.running() == "true"

    def wait_running(self):
        retry_until_success(self.assert_running)

    @property
    def ip(self):
        if not self._ip:
            self._ip = self.inspect(
                "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}"
            ).strip()
        return self._ip

    def logs(self):
        return run("docker logs %s" % self.id)

    def execute(self, cmd):
        return run("docker exec %s %s" % (self.id, cmd))


class Pod(object):

    def __init__(self, ns, name, image=None, annotations=None, yaml=None):
        if yaml:
            # Caller has provided the complete pod YAML.
            kubectl("""apply -f - <<'EOF'
%s
EOF
""" % yaml)
        elif annotations:
            # Caller wants specified image, plus annotations.
            pod = {
                "apiVersion": "v1",
                "kind": "Pod",
                "metadata": {
                    "annotations": annotations,
                    "name": name,
                    "namespace": ns,
                },
                "spec": {
                    "containers": [
                        {
                            "name": name,
                            "image": image,
                        },
                    ],
                    "terminationGracePeriodSeconds": 0,
                },
            }
            kubectl("""apply -f - <<'EOF'
%s
EOF
""" % json.dumps(pod))
        else:
            # Caller just wants the specified image.
            kubectl("run %s -n %s --generator=run-pod/v1 --image=%s" % (name, ns, image))
        self.name = name
        self.ns = ns
        self._ip = None

    def delete(self):
        kubectl("delete pod/%s -n %s" % (self.name, self.ns))

    def wait_ready(self):
        kubectl("wait --for=condition=ready pod/%s -n %s --timeout=300s" % (self.name, self.ns))

    @property
    def ip(self):
        if not self._ip:
            self._ip = run("kubectl get po %s -n %s -o json | jq '.status.podIP'" %
                           (self.name, self.ns)).strip().strip('"')
        return self._ip

    def execute(self, cmd):
        return kubectl("exec %s -n %s -- %s" % (self.name, self.ns, cmd))


class TestBaseV6(TestBase):

    def get_routes(self):
        return run("docker exec kube-node-extra ip -6 r")
