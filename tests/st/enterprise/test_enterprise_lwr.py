# Copyright 2016 Tigera, Inc
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
import copy
import functools
import json
import logging
import netaddr
import subprocess
import time
import unittest
import yaml
from nose_parameterized import parameterized
from multiprocessing.dummy import Pool

from tests.st.test_base import TestBase
from tests.st.enterprise.utils.ipfix_monitor import IpfixFlow, IpfixMonitor
from tests.st.utils.docker_host import DockerHost
from tests.st.utils.exceptions import CommandExecError
from tests.st.utils.utils import assert_network, assert_profile, \
    assert_number_endpoints, get_profile_name, ETCD_CA, ETCD_CERT, \
    ETCD_KEY, ETCD_HOSTNAME_SSL, ETCD_SCHEME, get_ip

_log = logging.getLogger(__name__)
_log.setLevel(logging.DEBUG)

POST_DOCKER_COMMANDS = ["docker load -i /code/calico-node.tar",
                        "docker load -i /code/busybox.tar",
                        "docker load -i /code/workload.tar"]

if ETCD_SCHEME == "https":
    ADDITIONAL_DOCKER_OPTIONS = "--cluster-store=etcd://%s:2379 " \
                                "--cluster-store-opt kv.cacertfile=%s " \
                                "--cluster-store-opt kv.certfile=%s " \
                                "--cluster-store-opt kv.keyfile=%s " % \
                                (ETCD_HOSTNAME_SSL, ETCD_CA, ETCD_CERT,
                                 ETCD_KEY)
else:
    ADDITIONAL_DOCKER_OPTIONS = "--cluster-store=etcd://%s:2379 " % \
                                get_ip()


def parallel_host_setup(num_hosts):
    makehost = functools.partial(DockerHost,
                                 additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                                 post_docker_commands=POST_DOCKER_COMMANDS,
                                 start_calico=False)
    hostnames = []
    for i in range(num_hosts):
        hostnames.append("host%s" % i)
    pool = Pool(num_hosts)
    hosts = pool.map(makehost, hostnames)
    pool.close()
    pool.join()
    return hosts


def wipe_etcd():
    _log.debug("Wiping etcd")
    # Delete /calico if it exists. This ensures each test has an empty data
    # store at start of day.
    curl_etcd(get_ip(), "calico", options=["-XDELETE"])

    # Disable Usage Reporting to usage.projectcalico.org
    # We want to avoid polluting analytics data with unit test noise
    curl_etcd(get_ip(),
              "calico/v1/config/UsageReportingEnabled",
              options=["-XPUT -d value=False"])
    curl_etcd(get_ip(),
              "calico/v1/config/LogSeverityScreen",
              options=["-XPUT -d value=debug"])


def curl_etcd(ip, path, options=None, recursive=True):
    """
    Perform a curl to etcd, returning JSON decoded response.
    :param ip: IP address of etcd server
    :param path:  The key path to query
    :param options:  Additional options to include in the curl
    :param recursive:  Whether we want recursive query or not
    :return:  The JSON decoded response.
    """
    if options is None:
        options = []
    if ETCD_SCHEME == "https":
        # Etcd is running with SSL/TLS, require key/certificates
        command = "curl --cacert %s --cert %s --key %s " \
                  "-sL https://%s:2379/v2/keys/%s?recursive=%s %s" % \
                  (ETCD_CA, ETCD_CERT, ETCD_KEY, ETCD_HOSTNAME_SSL, path,
                   str(recursive).lower(), " ".join(options))
    else:
        command = "curl -sL http://%s:2379/v2/keys/%s?recursive=%s %s" % \
                  (ip, path, str(recursive).lower(), " ".join(options))
    _log.debug("Running: %s", command)
    rc = subprocess.check_output(command, shell=True)
    return json.loads(rc.strip())


