# Copyright (c) 2020 Tigera, Inc. All rights reserved.

import logging
import re
import json
import subprocess
from datetime import datetime
from random import randint

from tests.k8st.test_base import Container, Pod, TestBase
from tests.k8st.utils.utils import DiagsCollector, calicoctl, kubectl, run, node_info, retry_until_success, stop_for_debug

_log = logging.getLogger(__name__)

# Note: The very first step for egress ip test cases is to setup ipipMode/vxlanMode accordingly.
# If we add more test cases for other feature, we would need to refactor overlay mode setup and
# make it easier for each test case to configure it.

def patch_ippool(name, vxlanMode=None, ipipMode=None):
    assert vxlanMode is not None
    assert ipipMode is not None
    json_str = calicoctl("get ippool %s -o json" % name)
    node_dict = json.loads(json_str)
    old_ipipMode = node_dict['spec']['ipipMode']
    old_vxlanMode = node_dict['spec']['vxlanMode']

    calicoctl("""patch ippool %s --patch '{"spec":{"vxlanMode": "%s", "ipipMode": "%s"}}'""" % (
        name,
        vxlanMode,
        ipipMode,
    ))
    _log.info("Updated vxlanMode of %s from %s to %s, ipipMode from %s to %s",
              name, old_vxlanMode, vxlanMode, old_ipipMode, ipipMode)

class _TestEgressIP(TestBase):
    def setUp(self):
        super(_TestEgressIP, self).setUp()

    def env_ippool_setup(self, backend, wireguard):
        self.disableDefaultDenyTest = False
        newEnv = {"FELIX_PolicySyncPathPrefix": "/var/run/nodeagent",
                  "FELIX_EGRESSIPSUPPORT": "EnabledPerNamespaceOrPerPod",
                  "FELIX_IPINIPENABLED": "false",
                  "FELIX_VXLANENABLED": "false",
                  "FELIX_WIREGUARDENABLED": "false"}
        if backend == "VXLAN":
            modeVxlan = "Always"
            modeIPIP = "Never"
            newEnv["FELIX_VXLANENABLED"] = "true"
        elif backend == "IPIP":
            modeVxlan = "Never"
            modeIPIP = "Always"
            newEnv["FELIX_IPINIPENABLED"] = "true"
        elif backend == "NoOverlay":
            modeVxlan = "Never"
            modeIPIP = "Never"
        else:
            raise Exception('wrong backend type')

        patch_ippool("default-ipv4-ippool",
                    vxlanMode=modeVxlan,
                    ipipMode=modeIPIP)

        if wireguard:
            self.disableDefaultDenyTest = True
            newEnv["FELIX_WIREGUARDENABLED"] = "true"
        self.update_ds_env("calico-node", "kube-system", newEnv)

        # Create egress IP pool.
        self.egress_cidr = "10.10.10.0/29"
        calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: egress-ippool-1
spec:
  cidr: %s
  blockSize: 29
  nodeSelector: '!all()'
  vxlanMode: %s
  ipipMode: %s
