# Copyright (c) 2020-2022 Tigera, Inc. All rights reserved.

import logging
import re
import json
import subprocess
import time
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
                  "FELIX_WIREGUARDENABLED": "false",
                  "FELIX_EGRESSGATEWAYPOLLINTERVAL": "1"}
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

        # After restarting felixes, wait for 20s to ensure Felix is past its route-cleanup grace period.
        time.sleep(20)

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
            for g in [gw, gw2, gw3]:
                g.wait_ready()

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
                self.check_ecmp_routes(client, servers, gw_ips, allowed_untaken_count=1)

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
            for g in [gw, gw2, gw3]:
                g.wait_ready()

            self.check_ecmp_routes(client, servers, gw_ips, allowed_untaken_count=1)

    def test_egw_readiness(self):
        """
        Test egress gateway readiness probes and felix-to-EGW probes.  After blocking an
        EGW readiness probe Felix should remove that EGW from the pool.
        :return: None
        """

        with DiagsCollector():
            # Prepare nine external servers with random port number.
            # The number is three times of the number of gateway pods to make sure
            # every ECMP route get chance to be used.
            servers = []
            for i in range(9):
                s = NetcatServerTCP(randint(100, 65000))
                servers.append(s)
                self.add_cleanup(s.kill)

            # Create a few egress gateways.  We set each one up with a different ICMP probe so that we can
            # break each one's probe separately.
            gw = self.create_gateway_pod("kind-worker", "gw", self.egress_cidr, icmp_probes=servers[0].ip)
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr, icmp_probes=servers[1].ip)
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr, icmp_probes=servers[2].ip)

            for s in servers:
                s.wait_running()
                self.server_add_route(s, gw)
                self.server_add_route(s, gw2)
                self.server_add_route(s, gw3)

            for g in [gw, gw2, gw3]:
                g.wait_ready()

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
            # we should expect packets take all three EMCP routes.
            gw_ips = [gw.ip, gw2.ip, gw3.ip]
            self.check_ecmp_routes(client, servers, gw_ips)

            # Break one of the gateway's ICMP probes, it should be taken out of service.
            self.server_del_route(servers[0], gw)
            gw.wait_not_ready()

            def check_routes(expected):
                tables = self.read_client_hops_for_node(client.nodename)
                hops = tables[client.ip]["hops"]
                assert set(hops) == set(expected), ("Expected client's hops to be %s not %s." % (expected, hops))

            retry_until_success(check_routes, retries=10, wait_time=3, function_args=[[gw2.ip, gw3.ip]])
            self.check_ecmp_routes(client, servers[1:], gw_ips[1:])

            # Reinstate the probe, it should be added back into service.
            self.server_add_route(servers[0], gw)
            gw.wait_ready()

            retry_until_success(check_routes, retries=10, wait_time=3, function_args=[[gw.ip, gw2.ip, gw3.ip]])
            self.check_ecmp_routes(client, servers, gw_ips)

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
            for g in [gw, gw2, gw2_1, gw3, gw3_1]:
                g.wait_ready()

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
            for g in [gw_blue, gw_red]:
                g.wait_ready()

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

    def test_egress_ip_host_endpoint_policy(self):
        with DiagsCollector():
            client, server, gw = self.setup_client_server_gateway("kind-worker2")

            # Create auto HEPs
            patchStr = "{\"spec\": {\"controllers\": {\"node\": {\"hostEndpoint\": {\"autoCreate\": \"Enabled\"}}}}}"
            patchStr_disable = "{\"spec\": {\"controllers\": {\"node\": {\"hostEndpoint\": {\"autoCreate\": \"Disabled\"}}}}}"
            kubectl("patch kubecontrollersconfiguration default --patch '%s'" % (patchStr)).strip()

            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: default-deny-all-heps
spec:
  selector: projectcalico.org/created-by == "calico-kube-controllers"
  types:
  - Ingress
  - Egress
