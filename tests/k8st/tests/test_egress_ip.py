# Copyright (c) 2020 Tigera, Inc. All rights reserved.

import logging
import os
import re
import time

from tests.k8st.test_base import Container, Pod, TestBase
from tests.k8st.utils.utils import DiagsCollector, calicoctl, kubectl, run, node_info

_log = logging.getLogger(__name__)


class TestEgressIP(TestBase):

    def setUp(self):
        super(TestEgressIP, self).setUp()

    def tearDown(self):
        super(TestEgressIP, self).tearDown()

    def test_egress_ip_mainline(self):

        with DiagsCollector():

            # Enable egress IP.
            newEnv = {"FELIX_EGRESSIPENABLED": "true"}
            self.update_ds_env("calico-node", "kube-system", newEnv)

            # Create external server.
            server_port = 8089
            server = NetcatServerTCP(server_port)
            self.add_cleanup(server.kill)
            server.wait_running()

            # Create egress IP pool.
            egress_cidr = "10.10.10.0/29"
            calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: egress-ippool-1
spec:
  cidr: %s
  blockSize: 29
EOF
""" % egress_cidr)
            self.add_cleanup(lambda: calicoctl("delete ippool egress-ippool-1"))

            # Create egress gateway, with an IP from that pool.
            gateway_ns = "default"
            gateway = Pod(gateway_ns, "gateway", image=None, yaml="""
apiVersion: v1
kind: Pod
metadata:
  annotations:
    cni.projectcalico.org/ipv4pools: "[\\\"%s\\\"]"
  labels:
    color: red
  name: gateway
  namespace: %s
spec:
  containers:
  - name: gateway
    image: songtjiang/gateway:ubi
    env:
    - name: EGRESS_POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    imagePullPolicy: Always
    securityContext:
      privileged: true
  nodeName: kind-worker2
  terminationGracePeriodSeconds: 0
""" % (egress_cidr, gateway_ns))
            self.add_cleanup(gateway.delete)
            gateway.wait_ready()

            # Temporary hack: add an address to the host-side veth for the egress gateway pod.
            #
            # First, find the interface name.
            if_name = run("docker exec -it kind-worker2 bash -c" +
                          " 'ip r | grep \"%s dev cali\" | cut -d\" \" -f3'" % gateway.ip).strip()
            # Now add 169.254.1.1 address to that interface.
            run("docker exec -it kind-worker2 ip a a 169.254.1.1/16 dev %s" % if_name)

            # Create client.
            client_ns = "default"
            client = NetcatClientTCP(client_ns, "test1", annotations={
                "egress.projectcalico.org/selector": "color == 'red'",
                "egress.projectcalico.org/namespaceSelector": "all()",
            })
            self.add_cleanup(client.delete)
            client.wait_ready()

            # Give the server a route back to the egress IP.
            nodes, ips, _ = node_info()
            for i in range(len(nodes)):
                if nodes[i] == "kind-worker2":
                    server.execute("ip r a %s/32 via %s" % ("10.10.10.0", ips[i]))

            _log.info("Client IP: %s Server IP: %s Port: %d", client.ip, server.ip, server_port)
            client.connect(server.ip, server_port)

            # Check the source IP as seen by the server.
            client_ips = server.get_client_ips()
            _log.info("Client IPs: %r", client_ips)
            self.assertListEqual(client_ips, [gateway.ip])


TestEgressIP.vanilla = True
TestEgressIP.dual_stack = False


class NetcatServerTCP(Container):

    def __init__(self, port):
        super(NetcatServerTCP, self).__init__("subfuzion/netcat", "-v -l -k -p %d" % port, "--privileged")

    def get_client_ips(self):
        ips = []
        for line in self.logs().split('\n'):
            m = re.match(r"Connection from ([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+) [0-9]+ received", line)
            if m:
                ips.append(m.group(1))
        return ips


class NetcatClientTCP(Pod):

    def __init__(self, ns, name, annotations=None):
        super(NetcatClientTCP, self).__init__(ns, name, image="laurenceman/alpine", annotations=annotations)

    def connect(self, ip, port):
        self.execute("nc %s %d </dev/null" % (ip, port))
