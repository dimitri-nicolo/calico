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
import json
import logging
import subprocess

from netaddr import IPAddress, IPNetwork
from nose_parameterized import parameterized
from tests.st.test_base import TestBase
from tests.st.utils.docker_host import DockerHost
from tests.st.utils.utils import assert_network, assert_number_endpoints, ETCD_CA, ETCD_CERT, \
    ETCD_KEY, ETCD_HOSTNAME_SSL, ETCD_SCHEME, get_ip

from tests.st.enterprise.utils.ipfix_monitor import IpfixFlow, IpfixMonitor

_log = logging.getLogger(__name__)
_log.setLevel(logging.DEBUG)

POST_DOCKER_COMMANDS = ["docker load -i /code/calico-node.tar",
                        "docker load -i /code/workload.tar",
                        "apk add --update conntrack-tools"]

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


nat_outgoing_ip = IPAddress("8.8.8.8")
default_ipv4_pool = IPNetwork("192.168.0.0/16")
ippool_cidr = "ippool " + str(default_ipv4_pool)

test_default = {
    'apiVersion': 'v1',
    'kind': 'ipPool',
    'metadata': {
        'cidr': str(default_ipv4_pool)
    },
    'spec': {}
}

test_ipip = {
    'apiVersion': 'v1',
    'kind': 'ipPool',
    'metadata': {
        'cidr': str(default_ipv4_pool)
    },
    'spec': {
        'ipip': {
            'enabled': bool(True)
        }
    }
}

test_nat_out = {
    'apiVersion': 'v1',
    'kind': 'ipPool',
    'metadata': {
        'cidr': str(default_ipv4_pool)
    },
    'spec': {
        'nat-outgoing': bool(True)
    }
}

test_ipip_nat_out = {
    'apiVersion': 'v1',
    'kind': 'ipPool',
    'metadata': {
        'cidr': str(default_ipv4_pool)
    },
    'spec': {
        'ipip': {
            'enabled': bool(True)
        },
        'nat-outgoing': bool(True)
    }
}