class MultiHostIpfix(TestBase):
    @classmethod
    def setUpClass(cls):
        super(TestBase, cls).setUpClass()
        cls.hosts = []
        cls.hosts.append(DockerHost("host1",
                                    additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                                    post_docker_commands=POST_DOCKER_COMMANDS,
                                    start_calico=False))
        cls.hosts.append(DockerHost("host2",
                                    additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                                    post_docker_commands=POST_DOCKER_COMMANDS,
                                    start_calico=False))
        for host in cls.hosts:
            host.start_calico_node()
        # Allow time for calico-node to load
        time.sleep(10)

        # Configure the address of the ipfix collector.
        cls.hosts[0].calicoctl("config set IpfixCollectorAddr " + get_ip() + " --raw=felix")
        # Disappointingly, tshark only appears to be able to decode IPFIX when the UDP port is 4739.
        cls.hosts[0].calicoctl("config set IpfixCollectorPort 4739 --raw=felix")

        cls.networks = []
        cls.networks.append(cls.hosts[0].create_network("testnet1"))

        cls.n1_workloads = []
        # Create two workloads on cls.hosts[0] and one on cls.hosts[1] all in network 1.
        cls.n1_workloads.append(cls.hosts[1].create_workload("workload_h2n1_1",
                                                             image="workload",
                                                             network=cls.networks[0]))
        cls.n1_workloads.append(cls.hosts[0].create_workload("workload_h1n1_1",
                                                             image="workload",
                                                             network=cls.networks[0]))
        cls.n1_workloads.append(cls.hosts[0].create_workload("workload_h1n1_2",
                                                             image="workload",
                                                             network=cls.networks[0]))
        # Assert that endpoints are in Calico
        assert_number_endpoints(cls.hosts[0], 2)
        assert_number_endpoints(cls.hosts[1], 1)

    @classmethod
    def tearDownClass(cls):
        # Tidy up
        for host in cls.hosts:
            host.cleanup()
            del host

    def setUp(self):
        # Start monitoring flows.  When writing tests be aware that flows will continue
        # to be reported for a short while after they the actual packets stop.
        _log.debug("Create flow monitor for each test")
        self.mon = IpfixMonitor(get_ip(), 4739)

    def tearDown(self):
        # Reset flows after every test
        _log.debug("Deleting flow monitor after test")
        del self.mon

    def test_multi_host_ping(self, iteration=1):
        """
        Run a mainline multi-host test with IPFIX.
        Because multihost tests are slow to setup, this tests most mainline
        functionality in a single test.
        - Create two hosts
        - Create a network using the Calico IPAM driver, and a workload on
          each host assigned to that network.
        - Check that hosts on the same network can ping each other.
        - Check that hosts on different networks cannot ping each other.
        - Modify the profile rules
        - Check that connectivity has changed to match the profile we set up
        - Re-apply the original profile
        - Check that connectivity goes back to what it was originally.
        """
        _log.debug("*"*80)
        _log.debug("Iteration %s", iteration)

        # Send a single ping packet between two workloads, and ensure it's reported in the
        # ipfix output. The flow is recorded at both ends, so it appears twice.
        print "==== Checking ping reported by ipfix ===="
        self.n1_workloads[0].check_can_ping(self.n1_workloads[1].ip, retries=0)
        ping_flows = [IpfixFlow(self.n1_workloads[0].ip,
                                self.n1_workloads[1].ip,
                                packets="1,1",
                                octets="84,84"),
                      IpfixFlow(self.n1_workloads[1].ip,
                                self.n1_workloads[0].ip,
                                packets="1,1",
                                octets="84,84")]
        self.mon.assert_flows_present(ping_flows, 20, allow_others=False)

    def test_multi_host_tcp(self, iteration=1):
        """
        Run a mainline multi-host test with IPFIX.
        Because multihost tests are slow to setup, this tests most mainline
        functionality in a single test.
        - Create two hosts
        - Create a network using the Calico IPAM driver, and a workload on
          each host assigned to that network.
        - Check that hosts on the same network can ping each other.
        - Check that hosts on different networks cannot ping each other.
        - Modify the profile rules
        - Check that connectivity has changed to match the profile we set up
        - Re-apply the original profile
        - Check that connectivity goes back to what it was originally.
        """
        _log.debug("*"*80)
        _log.debug("Iteration %s", iteration)

        # Send a single ping packet between two workloads, and ensure it's reported in the
        # ipfix output. The flow is recorded at both ends, so it appears twice.
        print "==== Checking tcp reported by ipfix ===="
        self.n1_workloads[0].check_can_tcp(self.n1_workloads[1].ip, retries=0)
        ping_flows = [IpfixFlow(self.n1_workloads[0].ip,
                                self.n1_workloads[1].ip,
                                packets="4,6",
                                octets="222,326"),
                      IpfixFlow(self.n1_workloads[1].ip,
                                self.n1_workloads[0].ip,
                                packets="6,4",
                                octets="326,222")]
        self.mon.assert_flows_present(ping_flows, 20, allow_others=False)

    def test_multi_host_udp(self, iteration=1):
        """
        Run a mainline multi-host test with IPFIX.
        Because multihost tests are slow to setup, this tests most mainline
        functionality in a single test.
        - Create two hosts
        - Create a network using the Calico IPAM driver, and a workload on
          each host assigned to that network.
        - Check that hosts on the same network can ping each other.
        - Check that hosts on different networks cannot ping each other.
        - Modify the profile rules
        - Check that connectivity has changed to match the profile we set up
        - Re-apply the original profile
        - Check that connectivity goes back to what it was originally.
        """
        _log.debug("*"*80)
        _log.debug("Iteration %s", iteration)

        # Send a single ping packet between two workloads, and ensure it's reported in the
        # ipfix output. The flow is recorded at both ends, so it appears twice.
        print "==== Checking udp reported by ipfix ===="
        self.n1_workloads[0].check_can_udp(self.n1_workloads[1].ip, retries=0)
        ping_flows = [IpfixFlow(self.n1_workloads[0].ip,
                                self.n1_workloads[1].ip,
                                packets="1,1",
                                octets="34,34"),
                      IpfixFlow(self.n1_workloads[1].ip,
                                self.n1_workloads[0].ip,
                                packets="1,1",
                                octets="34,34")]
        self.mon.assert_flows_present(ping_flows, 20, allow_others=False)

