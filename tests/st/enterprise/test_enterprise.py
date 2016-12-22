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
import logging
import netaddr
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

POLICY_DIR = "/calico/v1/policy/tier"
# Canned policies, all of which should apply to all the computes.  We use
# a variety of selectors just to hit different code paths.
POLICY_ALLOW_CALICO_TRAFFIC_COMPUTE = {
    "selector": "role == 'compute'",
    "order": 0,
    "inbound_rules": [
        {"protocol": "tcp", "dst_ports": [
            179,    # BGP
            2379,   # etcd
            2380,   # etcd
            4001,   # etcd
            5672],  # AMQP
         "action": "allow",
         "dst_selector": "role == 'compute' || role == 'control'"},
        # Otherwise defer to next tier.
        {"action": "next-tier"},
    ],
    "outbound_rules": [
        {"protocol": "tcp", "dst_ports": [
            80],
         "action": "allow",
         "dst_selector": "role == 'compute' || role == 'control'"},
        {"action": "next-tier"},
    ]
}
POLICY_ALLOW_CALICO_TRAFFIC_CONTROL = {
    "selector": "role == 'control'",
    "order": 0,
    "inbound_rules": [
        {"protocol": "tcp",
         "dst_ports": [
            80,     # Horizon UI
            443,    # Horizon UI
            2379,   # etcd
            2380,   # etcd
            4001,   # etcd
            5672,   # AMQP
            5000,   # keystone
            8773,   # nova
            8774,   # nova
            8775,   # nova
            9191,   # glance registry
            9292,   # glance api
            9696,   # neutron
            35357   # keystone (openstack api)
            ],
         "action": "allow",
         "dst_selector": "role == 'control'",
         },
        {"protocol": "udp", "action": "allow"},
        # Otherwise defer to next tier.
        {"action": "next-tier"},
    ],
    "outbound_rules": [
        {"protocol": "tcp",
         "dst_ports": [
            2379,    # etcd
            2380,    # etcd
            4001,    # etcd
            5672],   # AMQP
         "action": "allow",
         "dst_selector": "role == 'compute' || role == 'control'",
         },
        {"protocol": "udp", "action": "allow"},
        {"action": "next-tier"},
    ]
}
POLICY_ALLOW_ALL = {
    "selector": "role == 'compute'",
    "order": 10,
    "inbound_rules": [
        {"action": "allow"}
    ],
    "outbound_rules": [
        {"action": "allow"}
    ]
}
POLICY_DENY_ALL = {
    "selector": "role != 'control'",
    "order": 10,
    "inbound_rules": [
        {"action": "deny"}
    ],
    "outbound_rules": [
        {"action": "deny"}
    ]
}
POLICY_NEXT_ALL = {
    "selector": "role != 'control' && role == 'compute'",
    "order": 10,
    "inbound_rules": [
        {"action": "next-tier"}
    ],
    "outbound_rules": [
        {"action": "next-tier"}
    ]
}
POLICY_NONE_ALL = {
    "selector": "all()",
    "order": 10,
    "inbound_rules": [
    ],
    "outbound_rules": [
    ]
}

@staticmethod
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

class MultiHostIpfix(TestBase):
    @classmethod
    def setUpClass(cls):
        super(TestBase, cls).setUpClass()
        num_hosts = 2
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
            host.remove_workloads()
        for network in cls.networks:
            network.delete()
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


