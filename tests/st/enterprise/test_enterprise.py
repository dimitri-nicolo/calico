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


class MultiHostIpfix(TestBase):
    @classmethod
    def setUpClass(cls):
        super(TestBase, cls).setUpClass()
        num_hosts = 2
        cls.hosts = []
        makehost = functools.partial(DockerHost,
                                     additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                                     post_docker_commands=POST_DOCKER_COMMANDS,
                                     start_calico=True)

        hostnames = []
        for i in range(num_hosts):
            hostnames.append("host%s" % i)
        pool = Pool(num_hosts)
        cls.hosts = pool.map(makehost, hostnames)
        pool.close()
        pool.join()

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

        # Start monitoring flows.  When writing tests be aware that flows will continue
        # to be reported for a short while after they the actual packets stop.
        cls.mon = IpfixMonitor(get_ip(), 4739)

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
        # Reset flows before every test
        self.mon.reset_flows()

    def test_dummy(self):
        import pdb; pdb.set_trace()

    @parameterized.expand(["1", "2", "3", "4", "5"])
    def test_multi_host(self, iteration):
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
        self.mon.assert_flows_present(ping_flows, 10, allow_others=False)