EOF
""" % (self.egress_cidr, modeVxlan, modeIPIP))
        self.add_cleanup(lambda: calicoctl("delete ippool egress-ippool-1"))

    def tearDown(self):
        super(_TestEgressIP, self).tearDown()

    def test_access_service_node_port(self):

        def check_source_ip(client, dest_ip, port, expected_ips=[], not_expected_ips=[]):
            retry_until_success(client.can_connect, retries=3, wait_time=1, function_kwargs={"ip": dest_ip, "port": port, "command": "wget"})
            reply = client.get_last_output()
            m = re.match(r"^.*client_address=([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+).*$", reply.replace("\n",""))
            if m:
                if len(expected_ips) > 0:
                    self.assertIn(m.group(1), expected_ips)
                if len(not_expected_ips) > 0:
                    self.assertNotIn(m.group(1), not_expected_ips)
            else:
                raise Exception("failed to get client address")

        with DiagsCollector():
            client, _, gateway = self.setup_client_server_gateway("kind-worker2")

            # Create backend service.
            pod_ip, svc_ip, svc_port, node_port = self.create_backend_service("kind-worker3", "backend")

            # Accessing pod ip, service ip, node port, source ip should not be affected by gateway.
            check_source_ip(client, pod_ip, svc_port, expected_ips=[client.ip])
            check_source_ip(client, svc_ip, svc_port, expected_ips=[client.ip])
            _, ips, _ = node_info()
            for ip in ips:
                # For Node port access, source ip can be node ip or tunnel ip depends on setup.
                check_source_ip(client, ip, node_port, not_expected_ips=[gateway.ip])

    def test_ecmp_mainline(self):

        with DiagsCollector():
            # Create egress gateways, with an IP from that pool.
            gw = self.create_gateway_pod("kind-worker", "gw", self.egress_cidr)
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr)
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr)

            # Prepare nine external servers with random port number.
            # The number is three times of the number of gateway pods to make sure
            # every ECMP route get chance to be used.
            servers = []
            for i in range(9):
                s = NetcatServerTCP(randint(100, 65000))
                self.add_cleanup(s.kill)
                s.wait_running()
                self.server_add_route(s, gw)
                self.server_add_route(s, gw2)
                self.server_add_route(s, gw3)
                servers.append(s)

            # Create client.
            _log.info("ecmp create client")
            client_ns = "default"
            client = NetcatClientTCP(client_ns, "test1", annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            self.add_cleanup(client.delete)
            client.wait_ready()

            # Set ECMP hash policy to L4 (source, dest, src port, dest port)
            run("docker exec -t %s sysctl -w net.ipv4.fib_multipath_hash_policy=1" % client.nodename)

            # Send packet to servers and check source ip.
            # Since each server has same port number and different ip,
            # we should expect packets take all three EMCP route.
            gw_ips = [gw.ip, gw2.ip, gw3.ip]
            self.check_ecmp_routes(client, servers, gw_ips, allowed_untaken_count=1)

            # Delete gateway pod one by one and check
            # correct ECMP routes(or unreachable route when no gateway pod
            # is available) are used.
            for pod in [gw, gw2, gw3]:
                _log.info("Removing gateway pod %s", pod.name)
                self.delete_and_confirm(pod.name, "pod", pod.ns)
                self.cleanups.remove(pod.delete)
                gw_ips.remove(pod.ip)
                self.check_ecmp_routes(client, servers, gw_ips)

            # No Gateway pods in the cluster.
            # Validate all egress ip related ARP and FDB entries been removed.
            output = run("docker exec -t %s ip neigh" % client.nodename)
            if output.find('10.10.10') != -1:
                raise Exception('ARP entries not been properly cleared %s' % output)
            output = run("docker exec -t %s bridge fdb show" % client.nodename)
            if output.find('10.10.10') != -1:
                raise Exception('FDB entries not been properly cleared %s' % output)

            # Create gateway pods again.
            # Validate ECMP routes works again.
            gw = self.create_gateway_pod("kind-worker", "gw", self.egress_cidr)
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr)
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr)
            gw_ips = [gw.ip, gw2.ip, gw3.ip]

            self.check_ecmp_routes(client, servers, gw_ips, allowed_untaken_count=1)

    def test_ecmp_with_pod_namespace_selector(self):

        with DiagsCollector():
            # Create egress gateways, with an IP from that pool.
            self.create_namespace("ns2", labels={"egress": "yes"})
            self.create_namespace("ns3", labels={"egress": "yes"})
            gw = self.create_gateway_pod("kind-worker", "gw", self.egress_cidr)

            # blue gateways, different host, different namespaces
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr, ns="ns2", color="blue")
            gw2_1 = self.create_gateway_pod("kind-worker", "gw2-1", self.egress_cidr, ns="default", color="blue")

            # red gateways, same host, same namespaces
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr, ns="ns3")
            gw3_1 = self.create_gateway_pod("kind-worker3", "gw3-1", self.egress_cidr, ns="ns3")

            # Prepare nine external servers with random port number.
            # The number is three times of the number of gateway pods to make sure
            # every ECMP route get chance to be used.
            servers = []
            for i in range(9):
                s = NetcatServerTCP(randint(100, 65000))
                self.add_cleanup(s.kill)
                s.wait_running()
                self.server_add_route(s, gw)
                self.server_add_route(s, gw2)
                self.server_add_route(s, gw2_1)
                self.server_add_route(s, gw3)
                self.server_add_route(s, gw3_1)
                servers.append(s)

            # Create three clients on same node, two of which have same selector.
            client_ns = "default"
            client_red = NetcatClientTCP(client_ns, "testred", node="kind-worker", annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "egress == 'yes'",
            })
            self.add_cleanup(client_red.delete)
            client_red.wait_ready()

            client_red2 = NetcatClientTCP(client_ns, "testred2", node="kind-worker", annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "egress == 'yes'",
            })
            self.add_cleanup(client_red2.delete)
            client_red2.wait_ready()

            client_blue = NetcatClientTCP(client_ns, "testblue", node="kind-worker", annotations={
                "egress.projectcalico.org/selector": "color == 'blue'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            self.add_cleanup(client_blue.delete)
            client_blue.wait_ready()

            # Create one client on another node.
            client_red_all = NetcatClientTCP(client_ns, "testredall", node="kind-worker2", annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            self.add_cleanup(client_red_all.delete)
            client_red_all.wait_ready()

            run("docker exec -t kind-worker sysctl -w net.ipv4.fib_multipath_hash_policy=1")
            run("docker exec -t kind-worker2 sysctl -w net.ipv4.fib_multipath_hash_policy=1")

            # client_red should send egress packets via gw3, gw3_1
            self.check_ecmp_routes(client_red, servers, [gw3.ip, gw3_1.ip], allowed_untaken_count=1)
            # client_red2 should send egress packets via gw3, gw3_1
            self.check_ecmp_routes(client_red2, servers, [gw3.ip, gw3_1.ip], allowed_untaken_count=1)
            # client blue should send egress packets via gw2, gw2_1
            self.check_ecmp_routes(client_blue, servers, [gw2.ip, gw2_1.ip], allowed_untaken_count=1)
            # client_red all should send egress packets via gw, gw3, gw3_1
            self.check_ecmp_routes(client_red_all, servers, [gw.ip, gw3.ip, gw3_1.ip], allowed_untaken_count=1)

            # Restart Felix by updating log level.
            log_level = self.get_ds_env("calico-node", "kube-system", "FELIX_LOGSEVERITYSCREEN")
            if log_level == "Debug":
                new_log_level = "Info"
            else:
                new_log_level = "Debug"
            _log.info("--- Start restarting calico/node ---")
            oldEnv = {"FELIX_LOGSEVERITYSCREEN": log_level}
            newEnv = {"FELIX_LOGSEVERITYSCREEN": new_log_level}
            self.update_ds_env("calico-node", "kube-system", newEnv)
            self.add_cleanup(lambda: self.update_ds_env("calico-node", "kube-system", oldEnv))

            # client_red should send egress packets via gw3, gw3_1
            self.check_ecmp_routes(client_red, servers, [gw3.ip, gw3_1.ip], allowed_untaken_count=1)
            # client_red2 should send egress packets via gw3, gw3_1
            self.check_ecmp_routes(client_red2, servers, [gw3.ip, gw3_1.ip], allowed_untaken_count=1)
            # client blue should send egress packets via gw2, gw2_1
            self.check_ecmp_routes(client_blue, servers, [gw2.ip, gw2_1.ip], allowed_untaken_count=1)
            # client_red all should send egress packets via gw, gw3, gw3_1
            self.check_ecmp_routes(client_red_all, servers, [gw.ip, gw3.ip, gw3_1.ip], allowed_untaken_count=1)

    def test_support_mode(self):

        with DiagsCollector():
            # Support mode is EnabledPerNamespaceOrPerPod.

            # Create two gateway pods
            gw_red = self.create_gateway_pod("kind-worker", "gw-red", self.egress_cidr)
            gw_blue = self.create_gateway_pod("kind-worker2", "gw-blue", self.egress_cidr, color="blue")

            # Create namespace for client pods with egress annotations on red gateway.
            client_ns = "ns-client"
            self.create_namespace(client_ns, annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })

           # Create client with no annotations.
            client_no_annotations = NetcatClientTCP(client_ns, "test-red", node="kind-worker")
            self.add_cleanup(client_no_annotations.delete)
            client_no_annotations.wait_ready()
            client_annotation_override = NetcatClientTCP(client_ns, "test-blue", node="kind-worker", annotations={
                "egress.projectcalico.org/selector": "color == 'blue'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            self.add_cleanup(client_annotation_override.delete)
            client_annotation_override.wait_ready()

            # Create external server.
            server_port = 8089
            server = NetcatServerTCP(server_port)
            self.add_cleanup(server.kill)
            server.wait_running()

            # Give the server a route back to the egress IP.
            self.server_add_route(server, gw_red)
            self.server_add_route(server, gw_blue)

            self.validate_egress_ip(client_no_annotations, server, gw_red.ip)
            self.validate_egress_ip(client_annotation_override, server, gw_blue.ip)

            # Set EgressIPSupport to EnabledPerNamespace.
            newEnv = {"FELIX_EGRESSIPSUPPORT": "EnabledPerNamespace"}
            self.update_ds_env("calico-node", "kube-system", newEnv)

            # Validate egress ip again, pod annotations should be ignored.
            self.validate_egress_ip(client_no_annotations, server, gw_red.ip)
            self.validate_egress_ip(client_annotation_override, server, gw_red.ip)

    def test_egress_ip_with_policy_to_server(self):

        with DiagsCollector():
            client, server, _ = self.setup_client_server_gateway("kind-worker2")

            # Deny egress from client to server.
            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: deny-egress-to-server
spec:
  selector: app == 'client'
  types:
  - Egress
  egress:
  - action: Deny
    protocol: TCP
    destination:
      nets:
      - %s
EOF
""" % (server.ip + "/32"))
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy deny-egress-to-server"))
            retry_until_success(client.cannot_connect, retries=3, wait_time=3, function_kwargs={"ip": server.ip, "port": server.port})

    def test_egress_ip_with_policy_to_gateway(self):

        with DiagsCollector():
            client, server, _ = self.setup_client_server_gateway("kind-worker2")

            # Deny egress from client to server.
            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: deny-egress-to-gateway
