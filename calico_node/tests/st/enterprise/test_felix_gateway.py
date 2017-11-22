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
import time

import yaml
from nose_parameterized import parameterized

from tests.st.test_base import TestBase
from tests.st.utils.docker_host import DockerHost
from tests.st.utils.utils import assert_number_endpoints, ETCD_CA, ETCD_CERT, \
    ETCD_KEY, ETCD_HOSTNAME_SSL, ETCD_SCHEME, get_ip, log_and_run, retry_until_success
from tests.st.utils.utils import wipe_etcd as WIPE_ETCD

_log = logging.getLogger(__name__)
_log.setLevel(logging.DEBUG)

POST_DOCKER_COMMANDS = [
    "docker load -i /code/cnx-node.tar",
    "docker load -i /code/busybox.tar",
    "docker load -i /code/workload.tar",
]

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


class TestFelixOnGateway(TestBase):
    hosts = None
    networks = None

    def setUp(self):
        _log.debug("Override the TestBase setUp() method which wipes etcd. Do nothing.")
        # Wipe policies and tiers before each test
        self.delete_all("pol")
        self.delete_all("tier")
        self.delete_all("hostEndpoint")

        # Wait for felix to remove the policy and allow traffic through the gateway.
        retry_until_success(self.assert_host_can_curl_ext)

    def tearDown(self):
        # Wipe policies and tiers after each test
        self.delete_all("pol")
        self.delete_all("tier")
        self.delete_all("hostEndpoint")
        super(TestFelixOnGateway, self).tearDown()

    def delete_all(self, resource):
        # Grab all objects of a resource type
        objects = yaml.load(self.hosts[0].calicoctl("get %s -o yaml" % resource))
        # and delete them (if there are any)
        if len(objects) > 0:
            self._delete_data(objects, self.hosts[0])

    def _delete_data(self, data, host):
        _log.debug("Deleting data with calicoctl: %s", data)
        self._use_calicoctl("delete", data, host)

    @staticmethod
    def _use_calicoctl(action, data, host):
        # use calicoctl with data
        host.writefile("new_profile",
                       yaml.dump(data, default_flow_style=False))
        host.calicoctl("%s -f new_profile" % action)

    @staticmethod
    def sleep(length):
        _log.debug("Sleeping for %s" % length)
        time.sleep(length)

    @classmethod
    def setUpClass(cls):
        wipe_etcd()
        cls.policy_tier_name = "default"
        cls.next_tier_allowed = False

        # We set up an additional docker network to act as the external
        # network.  The Gateway container is connected to both networks.
        # and we configure it as a NAT gateway.
        #
        #  "cali-st-ext" host
        #   container
        #      |
        #  "cali-st-ext" docker
        #    bridge
        #      |
        #  Gateway           Host
        #  container         container
        #         \          /
        #        default docker
        #            bridge

        # First, create the hosts and the gateway.
        cls.hosts = []
        log_and_run("docker rm -f cali-st-gw || true")
        cls.gateway = DockerHost("cali-st-gw",
                                 additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                                 post_docker_commands=POST_DOCKER_COMMANDS,
                                 start_calico=False)
        cls.gateway_hostname = cls.gateway.execute("hostname")
        log_and_run("docker rm -f cali-st-host || true")
        cls.host = DockerHost("cali-st-host",
                              additional_docker_options=ADDITIONAL_DOCKER_OPTIONS,
                              post_docker_commands=POST_DOCKER_COMMANDS,
                              start_calico=False)
        cls.host_hostname = cls.host.execute("hostname")
        cls.hosts.append(cls.gateway)
        cls.hosts.append(cls.host)

        # Delete the nginx container if it still exists.  We need to do this
        # before we try to remove the network.
        log_and_run("docker rm -f cali-st-ext-nginx || true")

        # Create the external network.
        log_and_run("docker network rm cali-st-ext || true")
        # Use 172.19.0.0 to avoid clash with normal docker subnet and
        # docker-in-docker subnet
        log_and_run("docker network create --driver bridge --subnet 172.19.0.0/16 cali-st-ext")

        # And an nginx server on the external network only.
        log_and_run("docker run --network=cali-st-ext -d --name=cali-st-ext-nginx nginx")

        for host in cls.hosts:
            host.start_calico_node()

        # Get the internal IP of the gateway.  We do this before we add the second
        # network since it means we don't have to figure out which IP is which.
        int_ip = cls.get_container_ip("cali-st-gw")
        cls.gateway_int_ip = int_ip
        _log.info("Gateway internal IP: %s", cls.gateway_int_ip)

        # Add the gateway to the external network.
        log_and_run("docker network connect cali-st-ext cali-st-gw")
        cls.gateway.execute("ip addr")

        # Get the IP of the external server.
        ext_ip = cls.get_container_ip("cali-st-ext-nginx")
        cls.ext_server_ip = ext_ip
        _log.info("External workload IP: %s", cls.ext_server_ip)

        # Configure the internal host to use the gateway for the external IP.
        cls.host.execute("ip route add %s via %s" %
                         (cls.ext_server_ip, cls.gateway_int_ip))

        # Configure the gateway to forward and NAT.
        cls.gateway.execute("sysctl -w net.ipv4.ip_forward=1")
        cls.gateway.execute("iptables -t nat -A POSTROUTING --destination %s -j MASQUERADE" %
                            cls.ext_server_ip)

        # Create workload networks.
        cls.networks = []
        cls.networks.append(cls.gateway.create_network("testnet2"))
        cls.sleep(10)

        cls.n1_workloads = []
        # Create two workloads on cls.hosts[0] and one on cls.hosts[1] all in network 1.
        cls.n1_workloads.append(cls.host.create_workload("workload_hn1_1",
                                                         image="workload",
                                                         network=cls.networks[0]))
        cls.sleep(2)
        cls.n1_workloads.append(cls.gateway.create_workload("workload_gwn1_1",
                                                            image="workload",
                                                            network=cls.networks[0]))
        # Assert that endpoints are in Calico
        assert_number_endpoints(cls.gateway, 1)
        assert_number_endpoints(cls.host, 1)

    @classmethod
    def get_container_ip(cls, container_name):
        ip = log_and_run(
            "docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' %s" %
            container_name)
        return ip.strip()

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
        log_and_run("docker rm -f cali-st-ext-nginx || true")
        wipe_etcd()

    def test_ingress_policy_can_block_through_traffic(self):
        self.add_admin_tier()
        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'port80-int', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'ingress': [
                    {
                        'protocol': 'tcp',
                        'destination': {'ports': [80]},
                        'action': 'deny'
                    },
                ],
                'egress': [
                    {'action': 'deny'},
                ],
                'selector': 'role == "gateway-int"'
            }
        })
        self.add_gateway_internal_iface()
        retry_until_success(self.assert_host_can_not_curl_ext, 3)

    def test_ingress_policy_can_allow_through_traffic(self):
        self.add_admin_tier()
        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'port80-int', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'ingress': [
                    {
                        'protocol': 'tcp',
                        'destination': {'ports': [80]},
                        'action': 'allow'
                    },
                ],
                'egress': [
                    {'action': 'deny'},
                ],
                'selector': 'role == "gateway-int"'
            }
        })
        self.add_gateway_internal_iface()
        retry_until_success(self.assert_host_can_curl_ext, 3)

    def test_egress_policy_can_block_through_traffic(self):
        self.add_admin_tier()
        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'port80-ext', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'ingress': [
                    {
                        'action': 'deny',
                    },
                ],
                'egress': [
                    {
                        'protocol': 'tcp',
                        'destination': {'ports': [80]},
                        'action': 'deny'
                    },
                ],
                'selector': 'role == "gateway-ext"'
            }
        })
        self.add_gateway_external_iface()
        retry_until_success(self.assert_host_can_not_curl_ext, 3)

    def test_egress_policy_can_allow_through_traffic(self):
        self.add_admin_tier()
        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'port80-ext', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'ingress': [
                    {
                        'action': 'deny',
                    },
                ],
                'egress': [
                    {
                        'protocol': 'tcp',
                        'destination': {'ports': [80]},
                        'action': 'allow'
                    },
                ],
                'selector': 'role == "gateway-ext"'
            }
        })
        self.add_gateway_external_iface()
        retry_until_success(self.assert_host_can_curl_ext, 3)

    def test_ingress_and_egress_policy_can_allow_through_traffic(self):
        self.add_admin_tier()
        self.add_gateway_external_iface()
        self.add_gateway_internal_iface()
        self.add_host_iface()

        # Adding the host endpoints should break connectivity until we add policy back in.
        retry_until_success(self.assert_host_can_not_curl_ext, 3)

        # Add in the policy...
        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'host-out', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'selector': 'role == "host"',
                'egress': [{'action': 'allow'}],
                'ingress': [{'action': 'allow'}],
            }
        })
        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'port80-int', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'ingress': [
                    {
                        'protocol': 'tcp',
                        'destination': {
                            'ports': [80],
                            'net': self.ext_server_ip + "/32",
                        },
                        'source': {
                            'selector': 'role == "host"',
                        },
                        'action': 'allow'
                    },
                ],
                'egress': [],
                'selector': 'role == "gateway-int"'
            }
        })
        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'port80-ext', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'ingress': [],
                'egress': [
                    {
                        'protocol': 'tcp',
                        'destination': {
                            'ports': [80],
                            'net': self.ext_server_ip + "/32",
                        },
                        'source': {
                            'selector': 'role == "host"',
                        },
                        'action': 'allow'
                    },
                ],
                'selector': 'role == "gateway-ext"'
            }
        })
        retry_until_success(self.assert_host_can_curl_ext, 3)

    @parameterized.expand([
        ('allow', 'deny'),
        ('deny', 'allow')
    ])
    def test_conflicting_ingress_and_egress_policy(self, in_action, out_action):
        # If there is policy on the ingress and egress interface then both should
        # get applied and 'deny' should win.
        self.add_admin_tier()
        self.add_host_iface()
        self.add_gateway_external_iface()
        self.add_gateway_internal_iface()

        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'port80-int', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'ingress': [
                    {
                        'action': in_action
                    },
                ],
                'egress': [],
                'selector': 'role == "gateway-int"'
            }
        })
        self.add_policy({
            'apiVersion': 'v1',
            'kind': 'policy',
            'metadata': {'name': 'port80-ext', 'tier': 'admin'},
            'spec': {
                'order': 10,
                'ingress': [],
                'egress': [
                    {
                        'action': out_action
                    },
                ],
                'selector': 'role == "gateway-ext"'
            }
        })
        retry_until_success(self.assert_host_can_not_curl_ext, 3)

    def add_policy(self, policy_data):
        self._apply_resources(policy_data, self.gateway)

    def add_admin_tier(self):
        tier_data = {
            'apiVersion': 'v1',
            'kind': 'tier',
            'metadata': {'name': 'admin'},
            'spec': {'order': 1}
        }
        self._apply_resources(tier_data, self.gateway)

    def add_gateway_internal_iface(self):
        host_endpoint_data = {
            'apiVersion': 'v1',
            'kind': 'hostEndpoint',
            'metadata': {
                'name': 'gw-int',
                'node': self.gateway_hostname,
                'labels': {'role': 'gateway-int'}
            },
            'spec': {
                'interfaceName': 'eth0'
            }
        }
        self._apply_resources(host_endpoint_data, self.gateway)

    def add_gateway_external_iface(self):
        host_endpoint_data = {
            'apiVersion': 'v1',
            'kind': 'hostEndpoint',
            'metadata': {
                'name': 'gw-ext',
                'node': self.gateway_hostname,
                'labels': {'role': 'gateway-ext'}
            },
            'spec': {
                'interfaceName': 'eth1'
            }
        }
        self._apply_resources(host_endpoint_data, self.gateway)

    def add_host_iface(self):
        host_endpoint_data = {
            'apiVersion': 'v1',
            'kind': 'hostEndpoint',
            'metadata': {
                'name': 'host-int',
                'node': self.host_hostname,
                'labels': {'role': 'host'}
            },
            'spec': {
                'interfaceName': 'eth0',
                'expectedIPs': [self.get_container_ip('cali-st-host')],
            }
        }
        self._apply_resources(host_endpoint_data, self.gateway)

    @staticmethod
    def _apply_resources(resources, host):
        # Apply new resources.
        host.writefile("resources",
                       yaml.dump(resources, default_flow_style=False))
        host.calicoctl("apply -f resources")

    def assert_host_can_curl_ext(self):
        try:
            self.host.execute("curl -m 2 %s" % self.ext_server_ip)
        except subprocess.CalledProcessError:
            _log.exception("Internal host failed to curl external server IP: %s",
                           self.ext_server_ip)
            self.fail(
                "Internal host failed to curl external server IP: %s" % self.ext_server_ip)

    def assert_host_can_not_curl_ext(self):
        try:
            self.host.execute("curl -m 1 %s" % self.ext_server_ip)
        except subprocess.CalledProcessError:
            return
        else:
            self.fail("Internal host can curl external server IP: %s" % self.ext_server_ip)


class IpNotFound(Exception):
    pass


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