EOF
""")
            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: allowed-flows-all-heps
spec:
  selector: projectcalico.org/created-by == "calico-kube-controllers"
  ingress:
  # localhost to localhost ingress
  - action: Allow
    destination:
      nets:
      - 127.0.0.0/8
  # kube-apiserver to kubelet ingress
  - action: Allow
    protocol: TCP
    source:
      selector: has(node-role.kubernetes.io/control-plane)
    destination:
      ports:
      - 10250
  # prometheus to calico-node ingress
  - action: Allow
    protocol: TCP
    source:
      selector: projectcalico.org/created-by == "calico-kube-controllers"
    destination:
      ports:
      - 9081
      - 9091
      - 9900
  egress:
  # localhost to localhost egress (both ingress&egress required, conntrack entry stays in SYN_SENT with egress only?)
  - action: Allow
    destination:
      nets:
      - 127.0.0.0/8
  - action: Allow
    source: {}
    destination:
      domains:
        - k8s.gcr.io
        - storage.googleapis.com
EOF
""")

            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: allowed-flows-control-plane-heps
spec:
  selector: has(node-role.kubernetes.io/control-plane)
  egress:
  # kube-apiserver to kubelet egress (selects all destinations to include self) - kubectl logs, port-forwarding
  - action: Allow
    destination:
      ports:
      - 10250
    protocol: TCP
  # kube-apiserver to tigera-apiserver egress
  - action: Allow
    protocol: TCP
    destination:
      namespaceSelector: projectcalico.org/name == 'tigera-system'
      selector: k8s-app == 'tigera-apiserver'
EOF
""")
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy allowed-flows-control-plane-heps"))
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy allowed-flows-all-heps"))
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy default-deny-all-heps"))
            self.add_cleanup(lambda: kubectl("patch kubecontrollersconfiguration default --patch '%s'" % (patchStr_disable)).strip())
            retry_until_success(client.can_connect, retries=3, wait_time=1, function_kwargs={"ip": server.ip, "port": server.port})
            self.validate_egress_ip(client, server, gw.ip)

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
            client, server, gw = self.setup_client_server_gateway("kind-worker2")
            # Note: setup_client_server_gateway checks the baseline connectivity.

            # Start by denying all egress traffic from the client.  Otherwise, we can't be sure that our allow
            # policy is actually what makes the difference.
            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: deny-egress
spec:
  order: 100
  selector: app == 'client'
  types:
  - Egress
  egress:
  - action: Deny
EOF
""")
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy deny-egress"))
            retry_until_success(client.cannot_connect, retries=3, wait_time=1, function_kwargs={"ip": server.ip, "port": server.port})

            # Allow egress to the server only (there's no need to allow egress to the gateway).
            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: allow-to-server
spec:
  order: 99
  selector: app == 'client'
  types:
  - Egress
  egress:
  - action: Allow
    protocol: TCP
    destination:
      nets:
      - "%s/32"