spec:
  selector: app == 'client'
  types:
  - Egress
  egress:
  - action: Deny
    protocol: TCP
    destination:
      selector: color == 'red'
EOF
""")
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy deny-egress-to-gateway"))
            retry_until_success(client.cannot_connect, retries=3, wait_time=1, function_kwargs={"ip": server.ip, "port": server.port})

    def test_gateway_termination_annotations(self):

        with DiagsCollector():
            # Create egress gateways, with an IP from that pool.
            termination_grace_period = 5
            gw = self.create_gateway_pod("kind-worker", "gw", self.egress_cidr, "red", "default", termination_grace_period)
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr, "red", "default", termination_grace_period)
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr, "red", "default", termination_grace_period)

            # Create client.
            _log.info("ecmp create client")
            client_ns = "default"
            client = NetcatClientTCP(client_ns, "test1", annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            self.add_cleanup(client.delete)
            client.wait_ready()

            # Delete gateway pod one by one and check correct annotations are applied to the client pod.
            for pod in [gw, gw2, gw3]:
                pod_ip = pod.ip
                now = datetime.now()
                _log.info("Removing gateway pod %s", pod.name)
                self.delete(pod.name, "pod", pod.ns, "false")
                self.check_egress_annotations(client, pod_ip, now, termination_grace_period)
                self.confirm_deletion(pod.name, "pod", pod.ns)
                self.cleanups.remove(pod.delete)

    def test_egress_ip_with_default_deny_policy(self):

        with DiagsCollector():
            # Disable egress ippool encaps
            patch_ippool("egress-ippool-1", "Never", "Never")

            # creating egress gateway pods
            gw_red = self.create_gateway_pod("kind-worker", "gw-red", self.egress_cidr)
            gw_blue = self.create_gateway_pod("kind-worker2", "gw-blue",
                    self.egress_cidr, color="blue")

            # Create namespace for client pods with egress annotations on red gateway.
            client_ns = "ns-client"
            self.create_namespace(client_ns, annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })

            # Create client with no annotations.
            client_red = NetcatClientTCP(client_ns, "test-red", node="kind-worker")
            self.add_cleanup(client_red.delete)
            client_red.wait_ready()
            client_blue = NetcatClientTCP(client_ns, "test-blue", node="kind-worker", annotations={
                "egress.projectcalico.org/selector": "color == 'blue'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            self.add_cleanup(client_blue.delete)
            client_blue.wait_ready()

            # Create external server.
            server_port = 8089
            server = NetcatServerTCP(server_port)
            self.add_cleanup(server.kill)
            server.wait_running()

            # Give the server a route back to the egress IP.
            self.server_add_route(server, gw_red)
            self.server_add_route(server, gw_blue)

            # Add default deny policy
            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: default-deny-policy
spec:
  namespaceSelector: has(projectcalico.org/name) && projectcalico.org/name not in {"kube-system", "calico-system"}
  types:
  - Ingress
  - Egress
  egress:
  - action: Allow
    destination:
      ports:
      - 53
      selector: k8s-app == "kube-dns"
    protocol: UDP
    source: {}
EOF
""")
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy default-deny-policy"))
            # check can not connect to the same node
            retry_until_success(client_red.cannot_connect, retries=3, wait_time=1, function_kwargs={"ip": server.ip, "port": server.port})
            # check can not connect to a different node
            retry_until_success(client_blue.cannot_connect, retries=3, wait_time=1, function_kwargs={"ip": server.ip, "port": server.port})

            if self.disableDefaultDenyTest:
                return

            # Add client server allow policy
            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: allow-client-server
