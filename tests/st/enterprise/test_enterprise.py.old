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
import netaddr
import yaml
from nose_parameterized import parameterized

from tests.st.test_base import TestBase
from tests.st.enterprise.utils.ipfix_monitor import IpfixFlow, IpfixMonitor
from tests.st.utils.docker_host import DockerHost
from tests.st.utils.exceptions import CommandExecError
from tests.st.utils.utils import assert_network, assert_profile, \
    assert_number_endpoints, get_profile_name, ETCD_CA, ETCD_CERT, \
    ETCD_KEY, ETCD_HOSTNAME_SSL, ETCD_SCHEME, get_ip

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
    def test_multi_host(self):
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
        with DockerHost("host1",
                        additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                        post_docker_commands=POST_DOCKER_COMMANDS,
                        start_calico=False) as host1, \
                DockerHost("host2",
                           additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                           post_docker_commands=POST_DOCKER_COMMANDS,
                           start_calico=False) as host2:
            (n1_workloads, network) = self._setup_workloads(host1, host2)

            # Start monitoring flows.  When writing tests be aware that flows will continue to be reported
            # for a short while after they the actual packets stop.
            mon = IpfixMonitor(get_ip(), 4739)

            # Send a single ping packet between two workloads, and ensure it's reported in the ipfix output.
            # The flow is recorded at both ends, so it appears twice.
            print "==== Checking ping reported by ipfix ===="
            n1_workloads[0].check_can_ping(n1_workloads[1].ip, retries=0)
            ping_flows = [IpfixFlow(n1_workloads[0].ip, n1_workloads[1].ip, packets="1,1", octets="84,84"),
                          IpfixFlow(n1_workloads[1].ip, n1_workloads[0].ip, packets="1,1", octets="84,84")]
            mon.assert_flows_present(ping_flows, 10, allow_others=False)

            host1.remove_workloads()
            host2.remove_workloads()
            network.delete()

    def _setup_workloads(self, host1, host2):
        # Configure the address of the ipfix collector.
        host1.calicoctl("config set IpfixCollectorAddr " + get_ip() + " --raw=felix")
        # Disappointingly, tshark only appears to be able to decode IPFIX when the UDP port is 4739.
        host1.calicoctl("config set IpfixCollectorPort 4739 --raw=felix")

        host1.start_calico_node()
        host2.start_calico_node()

        network1 = host1.create_network("testnet1")
        assert_network(host2, network1)

        n1_workloads = []

        # Create two workloads on host1 and one on host2 all in network 1.
        n1_workloads.append(host2.create_workload("workload_h2n1_1",
                                                  image="workload",
                                                  network=network1))
        n1_workloads.append(host1.create_workload("workload_h1n1_1",
                                                  image="workload",
                                                  network=network1))
        n1_workloads.append(host1.create_workload("workload_h1n1_2",
                                                  image="workload",
                                                  network=network1))

        # Assert that endpoints are in Calico
        assert_number_endpoints(host1, 2)
        assert_number_endpoints(host2, 1)

        # Test deleting the network. It will fail if there are any
        # endpoints connected still.
        self.assertRaises(CommandExecError, network1.delete)

        return n1_workloads, network1