policy_next_all = {
    "apiVersion": "v1",
    "kind": "policy",
    "metadata": {"name": "policy_next_all"},
    "spec": {
        "order": 10,
        "ingress": [{"action": "pass"}],
        "egress": [{"action": "pass"}]
    }
}

policy_allow_all = {
    "apiVersion": "v1",
    "kind": "policy",
    "metadata": {"name": "policy_allow_all"},
    "spec": {
        "order": 10,
        "ingress": [{"action": "allow"}],
        "egress": [{"action": "allow"}]
    }
}

policy_deny_all = {
    "apiVersion": "v1",
    "kind": "policy",
    "metadata": {"name": "policy_deny_all"},
    "spec": {
        "order": 10,
        "ingress": [{"action": "deny"}],
        "egress": [{"action": "deny"}]}
}
policy_none_all = {
    "apiVersion": "v1",
    "kind": "policy",
    "metadata": {"name": "policy_none_all"},
    "spec": {
        "selector": "all()",
        "order": 10,
        "ingress": [],
        "egress": []}
}


class TieredPolicyWorkloads(TestBase):
    def setUp(self):
        _log.debug("Override the TestBase setUp() method which wipes etcd. Do nothing.")
        # Wipe policies and tiers before each test
        self.delete_all("pol")
        self.delete_all("tier")

    def delete_all(self, resource):
        # Grab all objects of a resource type
        objects = yaml.load(self.hosts[0].calicoctl("get %s -o yaml" % resource))
        # and delete them (if there are any)
        if len(objects) > 0:
            self._delete_data(objects, self.hosts[0])

    @staticmethod
    def sleep(length):
        _log.debug("Sleeping for %s" % length)
        time.sleep(length)

    @classmethod
    def setUpClass(cls):
        wipe_etcd()
        cls.policy_tier_name = "default"
        cls.next_tier_allowed = False
        cls.hosts = []
        cls.hosts.append(DockerHost("host1",
                                    additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                                    post_docker_commands=POST_DOCKER_COMMANDS,
                                    start_calico=False))
        cls.hosts.append(DockerHost("host2",
                                    additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                                    post_docker_commands=POST_DOCKER_COMMANDS,
                                    start_calico=False))
        for host in cls.hosts:
            host.start_calico_node()
        # Allow time for calico-node to load
        time.sleep(10)

        cls.networks = []
        cls.networks.append(cls.hosts[0].create_network("testnet2"))
        cls.sleep(10)

        cls.n1_workloads = []
        # Create two workloads on cls.hosts[0] and one on cls.hosts[1] all in network 1.
        cls.n1_workloads.append(cls.hosts[1].create_workload("workload_h2n1_1",
                                                             image="workload",
                                                             network=cls.networks[0]))
        cls.sleep(2)
        cls.n1_workloads.append(cls.hosts[0].create_workload("workload_h1n1_1",
                                                             image="workload",
                                                             network=cls.networks[0]))
        # Assert that endpoints are in Calico
        assert_number_endpoints(cls.hosts[0], 1)
        assert_number_endpoints(cls.hosts[1], 1)

    @classmethod
    def tearDownClass(cls):
        # Tidy up
        for host in cls.hosts:
            host.remove_workloads()
        for network in cls.networks:
            network.delete()
        for host in cls.hosts:
            host.cleanup()
            del host

    def test_tier_ordering_explicit(self):
        """Check correct ordering of tiers by their explicit order field."""
        self.policy_tier_name = "the-tier"
        self.next_tier_allowed = True
        self._do_tier_order_test("tier-c", 1,
                                 "tier-b", 2,
                                 "tier-a", 3)

    def test_tier_ordering_implicit(self):
        """Check correct ordering of tiers by name as tie-breaker."""
        self.policy_tier_name = "the-tier"
        self.next_tier_allowed = True
        self._do_tier_order_test("tier-1", 1,
                                 "tier-2", 1,
                                 "tier-3", 1)

    def set_tier(self, name=None, order=10):
        _log.debug("Setting tier data: \n"
                   "name : %s\norder : %s",
                   name, order)
        if name is None:
            name = self.policy_tier_name
        tier = {"apiVersion": "v1",
                "kind": "tier",
                "metadata": {"name": name},
                "spec": {"order": order}
                }
        self._apply_data(tier, self.hosts[0])

    def _do_tier_order_test(self, first_tier, first_tier_order, second_tier,
                            second_tier_order, third_tier, third_tier_order):
        # Note that the following tests need to check that connectivity to
        # the rpcbind service alternates between succeeding and failing so
        # that we spot if felix hasn't actually changed anything.
        # Check we start with connectivity.
        _log.info("Starting tier order test %s (%s); %s (%s); %s (%s)",
                  first_tier, first_tier_order, second_tier,
                  second_tier_order, third_tier, third_tier_order)

        self.assert_connectivity(self.n1_workloads)

        # Create tiers and endpoints.
        _log.info("Configuring tiers.")
        self.set_tier(name=first_tier, order=first_tier_order)
        self.set_tier(name=second_tier, order=second_tier_order)
        self.set_tier(name=third_tier, order=third_tier_order)

        # Slip a deny into the third tier, just to alternate DENY/ALLOW.
        self.set_policy(third_tier, "pol-1", policy_deny_all)
        # Check that access is blocked.
        self.assert_no_connectivity(self.n1_workloads)

        # Allow in first tier only, should allow.
        _log.info("Allow in first tier only, should allow.")
        self.set_policy(first_tier, "pol-1", policy_allow_all)
        self.set_policy(second_tier, "pol-1", policy_deny_all)
        self.set_policy(third_tier, "pol-1", policy_deny_all)
        self.assert_connectivity(self.n1_workloads)

        # Deny in all tiers, should drop.
        _log.info("Deny in all tiers, should drop.")
        # Fix up second tier
        self.set_policy(second_tier, "pol-1", policy_deny_all)
        self.set_policy(first_tier, "pol-1", policy_deny_all)
        self.assert_no_connectivity(self.n1_workloads)

        # Allow in first tier, should allow.
        self.set_policy(first_tier, "pol-1", policy_allow_all)
        self.assert_connectivity(self.n1_workloads)

        # Switch, now the first tier drops but the later ones allow.
        _log.info("Switch, now the first tier drops but the later ones "
                  "allow.")
        self.set_policy(first_tier, "pol-1", policy_deny_all)
        self.set_policy(second_tier, "pol-1", policy_allow_all)
        self.set_policy(third_tier, "pol-1", policy_allow_all)
        self.assert_no_connectivity(self.n1_workloads)

        # Fall through via a next-tier policy in the first tier.
        _log.info("Fall through via a next-tier policy in the first "
                  "tier.")
        self.set_policy(first_tier, "pol-1", policy_next_all)
        self.assert_connectivity(self.n1_workloads)

        # Swap the second tier for a drop.
        _log.info("Swap the second tier for a drop.")
        self.set_policy(second_tier, "pol-1", policy_deny_all)
        self.assert_no_connectivity(self.n1_workloads)

    def _apply_data(self, data, host):
        _log.debug("Applying data with calicoctl: %s", data)
        self._use_calicoctl("apply", data, host)

    def _delete_data(self, data, host):
        _log.debug("Deleting data with calicoctl: %s", data)
        self._use_calicoctl("delete", data, host)

    @staticmethod
    def _use_calicoctl(action, data, host):
        # use calicoctl with data
        host.writefile("new_profile",
                       yaml.dump(data, default_flow_style=False))
        host.calicoctl("%s -f new_profile" % action)

    def set_policy(self, tier, policy_name, data, order=None):
        data = copy.deepcopy(data)
        if order is not None:
            data["spec"]["order"] = order

        if not self.next_tier_allowed:
            for dirn in ["ingress", "egress"]:
                if dirn in data:
                    def f(rule):
                        return rule != {"action": "pass"}
                    data[dirn] = filter(f, data[dirn])

        data["metadata"]["name"] = policy_name
        if tier != "default":
            data["metadata"]["tier"] = tier

        self._apply_data(data, self.hosts[0])

    def assert_no_connectivity(self, workload_list, retries=0, type_list=None):
        """
        Checks that none of the workloads passed in can contact any of the others.
        :param workload_list:
        :param retries:
        :param type_list:
        :return:
        """
        for workload in workload_list:
            the_rest = [wl for wl in workload_list if wl is not workload]
            self.assert_connectivity([workload], fail_list=the_rest,
                                     retries=retries, type_list=type_list)

    def test_policy_ordering_explicit(self):
        """Check correct ordering of policies by their explicit order
        field."""
        self.policy_tier_name = "default"
        self.next_tier_allowed = False
        self._do_policy_order_test("pol-c", 1,
                                   "pol-b", 2,
                                   "pol-a", 3)

    def test_policy_ordering_implicit(self):
        """Check correct ordering of policies by name as tie-breaker."""
        self.policy_tier_name = "default"
        self.next_tier_allowed = False
        self._do_policy_order_test("pol-1", 1,
                                   "pol-2", 1,
                                   "pol-3", 1)

    def _do_policy_order_test(self,
                              first_pol, first_pol_order,
                              second_pol, second_pol_order,
                              third_pol, third_pol_order):
        """Checks that policies are ordered correctly."""
        # Note that the following tests need to check that connectivity to
        # the rpcbind service alternates between succeeding and failing so
        # that we spot if felix hasn't actually changed anything.

        _log.info("Check we start with connectivity.")
        self.assert_connectivity(self.n1_workloads)
        _log.info("Apply a single deny policy")
        self.set_policy(self.policy_tier_name, first_pol, policy_deny_all,
                        order=first_pol_order)
        _log.info("Check we now cannot access tcp service")
        self.assert_no_connectivity(self.n1_workloads)
        _log.info("Allow in first tier only, should allow.")
        self.set_policy(self.policy_tier_name, first_pol, policy_allow_all,
                        order=first_pol_order)
        self.set_policy(self.policy_tier_name, second_pol, policy_deny_all,
                        order=second_pol_order)
        self.set_policy(self.policy_tier_name, third_pol, policy_deny_all,
                        order=third_pol_order)
        self.assert_connectivity(self.n1_workloads)

        # Fix up second tier
        self.set_policy(self.policy_tier_name, second_pol, policy_deny_all,
                        order=second_pol_order)

        # Deny in all tiers, should drop.
        _log.info("Deny in all tiers, should drop.")
        self.set_policy(self.policy_tier_name, first_pol, policy_deny_all,
                        order=first_pol_order)
        self.assert_no_connectivity(self.n1_workloads)

        # Allow in first tier, should allow.
        _log.info("Allow in first tier, should allow.")
        self.set_policy(self.policy_tier_name, first_pol, policy_allow_all,
                        order=first_pol_order)
        self.assert_connectivity(self.n1_workloads)

        # Switch, now the first policy drops but the later ones allow.
        _log.info("Switch, now the first tier drops but the later ones "
                  "allow.")
        self.set_policy(self.policy_tier_name, first_pol, policy_deny_all,
                        order=first_pol_order)
        self.set_policy(self.policy_tier_name, second_pol, policy_allow_all,
                        order=second_pol_order)
        self.set_policy(self.policy_tier_name, third_pol, policy_allow_all,
                        order=third_pol_order)
        self.assert_no_connectivity(self.n1_workloads)

        # Fall through to second policy.
        _log.info("Fall through to second policy.")
        self.set_policy(self.policy_tier_name, first_pol, policy_none_all,
                        order=first_pol_order)
        self.assert_connectivity(self.n1_workloads)

        # Swap the second tier for a drop.
        _log.info("Swap the second tier for a drop.")
        self.set_policy(self.policy_tier_name, second_pol,  policy_deny_all,
                        order=second_pol_order)
        self.assert_no_connectivity(self.n1_workloads)


class IpNotFound(Exception):
    pass