spec:
  types:
  - Ingress
  - Egress
  egress:
  - action: Allow
    protocol: TCP
    destination:
      ports:
      - %s
    source: {}
EOF
""" % (server_port))
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy allow-client-server"))
            # check can connect to the same node
            self.validate_egress_ip(client_red, server, gw_red.ip)
            # check can connect to a different node
            self.validate_egress_ip(client_blue, server, gw_blue.ip)

    def has_ip_rule(self, nodename, ip):
        # Validate egress ip rule exists for a client pod ip on a node.
        output = run("docker exec -t %s ip rule" % nodename)
        if output.find(ip) == -1:
            raise Exception('ip rule does not exist for client pod ip %s, log %s' % (ip, output))

    def check_ecmp_routes(self, client, servers, gw_ips, allowed_untaken_count=0):
        """
        Validate that client went though every ECMP route when
        accessing number of servers.
        """
        _log.info("--- Checking ecmp routes %s from client %s %s ---", gw_ips, client.name, client.ip)
        if len(gw_ips) == 0:
            # No gateway is available, verify no connection can be made.
            for s in servers:
                _log.info("Checking cannot-connect, Client IP: %s Server IP: %s Port: %d", client.ip, s.ip, s.port)
                retry_until_success(client.cannot_connect, retries=3, wait_time=1, function_kwargs={"ip": s.ip, "port": s.port})
            return

        # In case calico-node just restarted and egress ip rule has not been programmed yet,
        # we should wait for it to happen.
        retry_until_success(self.has_ip_rule, retries=3, wait_time=3, function_kwargs={"nodename": client.nodename, "ip": client.ip})

        expected_ips = gw_ips[:]
        for s in servers:
            _log.info("Checking can-connect, Client IP: %s Server IP: %s Port: %d", client.ip, s.ip, s.port)
            retry_until_success(client.can_connect, retries=3, wait_time=1, function_kwargs={"ip": s.ip, "port": s.port})
            # Check the source IP as seen by the server.
            client_ip = s.get_recent_client_ip()
            _log.info("xxxxx ecmp route xxxxxxxxx   Client IPs: %r", client_ip)
            if client_ip not in gw_ips:
                run("docker exec -t %s ip rule" % client.nodename)
                run("docker exec -t %s ip route show table 250" % client.nodename)
                run("docker exec -t %s ip route show table 249" % client.nodename)
                _log.info("stop for debug ecmp route %s  Client IPs: %r", gw_ips, client_ip)
                stop_for_debug()
            assert client_ip in gw_ips, \
                "client ip %s is not one of gateway ips.%s" % (client_ip, gw_ips)

            if client_ip in expected_ips:
                expected_ips.remove(client_ip)

        if len(expected_ips) > allowed_untaken_count:
                _log.info("traffic is not taking ECMP route via gateway ips %s" % (expected_ips))
                stop_for_debug()
                raise Exception("traffic is not taking ECMP route via gateway ips %s" % (expected_ips))

        _log.info("--- Checking ecmp routes %s from client %s %s Done ---", gw_ips, client.name, client.ip)
        return expected_ips

    def validate_egress_ip(self, client, server, expected_egress_ip):
        """
        validate server seen expected ip as source ip
        """
        _log.info("Client IP: %s Server IP: %s Port: %d", client.ip, server.ip, server.port)
        retry_until_success(client.can_connect, retries=3, wait_time=1, function_kwargs={"ip": server.ip, "port": server.port})

        # Check the source IP accessing external server is gateway ip.
        client_ip = server.get_recent_client_ip()
        _log.info("Client IPs: %r", client_ip)
        self.assertIn(client_ip, [expected_egress_ip])

    def check_egress_annotations(self, client, egress_ip, now, termination_grace_period):
        """
        check the maintenance timestamp and cidr annotations were applied to the workload pod
        """
        _log.info("Client IP: %s", client.ip)
        retry_until_success(client.has_egress_annotations, retries=3, wait_time=5, function_kwargs={"egress_ip": egress_ip, "now": now, "termination_grace_period": termination_grace_period})

    def setup_client_server_gateway(self, gateway_node):
        """
        Setup client , server and gateway pod and validate server sees gateway ip as source ip.
        """
        # Create external server.
        server_port = 8089
        server = NetcatServerTCP(server_port)
        self.add_cleanup(server.kill)
        server.wait_running()

        # Create egress gateway, with an IP from that pool.
        gateway = self.create_gateway_pod(gateway_node, "gw", self.egress_cidr)

        client_ns = "default"
        client = NetcatClientTCP(client_ns, "test1", node="kind-worker", labels={"app": "client"}, annotations={
            "egress.projectcalico.org/selector": "color == 'red'",
            "egress.projectcalico.org/namespaceSelector": "all()",
        })
        self.add_cleanup(client.delete)
        client.wait_ready()

        # Give the server a route back to the egress IP.
        self.server_add_route(server, gateway)

        self.validate_egress_ip(client, server, gateway.ip)

        return client, server, gateway

    def copy_pull_secret(self, ns):
        out = run("kubectl get secret cnx-pull-secret -n kube-system -o json")

        # Remove revision and UID information so we can re-apply cleanly.
        # This used to be done with --export, but that option has been removed from kubectl.
        sec = json.loads(out)
        del sec["metadata"]["resourceVersion"]
        del sec["metadata"]["uid"]
        sec["metadata"]["namespace"] = ns
        secIn = json.dumps(sec)

        # Reapply in the new namespace.
        run("echo '%s' | kubectl apply -f -" % secIn)

    def server_add_route(self, server, pod):
        """
        add route to a pod for a server
        """
        server.execute("ip r a %s/32 via %s" % (pod.ip, pod.hostip))

    def create_backend_service(self, host, name, ns="default"):
        """
        Create a backend server pod and a service
        """
        pod = Pod(ns, name, image=None, yaml="""
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: backend
  name: %s
  namespace: %s
