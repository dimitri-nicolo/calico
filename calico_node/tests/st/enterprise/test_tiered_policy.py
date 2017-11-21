# Copyright 2016-2017 Tigera, Inc
import copy
import functools
import json
import logging
import subprocess
import time
import yaml
from nose_parameterized import parameterized
from multiprocessing.dummy import Pool

from tests.st.test_base import TestBase
from tests.st.utils.docker_host import DockerHost
from tests.st.utils.utils import assert_number_endpoints, get_ip, \
    ETCD_CA, ETCD_CERT, ETCD_KEY, ETCD_HOSTNAME_SSL, ETCD_SCHEME
from tests.st.utils.utils import wipe_etcd as WIPE_ETCD

_log = logging.getLogger(__name__)

_log = logging.getLogger(__name__)
_log.setLevel(logging.DEBUG)

POST_DOCKER_COMMANDS = ["docker load -i /code/cnx-node.tar",
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
        # Allow time for cnx-node to load
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

    @parameterized.expand([
        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true1"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source": {"selector": "test == 'True'"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination": {"selector": "test == 'True'"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "True"},
         True
         ),

        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true2"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source": {"selector": "test != 'True'"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination": {"selector": "test != 'True'"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "False"},
         False
         ),

        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true3"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source": {"selector": "has(test)"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination": {"selector": "has(test)"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "any_old_value"},
         True
         ),

        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true4"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source": {"selector": "!has(test)"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination": {"selector": "!has(test)"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "no_one_cares"},
         False
         ),

        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true5"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source": {"selector": "test in {'true', 'false'}"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination": {"selector": "test in {'true', 'false'}"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "true"},
         True
         ),

        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true6"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source": {"selector": "test in {'true', 'false'}"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination": {"selector": "test in {'true', 'false'}"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "false"},
         True
         ),

        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true7"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source": {"selector": "test not in {'true', 'false'}"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination": {"selector": "test not in {'true', 'false'}"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "neither"},
         False
         ),

        ([{"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true8a"},
           "spec":
               {
                   "selector": "test == 'true'",
                   "ingress": [
                       {"action": "deny"},
                   ],
                   "egress": [
                       {"action": "deny"},
                   ]
               }
           },
          {"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true8b"},
           "spec":
               {
                   "ingress": [
                       {"action": "allow"},
                   ],
                   "egress": [
                       {"action": "allow"},
                   ]
               }
           }
          ],
         {"test": "true"},
         True
         ),

        ([{"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true9a"},
           "spec":
               {
                   "selector": "test != 'true'",
                   "ingress": [
                       {"action": "deny"},
                   ],
                   "egress": [
                       {"action": "deny"},
                   ]
               }
           },
          {"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true9b"},
           "spec":
               {
                   "ingress": [
                       {"action": "allow"},
                   ],
                   "egress": [
                       {"action": "allow"},
                   ]
               }
           }
          ],
         {"test": "true"},
         False
         ),

        ([{"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true10a"},
           "spec":
               {
                   "selector": "has(test)",
                   "ingress": [
                       {"action": "deny"},
                   ],
                   "egress": [
                       {"action": "deny"},
                   ]
               }
           },
          {"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true10b"},
           "spec":
               {
                   "ingress": [
                       {"action": "allow"},
                   ],
                   "egress": [
                       {"action": "allow"},
                   ]
               }
           }
          ],
         {"test": "true"},
         True
         ),

        ([{"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true11a"},
           "spec":
               {
                   "selector": "!has(test)",
                   "ingress": [
                       {"action": "deny"},
                   ],
                   "egress": [
                       {"action": "deny"},
                   ]
               }
           },
          {"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true11b"},
           "spec":
               {
                   "ingress": [
                       {"action": "allow"},
                   ],
                   "egress": [
                       {"action": "allow"},
                   ]
               }
           }
          ],
         {"test": "true"},
         False
         ),

        ([{"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true12a"},
           "spec":
               {
                   "selector": "test in {'true', 'false'}",
                   "ingress": [
                       {"action": "deny"},
                   ],
                   "egress": [
                       {"action": "deny"},
                   ]
               }
           },
          {"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true12b"},
           "spec":
               {
                   "ingress": [
                       {"action": "allow"},
                   ],
                   "egress": [
                       {"action": "allow"},
                   ]
               }
           }
          ],
         {"test": "true"},
         True
         ),

        ([{"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true13a"},
           "spec":
               {
                   "selector": "test in {'true', 'false'}",
                   "ingress": [
                       {"action": "deny"},
                   ],
                   "egress": [
                       {"action": "deny"},
                   ]
               }
           },
          {"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true13b"},
           "spec":
               {
                   "ingress": [
                       {"action": "allow"},
                   ],
                   "egress": [
                       {"action": "allow"},
                   ]
               }
           }
          ],
         {"test": "false"},
         True
         ),

        ([{"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true14a"},
           "spec":
               {
                   "selector": "test not in {'true', 'false'}",
                   "ingress": [
                       {"action": "deny"},
                   ],
                   "egress": [
                       {"action": "deny"},
                   ]
               }
           },
          {"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true14b"},
           "spec":
               {
                   "ingress": [
                       {"action": "allow"},
                   ],
                   "egress": [
                       {"action": "allow"},
                   ]
               }
           }
          ],
         {"test": "neither"},
         False
         ),

        ([{"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true15a"},
           "spec":
               {
                   "selector": "has(test) && test in {'true', 'false'} && test == 'true'",
                   "ingress": [
                       {"action": "deny"},
                   ],
                   "egress": [
                       {"action": "deny"},
                   ]
               }
           },
          {"apiVersion": "v1",
           "kind": "policy",
           "metadata": {"name": "deny-test-true15b"},
           "spec":
               {
                   "ingress": [
                       {"action": "allow"},
                   ],
                   "egress": [
                       {"action": "allow"},
                   ]
               }
           }
          ],
         {"test": "true"},
         True
         ),

        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true16"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source":
                      {"selector": "has(test) && test in {'True', 'False'} && test == 'True'"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination":
                       {"selector": "has(test) && test in {'True', 'False'} && test == 'True'"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "True"},
         True
         ),

        ({"apiVersion": "v1",
          "kind": "policy",
          "metadata": {"name": "deny-test-true17"},
          "spec": {
              "ingress": [{
                  "action": "deny",
                  "source":
                      {"selector":
                           "has(test) && test not in {'True', 'False'} && test == 'Sausage'"},
              },
                  {"action": "allow"}
              ],
              "egress": [
                  {"action": "deny",
                   "destination":
                       {"selector":
                            "has(test) && test not in {'True', 'False'} && test == 'Sausage'"}},
                  {"action": "allow"}
              ]},
          },
         {"test": "Sausage"},
         True
         ),
    ])
    def test_selectors(self, policy, workload_label, no_label_expected_result):
        """
        Tests selectors in policy.
        :param policy: The policy to apply
        :param workload_label: The label to add to one of the workload endpoints
        :param no_label_expected_result: Whether we'd expect the policy to block connectivity if
        the workloads do not have the label.
        :return:
        """
        # set workload config
        host = self.hosts[0]
        weps = yaml.load(host.calicoctl("get wep -o yaml"))
        wep_old = weps[0]
        wep = copy.deepcopy(wep_old)
        wep['metadata']['labels'] = workload_label
        self._apply_data(wep, host)
        # check connectivity OK
        self.assert_connectivity(self.n1_workloads)
        # set up policy
        self._apply_data(policy, host)
        # check connectivity not OK
        self.assert_no_connectivity(self.n1_workloads)
        # Restore workload config (i.e. remove the label)
        self._apply_data(wep_old, host)
        if no_label_expected_result:
            # check connectivity OK again
            self.assert_connectivity(self.n1_workloads)
        else:
            self.assert_no_connectivity(self.n1_workloads)


class IpNotFound(Exception):
    pass