EOF
""" % server.ip)
            self.add_cleanup(lambda: calicoctl("delete globalnetworkpolicy allow-to-server"))
            retry_until_success(client.can_connect, retries=3, wait_time=1, function_kwargs={"ip": server.ip, "port": server.port})
            self.validate_egress_ip(client, server, gw.ip)

    def test_gateway_termination_annotations(self):

        with DiagsCollector():
            # Create egress gateways, with an IP from that pool.
            termination_grace_period = 10
            gw = self.create_gateway_pod("kind-worker", "gw", self.egress_cidr, "red", "default", termination_grace_period)
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr, "red", "default", termination_grace_period)
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr, "red", "default", termination_grace_period)
            for g in [gw, gw2, gw3]:
                g.wait_ready()

            # Create client.
            _log.info("ecmp create client")
            client_ns = "default"
            client = NetcatClientTCP(client_ns, "test1", annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            self.add_cleanup(client.delete)
            client.wait_ready()

            retry_until_success(self.has_ip_route_and_table, retries=3, wait_time=3,
                                function_kwargs={
                                    "nodename": client.nodename,
                                    "client_ip": client.ip,
                                    "gateway_ips": [gw.ip, gw2.ip, gw3.ip]
                                })

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
            for g in [gw_blue, gw_red]:
                g.wait_ready()

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

    def test_max_hops_pod_annotation(self):
        with DiagsCollector():
            # Create 3 egress gateways, with an IP from that pool.
            gw1 = self.create_gateway_pod("kind-worker", "gw1", self.egress_cidr)
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr)
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr)
            for g in [gw1, gw2, gw3]:
                g.wait_ready()
            _log.info("test_max_hops_pod_annotation: created gw pods [%s, %s, %s]", gw1.ip, gw2.ip, gw3.ip)

            # Create three clients on the same node with maxNextHops set to 2.
            client_ns = "default"
            node = "kind-worker"
            clients = []
            for i in range(3):
                _log.info("test_max_hops_pod_annotation: create client%d", i)
                c = NetcatClientTCP(client_ns, "test%d" % i, node=node, annotations={
                    "egress.projectcalico.org/maxNextHops": "2",
                    "egress.projectcalico.org/selector": "color == 'red'",
                    "egress.projectcalico.org/namespaceSelector": "all()",
                })
                self.add_cleanup(c.delete)
                c.wait_ready()
                clients.append(c)
            c1 = clients[0]
            c2 = clients[1]
            c3 = clients[2]
            _log.info("test_max_hops_pod_annotation: created client pods [%s, %s, %s]", c1.ip, c2.ip, c3.ip)

            retry_until_success(self.has_ip_rule, retries=3, wait_time=3, function_kwargs={"nodename": c3.nodename, "ip": c3.ip})

            def verify_tables_and_hops():
                node_rules_and_tables = self.read_client_hops_for_node(node)
                assert len(node_rules_and_tables) == 3
                print(node_rules_and_tables)
                # Verify we have 3 different tables.
                table1 = node_rules_and_tables[c1.ip]["table"]
                table2 = node_rules_and_tables[c2.ip]["table"]
                table3 = node_rules_and_tables[c3.ip]["table"]
                print(table1, table2, table3)
                table_set = {table1, table2, table3}
                assert len(table_set) == 3

                hops1 = sorted(node_rules_and_tables[c1.ip]["hops"])
                hops2 = sorted(node_rules_and_tables[c2.ip]["hops"])
                hops3 = sorted(node_rules_and_tables[c3.ip]["hops"])
                return hops1, hops2, hops3

            hops1, hops2, hops3 = verify_tables_and_hops()
            # Verify each table has different hops.
            assert (hops1 != hops2) and (hops2 != hops3) and (hops1 != hops3)

            # Delete one gateway, only two hops left
            pod = gw1
            _log.info("Removing gateway pod %s", pod.name)
            self.delete_and_confirm(pod.name, "pod", pod.ns)
            self.cleanups.remove(pod.delete)

            # We should see same set of hops for each client since each table should has at least two hops.
            hops1, hops2, hops3 = verify_tables_and_hops()
            assert (hops1 == hops2) and (hops2 == hops3)

    def test_max_hops_namespace_annotation(self):
        with DiagsCollector():
            # Create 3 egress gateways, with an IP from that pool.
            # Create namespace for client pods with egress annotations on red gateway.
            client_ns = "ns-client"
            self.create_namespace(client_ns, annotations={
                "egress.projectcalico.org/maxNextHops": "2",
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            gw1 = self.create_gateway_pod("kind-worker", "gw1", self.egress_cidr, ns=client_ns)
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr, ns=client_ns)
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr, ns=client_ns)
            for g in [gw1, gw2, gw3]:
                g.wait_ready()
            _log.info("test_max_hops_namespace_annotation: created gw pods [%s, %s, %s]", gw1.ip, gw2.ip, gw3.ip)

            # Create three clients without annotations on the same node with maxNextHops set to 2.
            node = "kind-worker"
            clients = []
            for i in range(3):
                _log.info("test_max_hops_namespace_annotation: create client%d", i)
                c = NetcatClientTCP(client_ns, "test%d" % i, node=node)
                self.add_cleanup(c.delete)
                c.wait_ready()
                clients.append(c)
            c1 = clients[0]
            c2 = clients[1]
            c3 = clients[2]
            _log.info("test_max_hops_namespace_annotation: created client pods [%s, %s, %s]", c1.ip, c2.ip, c3.ip)

            retry_until_success(self.has_ip_rule, retries=3, wait_time=3, function_kwargs={"nodename": c3.nodename, "ip": c3.ip})

            def verify_tables_and_hops():
                node_rules_and_tables = self.read_client_hops_for_node(node)
                assert len(node_rules_and_tables) == 3
                print(node_rules_and_tables)
                # Verify we have 3 different tables.
                table1 = node_rules_and_tables[c1.ip]["table"]
                table2 = node_rules_and_tables[c2.ip]["table"]
                table3 = node_rules_and_tables[c3.ip]["table"]
                print(table1, table2, table3)
                table_set = {table1, table2, table3}
                assert len(table_set) == 3

                hops1 = sorted(node_rules_and_tables[c1.ip]["hops"])
                hops2 = sorted(node_rules_and_tables[c2.ip]["hops"])
                hops3 = sorted(node_rules_and_tables[c3.ip]["hops"])
                return hops1, hops2, hops3

            hops1, hops2, hops3 = verify_tables_and_hops()
            # Verify each table has different hops.
            assert (hops1 != hops2) and (hops2 != hops3) and (hops1 != hops3)

            # Delete one gateway, only two hops left
            pod = gw1
            _log.info("Removing gateway pod %s", pod.name)
            self.delete_and_confirm(pod.name, "pod", pod.ns)
            self.cleanups.remove(pod.delete)

            def check_hops():
                # We should see same set of hops for each client since each table should has at least two hops.
                hops1, hops2, hops3 = verify_tables_and_hops()
                assert (hops1 == hops2) and (hops2 == hops3)
            retry_until_success(check_hops)

    def test_reuse_valid_table_on_restart(self):
        with DiagsCollector():
            _log.info("--- Restarting calico/node with routeTableRage 1,200 ---")
            oldEnv = {"FELIX_ROUTETABLERANGES": "201-250"}
            newEnv = {"FELIX_ROUTETABLERANGES": "1-200"}
            self.update_ds_env("calico-node", "kube-system", newEnv)

            def undo_route_table_range():
                self.update_ds_env("calico-node", "kube-system", {"FELIX_ROUTETABLERANGES": "1-250"})
            self.add_cleanup(undo_route_table_range)

            # Create 3 egress gateways, with an IP from that pool.
            gw1 = self.create_gateway_pod("kind-worker", "gw1", self.egress_cidr)
            gw2 = self.create_gateway_pod("kind-worker2", "gw2", self.egress_cidr)
            gw3 = self.create_gateway_pod("kind-worker3", "gw3", self.egress_cidr)
            for g in [gw1, gw2, gw3]:
                g.wait_ready()
            _log.info("test_max_hops: created gw pods [%s, %s, %s]", gw1.ip, gw2.ip, gw3.ip)

            # Create three clients on the same node with maxNextHops set to 2.
            client_ns = "default"
            node = "kind-worker"
            clients = []
            for i in range(3):
                _log.info("test_max_hops: create client%d", i)
                c = NetcatClientTCP(client_ns, "test%d" % i, node=node, annotations={
                    "egress.projectcalico.org/maxNextHops": "2",
                    "egress.projectcalico.org/selector": "color == 'red'",
                    "egress.projectcalico.org/namespaceSelector": "all()",
                })
                self.add_cleanup(c.delete)
                c.wait_ready()
                clients.append(c)
            c1 = clients[0]
            c2 = clients[1]
            c3 = clients[2]
            _log.info("test_max_hops: created client pods [%s, %s, %s]", c1.ip, c2.ip, c3.ip)

            retry_until_success(self.has_ip_rule, retries=3, wait_time=3, function_kwargs={"nodename": c3.nodename, "ip": c3.ip})

            def verify_tables_and_hops():
                node_rules_and_tables = self.read_client_hops_for_node(node)
                assert len(node_rules_and_tables) == 3
                print(node_rules_and_tables)
                # Verify we have 3 different tables.
                table1 = node_rules_and_tables[c1.ip]["table"]
                table2 = node_rules_and_tables[c2.ip]["table"]
                table3 = node_rules_and_tables[c3.ip]["table"]
                print(table1, table2, table3)
                table_set = {table1, table2, table3}
                assert len(table_set) == 3
                assert (int(table1) <= 200) and (int(table2) <= 200) and (int(table3) <= 200)
                return table1, table2, table3

            table1, table2, table3 = verify_tables_and_hops()

            def customise_ip_rule_and_table(node, src, current_table, new_table, hop1, hop2):
                run("docker exec %s ip rule add priority 100 from %s fwmark 0x80000/0x80000 lookup %s" % (node, src, new_table))
                if hop1 == "":
                    raise Exception('wrong parameters passed for customise_ip_rule_table')
                if hop2 == "":
                    run("docker exec %s ip route add table %s default nexthop via %s dev egress.calico onlink" % (node, new_table, hop1))
                else:
                    run("docker exec %s ip route add table %s default nexthop via %s dev egress.calico onlink nexthop via %s dev egress.calico onlink" % (node, new_table, hop1, hop2))
                # Delete current ip rule, so that felix would not pick it up again after restarting.
                run("docker exec %s ip rule del from %s fwmark 0x80000/0x80000 lookup %s" % (node, src, current_table))


            # Create ip rule for c3 and table outside of 1-200 so that it would not be managed by felix.
            # The new range will be 201-250, but start creating table at 211 to allow for some tables to be used by other
            # features. Egress Gateway won't get the very bottom of the table range specified.
            # The new table has valid hops.
            customise_ip_rule_and_table(node, c3.ip, table3, "213", gw2.ip, gw3.ip)
            # Create ip rule for c2 and table outside of 1-200 so that it would not be managed by felix.
            # The new table has invalid number of hops.
            customise_ip_rule_and_table(node, c2.ip, table2, "212", gw1.ip, "")
            # Create ip rule for c1 and table outside of 1-200 so that it would not be managed by felix.
            # The new table has invalid ip for one of its' hops.
            customise_ip_rule_and_table(node, c1.ip, table1, "211", gw2.ip, "10.10.0.0")

            run("docker exec %s ip rule" % node)
            run("docker exec %s ip route show table %s" % (node, "213"))
            run("docker exec %s ip route show table %s" % (node, "212"))
            run("docker exec %s ip route show table %s" % (node, "211"))

            _log.info("--- Restarting calico/node with routeTableRage 201-250 ---")
            self.update_ds_env("calico-node", "kube-system", oldEnv)

            retry_until_success(self.has_ip_rule, retries=3, wait_time=3, function_kwargs={"nodename": c3.nodename, "ip": c3.ip})

            run("docker exec %s ip rule" % node)

            node_rules_and_tables = self.read_client_hops_for_node(node, range(201, 251))
            assert len(node_rules_and_tables) == 3
            # Verify we have 3 different tables.
            # Two tables have latest indexes and table3 is 203.
            table1 = node_rules_and_tables[c1.ip]["table"]
            table2 = node_rules_and_tables[c2.ip]["table"]
            table3 = node_rules_and_tables[c3.ip]["table"]
            assert {table1, table2, table3} == {"250", "249", "213"}

            hops1 = sorted(node_rules_and_tables[c1.ip]["hops"])
            hops2 = sorted(node_rules_and_tables[c2.ip]["hops"])
            hops3 = sorted(node_rules_and_tables[c3.ip]["hops"])
            assert (hops1 != hops2) and (hops2 != hops3) and (hops1 != hops3)

            # Cleanup manually added tables. ip rules should have been cleaned up already by felix.
            run("docker exec %s ip route flush table %s" % (node, "213"))
            run("docker exec %s ip route flush table %s" % (node, "212"))
            run("docker exec %s ip route flush table %s" % (node, "211"))

    def has_ip_rule(self, nodename, ip):
        # Validate egress ip rule exists for a client pod ip on a node.
        output = run("docker exec -t %s ip rule" % nodename)
        if output.find(ip) == -1:
            raise Exception('ip rule does not exist for client pod ip %s, log %s' % (ip, output))

    def has_ip_route_and_table(self, nodename, client_ip, gateway_ips):
        node_rules_and_tables = self.read_client_hops_for_node(nodename)
        if client_ip in node_rules_and_tables:
            rule_and_table = node_rules_and_tables[client_ip]
            table = rule_and_table["table"]
            hops = rule_and_table["hops"]
            _log.info("found rule with ip: %s, pointing to table: %s, with hops: [%s], was looking for hops: [%s]", client_ip, table, hops, gateway_ips)
            assert set(hops) == set(gateway_ips)
        else:
            stop_for_debug()
            raise Exception("rule and table not found for client with ip %s and hops %s" % (client_ip, gateway_ips))

    def read_client_hops_for_node(self, nodename, table_range=None):
        # Read client hops for a node
        rule_dict = {}
        output = run("docker exec -t %s ip rule" % nodename)
        for l in output.splitlines():
            if (l.find("fwmark") != -1) and (l.find("from all fwmark") == -1):
                # read routing rule, i.e. "100:    from 192.168.162.159 fwmark 0x80000/0x80000 lookup 250"
                _log.info("read_client_hops_for_node: %s", l)
                src = re.search(r'\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}', l).group()
                table = re.search(r'(\d+)\D*$', l).group(1)
                # In the test_reuse_valid_table_on_restart testcase, there will be multiple routing tables per
                # workload. In that case, we only care about tables in the currently configured Felix table range.
                if (not (table_range is None)) and (not (int(table) in set(table_range))):
                    _log.info("read_client_hops_for_node: ignoring table %s, not in range %s", table, table_range)
                    continue
                table_output = run("docker exec -t %s ip route show table %s" % (nodename, table))
                _log.info("read_client_hops_for_node: src %s to table %s [%s]", src, table, table_output)

                # read route table content
                hops = []
                for l in table_output.splitlines():
                    if l.find("egress.calico") != -1:
                        # "nexthop via 10.10.10.0 dev egress.calico weight 1 onlink"
                        # or
                        # "default via 10.10.10.0 dev egress.calico onlink"
                        _log.info("read_client_hops_for_node: %s", l)
                        hop = re.search(r'\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}', l).group()
                        hops.append(hop)
                rule_dict[src] = {"table": table, "hops": hops}
        print(rule_dict)
        return rule_dict

    def check_ecmp_routes(self, client, servers, gw_ips, allowed_untaken_count=0):
        """
        Validate that client went through every ECMP route when
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
        for s in servers * 5:
            _log.info("Checking can-connect, Client IP: %s Server IP: %s Port: %d", client.ip, s.ip, s.port)
            retry_until_success(client.can_connect, retries=3, wait_time=1, function_kwargs={"ip": s.ip, "port": s.port})
            # Check the source IP as seen by the server.
            client_ip = s.get_recent_client_ip()
            _log.info("xxxxx ecmp route xxxxxxxxx   Client IPs: %r", client_ip)
            if client_ip not in gw_ips:
                _log.error("Got unexpected client IP %s (expected %s); collecting routing tables before failure...", client_ip, gw_ips)
                run("docker exec -t %s ip rule" % client.nodename)
                run("docker exec -t %s ip route show table 250" % client.nodename, allow_fail=True)
                run("docker exec -t %s ip route show table 249" % client.nodename, allow_fail=True)
                run("docker exec -t %s ip route show table 248" % client.nodename, allow_fail=True)
                run("docker exec -t %s ip route show table 247" % client.nodename, allow_fail=True)
                run("docker exec -t %s ip route show table 246" % client.nodename, allow_fail=True)
                run("docker exec -t %s ip route show table 245" % client.nodename, allow_fail=True)
                _log.info("stop for debug ecmp route %s  Client IPs: %r", gw_ips, client_ip)
                stop_for_debug()
            assert client_ip in gw_ips, \
                "client ip %s is not one of gateway ips.%s" % (client_ip, gw_ips)

            if client_ip in expected_ips:
                expected_ips.remove(client_ip)

            if len(expected_ips) == 0:
                break

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

        # Create egress gateway, with an IP from that pool.  Configure it to ping the server.
        gateway = self.create_gateway_pod(gateway_node, "gw", self.egress_cidr, icmp_probes=server.ip)

        client_ns = "default"
        client = NetcatClientTCP(client_ns, "test1", node="kind-worker", labels={"app": "client"}, annotations={
            "egress.projectcalico.org/selector": "color == 'red'",
            "egress.projectcalico.org/namespaceSelector": "all()",
        })
        self.add_cleanup(client.delete)
        client.wait_ready()

        # Give the server a route back to the egress IP.
        self.server_add_route(server, gateway)
        gateway.wait_ready()

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

    def server_del_route(self, server, pod):
        """
        delete route to a pod for a server
        """
        server.execute("ip r del %s/32 via %s" % (pod.ip, pod.hostip))

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

    def create_gateway_pod(self, host, name, egress_cidr, color="red", ns="default", termgraceperiod=0, probe_url="", icmp_probes=""):
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
  initContainers:
  - name: egress-gateway-init
    image: docker.io/tigera/egress-gateway:latest-amd64
    env:
    - name: EGRESS_POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    imagePullPolicy: Never
    securityContext:
      privileged: true
    command: ["/init-gateway.sh"]
  containers:
  - name: gateway
    image: docker.io/tigera/egress-gateway:latest-amd64
    env:
    # Optional: comma-delimited list of IP addresses to send ICMP pings to; if all probes fail, the egress
    # gateway will report non-ready.
    - name: ICMP_PROBE_IPS
      value: "%s"
    # Only used if ICMP_PROBE_IPS is non-empty: interval to send probes.
    - name: ICMP_PROBE_INTERVAL
      value: "1s"
    # Only used if ICMP_PROBE_IPS is non-empty: timeout on each probe.
    - name: ICMP_PROBE_TIMEOUT
      value: "3s"
    # Optional HTTP URL to send periodic probes to; if the probe fails that is reflected in 
    # the health reported on the health port.
    - name: HTTP_PROBE_URL
      value: "%s"
    # Only used if HTTP_PROBE_URL is non-empty: interval to send probes.
    - name: HTTP_PROBE_INTERVAL
      value: "10s"
    # Only used if HTTP_PROBE_URL is non-empty: timeout before reporting non-ready if there are no successful probes.
    - name: HTTP_PROBE_TIMEOUT
      value: "30s"
    # Port that the egress gateway serves its health reports.  Must match the readiness probe and health
    # port defined below.
    - name: HEALTH_PORT
      value: "8080"
    # Use downward API to tell the pod its own IP address.
    - name: EGRESS_POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    imagePullPolicy: Never
    securityContext:
      capabilities:
        add: ["NET_ADMIN"]
    command: ["/start-gateway.sh"]
    volumeMounts:
        - mountPath: /var/run
          name: policysync
    ports:
        - name: health
          containerPort: 8080
    readinessProbe:
        httpGet:
          path: /readiness
          port: 8080
        initialDelaySeconds: 3
        periodSeconds: 3
  nodeName: %s
  terminationGracePeriodSeconds: %d
  volumes:
      - flexVolume:
          driver: nodeagent/uds
        name: policysync
""" % (egress_cidr, color, name, ns, icmp_probes, probe_url, host, termgraceperiod))
        self.add_cleanup(gateway.delete)

        return gateway

_TestEgressIP.vanilla = False
_TestEgressIP.egress_ip = True


class NetcatServerTCP(Container):

    def __init__(self, port):
        super(NetcatServerTCP, self).__init__("subfuzion/netcat", "-v -l -k -p %d" % port, "--privileged")
        self.port = port

    def get_recent_client_ip(self):
        ip = None
        for attempt in range(3):
            for line in self.logs().split('\n'):
                m = re.match(r"Connection from ([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+) [0-9]+ received", line)
                if m:
                    ip = m.group(1)
            if ip is not None:
                return ip
            else:
                time.sleep(1)
        assert False, "Couldn't find a recent client IP in the logs."

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