spec:
  containers:
  - name: backend
    image: gcr.io/kubernetes-e2e-test-images/echoserver:2.2
  nodeName: %s
  terminationGracePeriodSeconds: 0
""" % (name, ns, host))
        self.add_cleanup(pod.delete)
        pod.wait_ready()

        self.create_service(name, "backend", ns, 8080, traffic_policy="Cluster")
        self.add_cleanup(lambda: kubectl("delete service %s -n %s" % (name, ns)))

        svc_ip = run("kubectl get service " + name + " -n %s -o json | jq -r '.spec.clusterIP'" % ns).strip()
        node_port = run("kubectl get service " + name + " -n %s -o json 2> /dev/null | jq -r '.spec.ports[] | \"\(.nodePort)\"'" % ns).strip()

        return pod.ip, svc_ip, 8080, int(node_port)

    def create_gateway_pod(self, host, name, egress_cidr, color="red", ns="default", termgraceperiod=0):
        """
        Create egress gateway pod, with an IP from that pool.
        """
        self.copy_pull_secret(ns)

        gateway = Pod(ns, name, image=None, yaml="""
apiVersion: v1
kind: Pod
metadata:
  annotations:
    cni.projectcalico.org/ipv4pools: "[\\\"%s\\\"]"
  labels:
    color: %s
  name: %s
  namespace: %s