class MultiHostIpfixAsymTCP(TestBase):
    @classmethod
    def setUpClass(cls):
        # Wipe etcd before setting up a new test rig for this class.
        wipe_etcd()

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
            _log.debug("This is a host: %s", host.name)
            host.start_calico_node()

        # Configure the address of the ipfix collector.
        cls.hosts[0].calicoctl("config set IpfixCollectorAddr " + get_ip() + " --raw=felix")
        # Disappointingly, tshark only appears to be able to decode IPFIX when the UDP port is 4739.
        cls.hosts[0].calicoctl("config set IpfixCollectorPort 4739 --raw=felix")

        cls.networks = []
        _log.debug("This is the cls.networks: %s", cls.networks)
        cls.networks.append(cls.hosts[0].create_network("testnet1"))

        _log.debug(cls.hosts[0].name)
        _log.debug(cls.hosts[1].name)
        _log.debug(cls.networks[0])

        # Assert that network is created. 
        assert_network(cls.hosts[0], cls.networks[0])

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
        # This method is REQUIRED or else the TestBase setUp() method will wipe etcd before
        # every test.
        _log.debug("Override the TestBase setUp() method which wipes etcd.")

        # Start monitoring flows.  When writing tests be aware that flows will continue
        # to be reported for a short while after they the actual packets stop.
        _log.debug("Create flow monitor for each test")
        self.mon = IpfixMonitor(get_ip(), 4739)

    def tearDown(self):
        # Reset flows after every test
        _log.debug("Deleting flow monitor after test")
        del self.mon

    @parameterized.expand(["0", "1"])
    def test_multi_host_tcp_asym(self, ipip):
        """
        Run a mainline multi-host test with IPFIX.
        Because multihost tests are slow to setup, this tests most mainline
        functionality in a single test.
        - Create two hosts
        - Create a network using the Calico IPAM driver, and a workload on
          each host assigned to that network.
        - Check that hosts on the same network can ping each other.
        - Check that hosts on different networks cannot ping each other.
        - Run the asymmetric TCP packets testcase with default ippool enabled.
        - Check that a asymmetric TCP packets are reported correctly by ipfix
        - Delete default ippool.
        - Run the asymmetric TCP packets testcase with ipip ippool enabled.
        - Check that a asymmetric TCP packets do *not* include IPIP headers in ipfix.
        """
        _log.debug("*" * 80)
        _log.debug("ipip %s", int(ipip))

        _log.debug("==== ippool configuration ====")
        self.hosts[0].calicoctl("get ippool -o json")

        _log.debug("==== Setting ippool configuration to - default ====")
        self.hosts[0].writefile("test_default.yaml", test_default)
        self.hosts[0].calicoctl("%s -f test_default.yaml" % "apply")

        if not int(ipip):
            _log.debug("==== Validate that the ippool configuration for default has been applied ====")
            self.check_data_in_datastore(self.hosts[0], [test_default], ippool_cidr)

        else:
            _log.debug("==== Delete default ippool configuration ====")
            self.hosts[0].calicoctl("%s -f test_default.yaml" % "delete")

            _log.debug("==== Changing ippool configuration to - ipip enabled ====")
            self.hosts[0].writefile("test_ipip.yaml", test_ipip)
            self.hosts[0].calicoctl("%s -f test_ipip.yaml" % "apply")

            _log.debug("==== Validate that the ippool configuration for ipip has been applied ====")
            self.check_data_in_datastore(self.hosts[0], [test_ipip], ippool_cidr)

        _log.debug("==== Run the asymmetric TCP packets testcase ====")
        self.n1_workloads[0].check_can_tcp_asym(self.n1_workloads[1].ip, retries=0)

        # ipfix output. The flow is recorded at both ends, so it appears twice.
        _log.debug("==== Checking tcp reported by ipfix ====")
        tcp_packets = [IpfixFlow(self.n1_workloads[0].ip,
                                 self.n1_workloads[1].ip,
                                 protocol="6",
                                 dstport="8080",
                                 packets="6,4",
                                 octets="325,226"),
                       IpfixFlow(self.n1_workloads[1].ip,
                                 self.n1_workloads[0].ip,
                                 protocol="6",
                                 srcport="8080",
                                 packets="4,6",
                                 octets="226,325")]

        # Currently ipfix_monitor doesn't consistently handle the case of finding in both directions.
        # Adding node dumpstats, /var/log/calico/stats/dump and conntrack -L on each node to aid in validation.

        for host in self.hosts:
            host.calicoctl("node dumpstats")
            host.execute("cat /var/log/calico/stats/dump")
            host.execute("conntrack -L")

        # The 2 variants for packets and octets are: # 6,4;325,226 # 5,5;278,273
        self.mon.assert_flows_present(tcp_packets, 20, allow_others=True)

    def test_reconfigure_pools_connectivity(self):
        """
        Run a mainline multi-host test with IPFIX.
        Because multihost tests are slow to setup, this tests most mainline
        functionality in a single test.
        - Create two hosts
        - Create a network using the Calico IPAM driver, and a workload on
          each host assigned to that network.
        - Check that hosts on the same network can ping each other.
        - Check that hosts on different networks cannot ping each other.
        - Delete default ippool configuration 
        - Apply ippool configuration to ipip enabled. 
        - Run the asymmetric TCP packets testcase with IPIP turned on a pool.  
        - Checking connectivity on the changed ippool configuration
        - Apply ippool configuration to nat-outgoing enabled        
        - Checking connectivity on the changed ippool configuration
        - Changing ippool configuration to nat-outgoing with ipip enabled
        """
        _log.debug("*" * 80)
        for iteration in range(10):
            _log.debug("Iteration %s", iteration)
            _log.debug("==== List starting default ippool configuration ====")
            self.hosts[0].calicoctl("get ippool -o json")

            _log.debug("==== Setting ippool configuration to - default ====")
            self.hosts[0].writefile("test_default.yaml", test_default)
            self.hosts[0].calicoctl("%s -f test_default.yaml" % "apply")
            _log.debug("==== Validate that the ippool configuration for default has been applied ====")
            self.check_data_in_datastore(self.hosts[0], [test_default], ippool_cidr)
            _log.debug("====  nat-outgoing disabled: checking cant ping an external IP - 8.8.8.8 ====")
            self.n1_workloads[0].check_cant_ping(str(nat_outgoing_ip), retries=3)
            _log.debug("==== Checking connectivity on the changed ippool configuration ====")
            self.assert_connectivity(self.n1_workloads, retries=5)
            _log.debug("==== Finished: Checking connectivity on the changed ippool configuration ====")

            _log.debug("==== Apply ippool configuration to - ipip enabled ====")
            self.hosts[0].writefile("test_ipip.yaml", test_ipip)
            self.hosts[0].calicoctl("%s -f test_ipip.yaml" % "apply")
            _log.debug("==== Validate that the ippool configuration for ipip has been applied ====")
            self.check_data_in_datastore(self.hosts[0], [test_ipip], ippool_cidr)
            _log.debug("====  nat-outgoing disabled: checking cant ping an external IP - 8.8.8.8 ====")
            self.n1_workloads[0].check_cant_ping(str(nat_outgoing_ip), retries=3)
            _log.debug("==== Checking connectivity on the changed ippool configuration ====")
            self.assert_connectivity(self.n1_workloads, retries=5)
            _log.debug("==== Finished: Checking connectivity on the changed ippool configuration ====")

            _log.debug("==== Changing ippool configuration to - nat-outgoing enabled ====")
            self.hosts[0].writefile("test_nat_out.yaml", test_nat_out)
            self.hosts[0].calicoctl("%s -f test_nat_out.yaml" % "apply")
            _log.debug("==== Validate that the ippool configuration for nat-outgoing has been applied ====")
            self.check_data_in_datastore(self.hosts[0], [test_nat_out], ippool_cidr)
            _log.debug("====  nat-outgoing: checking can ping an external IP - 8.8.8.8 ====")
            self.n1_workloads[0].check_can_ping(str(nat_outgoing_ip), retries=3)
            _log.debug("==== Checking connectivity on the changed nat-outgoing enabled configuration ====")
            self.assert_connectivity(self.n1_workloads, retries=5)
            _log.debug("==== Finished: Checking connectivity on the changed nat-outgoing enabled configuration ====")

            _log.debug("==== Changing ippool configuration to - nat-outgoing and ipip enabled ====")
            self.hosts[0].writefile("test_ipip_nat_out.yaml", test_ipip_nat_out)
            self.hosts[0].calicoctl("%s -f test_ipip_nat_out.yaml" % "apply")
            _log.debug("==== Validate that the ippool configuration for nat-outgoing / ipip has been applied ====")
            self.check_data_in_datastore(self.hosts[0], [test_ipip_nat_out], ippool_cidr)
            _log.debug("====  nat-outgoing: checking can ping an external IP - 8.8.8.8 ====")
            self.n1_workloads[0].check_can_ping(str(nat_outgoing_ip), retries=3)
            _log.debug("==== Checking connectivity on the changed nat-outgoing and ipip enabled configuration ====")
            self.assert_connectivity(self.n1_workloads, retries=5)
            _log.debug(
                "=== Finished: Checking connectivity on the changed nat-outgoing and ipip enabled configuration ===")

        for host in self.hosts:
            host.calicoctl("node dumpstats")
            host.execute("cat /var/log/calico/stats/dump")
            host.execute("conntrack -L")
