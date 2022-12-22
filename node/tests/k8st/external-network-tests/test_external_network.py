# Copyright (c) 2022 Tigera, Inc. All rights reserved.

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

class TestExternalNetwork(TestBase):
    def setUp(self):
        super(TestExternalNetwork, self).setUp()

    def tearDown(self):
        super(TestExternalNetwork, self).tearDown()

    def _getCalicoNodePod(self, nodeName):
        calicoPod = kubectl("-n kube-system get pods -o wide | grep calico-node | grep '%s '| cut -d' ' -f1" % nodeName)
        if calicoPod is not None:
            calicoPod = calicoPod.strip()
        return calicoPod

    def _findRouteInClusterBird(self, calicoPod, route, peerIP, ipv6=False):
        birdCmd = "birdcl6" if ipv6 else "birdcl"
        birdPeer = "Node_" + peerIP.replace(".", "_").replace(":","_")
        routes = kubectl("exec -n kube-system %s -- %s show route protocol %s" % (calicoPod, birdCmd, birdPeer))
        # Calico seems to use a link-local address for 'via'
        peerIPRegex = "fe80::.*" if ipv6 else re.escape(peerIP)
        result = re.search("%s *via %s on .* \[%s" % (re.escape(route), peerIPRegex, birdPeer), routes)
        return result

    def _findRouteInExternalBird(self, birdContainer, birdPeer, routeRegex, peerIP, ipv6=False):
        birdCmd = "birdcl6" if ipv6 else "birdcl"
        routes = run("docker exec %s %s show route protocol %s" % (birdContainer, birdCmd, birdPeer))
        result = re.search("%s *via %s on .* \[%s" % (routeRegex, re.escape(peerIP), birdPeer), routes)
        return result

    def test_bgp_filter_basic_export_import_v4(self):
        with DiagsCollector():
            # Add static route bird config
            run("""cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/static-route.conf"
protocol static static1 {
    route 10.111.111.0/24 via 172.31.11.2;
    export all;
}
EOF
""")
            self.add_cleanup(lambda: run("docker exec bird-a1 sh -c 'rm /etc/bird/static-route.conf; birdcl configure'"))

            # Tell bird to reload config
            run("docker exec bird-a1 birdcl configure")

            # Figure out which calico-node pod runs on kind-worker node
            calicoPod = self._getCalicoNodePod("kind-worker")
            assert calicoPod is not None

            # Check that the kind-worker node has the route advertised by the external bird instance
            result = self._findRouteInClusterBird(calicoPod, "10.111.111.0/24", "172.31.11.1")
            assert result is not None

            # Check that the external bird instance bird-a1 has a route for an IPAM block from the default IP pool
            result = self._findRouteInExternalBird("bird-a1", "node1", "192\.168\.\d+\.\d+/\d+", "172.31.11.4")
            assert result is not None

            # Add BGPFilter with import rule
            kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-export-1
spec:
  exportV4:
  - cidr: 192.168.0.0/16
    matchOperator: In
    action: Reject
EOF
""")
            self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-export-1"))

            patchStr = "{\"spec\": {\"filters\": [\"test-filter-export-1\"]}}"
            kubectl("patch bgppeer peer-a1 --patch '%s'" % patchStr)
            self.add_cleanup(lambda: kubectl("patch bgppeer peer-a1 --patch '{\"spec\": {\"filters\": []}}'"))

            result = self._findRouteInExternalBird("bird-a1", "node1", "192\.168\.\d+\.\d+/\d+", "172.31.11.4")
            assert result is None

            # Add BGPFilter with export rule
            kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-import-1
spec:
  importV4:
  - cidr: 10.111.111.0/24
    matchOperator: Equal
    action: Reject
EOF
""")
            self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-import-1"))

            patchStr = "{\"spec\": {\"filters\": [\"test-filter-import-1\"]}}"
            kubectl("patch bgppeer peer-a1 --patch '%s'" % patchStr)
            self.add_cleanup(lambda: kubectl("patch bgppeer peer-a1 --patch '{\"spec\": {\"filters\": []}}'"))

            result = self._findRouteInClusterBird(calicoPod, "10.111.111.0/24", "172.31.11.1")
            assert result is None

    def test_bgp_filter_basic_export_import_v6(self):
        with DiagsCollector():
            # Add static route bird config
            run("""cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird6/static-route.conf"
protocol static static1 {
    route fd00:1111:1111:1111::/64 via fd00:0:1234:1111::2;
    export all;
}
EOF
""")
            self.add_cleanup(lambda: run("docker exec bird-a1 sh -c 'rm /etc/bird6/static-route.conf; birdcl6 configure'"))

            # Tell bird to reload config
            run("docker exec bird-a1 birdcl6 configure")

            # Figure out which calico-node pod runs on kind-worker node
            calicoPod = self._getCalicoNodePod("kind-worker")
            assert calicoPod is not None

            # Check that the kind-worker node has the route advertised by the external bird instance
            result = self._findRouteInClusterBird(calicoPod, "fd00:1111:1111:1111::/64", "fd00:0:1234:1111::1", ipv6=True)
            assert result is not None

            # Check that the external bird instance bird-a1 has a route for an IPAM block from the default IP pool
            result = self._findRouteInExternalBird("bird-a1", "node1", "fd00:10:244:.*/\d+", "fd00:0:1234:1111::4", ipv6=True)
            assert result is not None

            # Add BGPFilter with import rule
            kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-export-1
spec:
  exportV6:
  - cidr: fd00:10:244::/64
    matchOperator: In
    action: Reject
EOF
""")
            self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-export-1"))

            patchStr = "{\"spec\": {\"filters\": [\"test-filter-export-1\"]}}"
            kubectl("patch bgppeer peer-a1-v6 --patch '%s'" % patchStr)
            self.add_cleanup(lambda: kubectl("patch bgppeer peer-a1-v6 --patch '{\"spec\": {\"filters\": []}}'"))

            result = self._findRouteInExternalBird("bird-a1", "node1", "fd00:10:244:.*/\d+", "fd00:0:1234:1111::4", ipv6=True)
            assert result is None

            # Add BGPFilter with export rule
            kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-import-1
spec:
  importV6:
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: Equal
    action: Reject
EOF
""")
            self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-import-1"))

            patchStr = "{\"spec\": {\"filters\": [\"test-filter-import-1\"]}}"
            kubectl("patch bgppeer peer-a1-v6 --patch '%s'" % patchStr)
            self.add_cleanup(lambda: kubectl("patch bgppeer peer-a1-v6 --patch '{\"spec\": {\"filters\": []}}'"))

            result = self._findRouteInClusterBird(calicoPod, "fd00:1111:1111:1111::/64", "fd00:0:1234:1111::1", ipv6=True)
            assert result is None