spec:
  imagePullSecrets:
  - name: cnx-pull-secret
  containers:
  - name: gateway
    image: gcr.io/unique-caldron-775/cnx/tigera/egress-gateway:master-amd64 
    env:
    - name: EGRESS_POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    imagePullPolicy: Always
    securityContext:
      privileged: true
    volumeMounts:
        - mountPath: /var/run
          name: policysync
  nodeName: %s
  terminationGracePeriodSeconds: %d
  volumes:
      - flexVolume:
          driver: nodeagent/uds
        name: policysync
""" % (egress_cidr, color, name, ns, host, termgraceperiod))
        self.add_cleanup(gateway.delete)
        gateway.wait_ready()

        return gateway


class NetcatServerTCP(Container):

    def __init__(self, port):
        super(NetcatServerTCP, self).__init__("subfuzion/netcat", "-v -l -k -p %d" % port, "--privileged")
        self.port = port

    def get_recent_client_ip(self):
        for line in self.logs().split('\n'):
            m = re.match(r"Connection from ([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+) [0-9]+ received", line)
            if m:
                ip = m.group(1)
        return ip

class NetcatClientTCP(Pod):

    def __init__(self, ns, name, node=None, labels=None, annotations=None):
        cmd = ["sleep", "3600"]
        super(NetcatClientTCP, self).__init__(ns, name, image="alpine", node=node, labels=labels, annotations=annotations, cmd=cmd)
        self.last_output = ""

    def can_connect(self, ip, port, command="nc"):
        run("docker exec %s ip rule" % self.nodename, allow_fail=True)
        run("docker exec %s ip r l table 250" % self.nodename, allow_fail=True)
        run("docker exec %s ip r l table 249" % self.nodename, allow_fail=True)
        try:
            self.check_connected(ip, port, command)
            _log.info("'%s' connected, as expected", self.name)
        except subprocess.CalledProcessError:
            _log.exception("Failed to access server")
            _log.warning("'%s' failed to connect, when connection was expected", self.name)
            stop_for_debug()
            raise self.ConnectionError

    def cannot_connect(self, ip, port, command="nc"):
        try:
            self.check_connected(ip, port, command)
            _log.warning("'%s' unexpectedly connected", self.name)
            stop_for_debug()
            raise self.ConnectionError
        except subprocess.CalledProcessError:
            _log.info("'%s' failed to connect, as expected", self.name)

    def check_connected(self, ip, port, command="nc"):
        self.last_output = ""
        if command == "nc":
            self.last_output = self.execute("nc -w 2 %s %d </dev/null" % (ip, port))
        elif command == "wget":
            self.last_output = self.execute("wget -T 2 %s:%d -O -" % (ip, port))
        else:
            raise Exception('received invalid command')

    def has_egress_annotations(self, egress_ip, now, termination_grace_period):
        error_margin = 3
        annotations = self.annotations
        gateway_ip = annotations["egress.projectcalico.org/gatewayMaintenanceGatewayIP"]
        if gateway_ip != egress_ip:
            raise Exception('egress.projectcalico.org/gatewayMaintenanceGatewayCIDR annotation expected to be: %s, but was: %s. Annotations were: %s' % (egress_ip, gateway_ip, annotations))
        started_str = annotations["egress.projectcalico.org/gatewayMaintenanceStartedTimestamp"]
        started = datetime.strptime(started_str, "%Y-%m-%dT%H:%M:%SZ")
        if abs((started - now).total_seconds()) > error_margin:
            raise Exception('egress.projectcalico.org/gatewayMaintenanceStartedTimestamp annotation expected to be: within %ds of %s, but was: %s. Annotations were: %s' % (error_margin, now, started_str, annotations))
        finished_str = annotations["egress.projectcalico.org/gatewayMaintenanceFinishedTimestamp"]
        finished = datetime.strptime(finished_str, "%Y-%m-%dT%H:%M:%SZ")
        if abs((finished - started).total_seconds()) > (error_margin + termination_grace_period):
            raise Exception('egress.projectcalico.org/gatewayMaintenanceFinishedTimestamp annotation expected to be: within %ds of %s, but was: %s. Annotations were: %s' % ((error_margin + termination_grace_period), started, finished_str, annotations))
 
    def get_last_output(self):
        return self.last_output

    class ConnectionError(Exception):
        pass

class TestEgressIPNoOverlay(_TestEgressIP):
    def setUp(self):
        super(_TestEgressIP, self).setUp()
        self.env_ippool_setup(backend="NoOverlay", wireguard=False)

class TestEgressIPWithIPIP(_TestEgressIP):
    def setUp(self):
        super(_TestEgressIP, self).setUp()
        self.env_ippool_setup(backend="IPIP", wireguard=False)

class TestEgressIPWithVXLAN(_TestEgressIP):
    def setUp(self):
        super(_TestEgressIP, self).setUp()
        self.env_ippool_setup(backend="VXLAN", wireguard=False)

class TestEgressIPNoOverlayAndWireguard(TestEgressIPNoOverlay):
    def setUp(self):
        super(_TestEgressIP, self).setUp()
        self.env_ippool_setup(backend="NoOverlay", wireguard=True)

class TestEgressIPWithIPIPAndWireguard(TestEgressIPWithIPIP):
    def setUp(self):
        super(_TestEgressIP, self).setUp()
        self.env_ippool_setup(backend="IPIP", wireguard=True)

class TestEgressIPWithVXLANAndWireguard(_TestEgressIP):
    def setUp(self):
        super(_TestEgressIP, self).setUp()
        self.env_ippool_setup(backend="VXLAN", wireguard=True)
