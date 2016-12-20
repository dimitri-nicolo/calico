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
            (n1_workloads, n2_workloads, networks) = \
                self._setup_workloads(host1, host2)

            # Get the original profiles:
            output = host1.calicoctl("get profile -o yaml")
            original_profiles = yaml.safe_load(output)
            # Make a copy of the profiles to mess about with.
            new_profiles = copy.deepcopy(original_profiles)

            profile0_tag = new_profiles[0]['metadata']['tags'][0]
            profile1_tag = new_profiles[1]['metadata']['tags'][0]
            rule0 = {'action': 'allow',
                     'source':
                         {'tag': profile1_tag}}
            rule1 = {'action': 'allow',
                     'source':
                         {'tag': profile0_tag}}
            new_profiles[0]['spec']['ingress'].append(rule0)
            new_profiles[1]['spec']['ingress'].append(rule1)
            self._apply_new_profile(new_profiles, host1)
            # Check everything can contact everything else now
            self.assert_connectivity(retries=2,
                                     pass_list=n1_workloads + n2_workloads)

            # Now restore the original profile and check it all works as before
            self._apply_new_profile(original_profiles, host1)
            host1.calicoctl("get profile -o yaml")
            self._check_original_connectivity(n1_workloads, n2_workloads)

            # Tidy up
            host1.remove_workloads()
            host2.remove_workloads()
            for network in networks:
                network.delete()

    @staticmethod
    def _get_profiles(profiles):
        """
        Sorts and returns the profiles for the networks.
        :param profiles: the list of profiles
        :return: tuple: profile for network1, profile for network2
        """
        prof_n1 = None
        prof_n2 = None
        for profile in profiles:
            if profile['metadata']['name'] == "testnet1":
                prof_n1 = profile
            elif profile['metadata']['name'] == "testnet2":
                prof_n2 = profile
        assert prof_n1 is not None, "Could not find testnet1 profile"
        assert prof_n2 is not None, "Could not find testnet2 profile"
        return prof_n1, prof_n2

    @staticmethod
    def _apply_new_profile(new_profile, host):
        # Apply new profiles
        host.writefile("new_profiles",
                       yaml.dump(new_profile, default_flow_style=False))
        host.calicoctl("apply -f new_profiles")

    def _setup_workloads(self, host1, host2):
        # TODO work IPv6 into this test too
        host1.start_calico_node()
        host2.start_calico_node()

        # Create the networks on host1, but it should be usable from all
        # hosts.  We create one network using the default driver, and the
        # other using the Calico driver.
        network1 = host1.create_network("testnet1")
        network2 = host1.create_network("testnet2")
        networks = [network1, network2]

        # Assert that the networks can be seen on host2
        assert_network(host2, network2)
        assert_network(host2, network1)

        n1_workloads = []
        n2_workloads = []

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

        # Create similar workloads in network 2.
        n2_workloads.append(host1.create_workload("workload_h1n2_1",
                                                  image="workload",
                                                  network=network2))
        n2_workloads.append(host1.create_workload("workload_h1n2_2",
                                                  image="workload",
                                                  network=network2))
        n2_workloads.append(host2.create_workload("workload_h2n2_1",
                                                  image="workload",
                                                  network=network2))
        print "*******************"
        print "Network1 is:\n%s\n%s" % (
            [x.ip for x in n1_workloads],
            [x.name for x in n1_workloads])
        print "Network2 is:\n%s\n%s" % (
            [x.ip for x in n2_workloads],
            [x.name for x in n2_workloads])
        print "*******************"

        # Assert that endpoints are in Calico
        assert_number_endpoints(host1, 4)
        assert_number_endpoints(host2, 2)

        self._check_original_connectivity(n1_workloads, n2_workloads)

        # Test deleting the network. It will fail if there are any
        # endpoints connected still.
        self.assertRaises(CommandExecError, network1.delete)
        self.assertRaises(CommandExecError, network2.delete)

        return n1_workloads, n2_workloads, networks

    def _check_original_connectivity(self, n1_workloads, n2_workloads,
                                     types=None):
        # Assert that workloads can communicate with each other on network
        # 1, and not those on network 2.  Ping using IP for all workloads,
        # and by hostname for workloads on the same network (note that
        # a workloads own hostname does not work).
        if types is None:
            types = ['icmp', 'tcp', 'udp']
        self.assert_connectivity(retries=2,
                                 pass_list=n1_workloads,
                                 fail_list=n2_workloads,
                                 type_list=types)

        # Repeat with network 2.
        self.assert_connectivity(pass_list=n2_workloads,
                                 fail_list=n1_workloads,
                                 type_list=types)
