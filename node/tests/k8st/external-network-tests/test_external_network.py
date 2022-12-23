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
        """Get the calico-node pod name for a given kind node"""
        def fn():
            calicoPod = kubectl("-n kube-system get pods -o wide | grep calico-node | grep '%s '| cut -d' ' -f1" % nodeName)
            if calicoPod is None:
                raise Exception('calicoPod is None')
            return calicoPod.strip()
        calicoPod = retry_until_success(fn)
        return calicoPod

    def _checkRouteInClusterBird(self, calicoPod, route, peerIP, ipv6=False, present=True):
        """Check that a route is present/not present in a (in-cluster) calico-node bird instance"""
        def fn():
            birdCmd = "birdcl6" if ipv6 else "birdcl"
            birdPeer = "Node_" + peerIP.replace(".", "_").replace(":","_")
            routes = kubectl("exec -n kube-system %s -- %s show route protocol %s" % (calicoPod, birdCmd, birdPeer))
            # Calico seems to use a link-local address for 'via'
            peerIPRegex = "fe80::.*" if ipv6 else re.escape(peerIP)
            result = re.search("%s *via %s on .* \[%s" % (re.escape(route), peerIPRegex, birdPeer), routes)
            if result is None and present:
                raise Exception('route not present when it should be')
            if result is not None and not present:
                raise Exception('route present when it should not be')
            return result
        result = retry_until_success(fn, wait_time=3)
        return result

    def _checkRouteInExternalBird(self, birdContainer, birdPeer, routeRegex, peerIP, ipv6=False, present=True):
        """Check that a route is present/not present in an external (plaing docker container) bird instance"""
        def fn():
            birdCmd = "birdcl6" if ipv6 else "birdcl"
            routes = run("docker exec %s %s show route protocol %s" % (birdContainer, birdCmd, birdPeer))
            result = re.search("%s *via %s on .* \[%s" % (routeRegex, re.escape(peerIP), birdPeer), routes)
            if result is None and present:
                raise Exception('route not present when it should be')
            if result is not None and not present:
                raise Exception('route present when it should not be')
            return result
        result = retry_until_success(fn, wait_time=3)
        return result

    def _patchPeerFilters(self, peer, filters):
        """Patch BGPFilters in a BGPPeer"""
        filterStr = "\"" + "\", \"".join(filters) + "\"" if len(filters) > 0 else ""
        patchStr = "{\"spec\": {\"filters\": [%s]}}" % filterStr
        kubectl("patch bgppeer %s --patch '%s'" % (peer, patchStr))

    def _test_bgp_filter_basic(self, ipv4, ipv6):
        """Basic test case:
        - Add IPv4/IPv6 route to the external bird instance
        - Verify it is present in cluster bird instance (to validate import)
        - Verify that IPAM block from IP pool is present in external bird instance (to validate export)
        - Add BGPFilter with export reject rule and verify that route is no longer present in external bird instance
        - Add BGPFilter with import reject rule and verify that route is no longer present in cluster bird instance
        """
        with DiagsCollector():
            # Add static route bird config
            if ipv4:
                run("""cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/static-route.conf"
protocol static static1 {
    route 10.111.111.0/24 via 172.31.11.2;
    export all;
}
EOF
""")
                self.add_cleanup(lambda: run("docker exec bird-a1 sh -c 'rm /etc/bird/static-route.conf; birdcl configure'"))

                run("docker exec bird-a1 birdcl configure")
            if ipv6:
                run("""cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird6/static-route.conf"
protocol static static1 {
    route fd00:1111:1111:1111::/64 via fd00:0:1234:1111::2;
    export all;
}
EOF
""")
                self.add_cleanup(lambda: run("docker exec bird-a1 sh -c 'rm /etc/bird6/static-route.conf; birdcl6 configure'"))

                run("docker exec bird-a1 birdcl6 configure")

            # Figure out which calico-node pod runs on kind-worker node
            calicoPod = self._getCalicoNodePod("kind-worker")

            # Check that the kind-worker node has the route advertised by the external bird instance
            if ipv4:
                result = self._checkRouteInClusterBird(calicoPod, "10.111.111.0/24", "172.31.11.1")
            if ipv6:
                result = self._checkRouteInClusterBird(calicoPod, "fd00:1111:1111:1111::/64", "fd00:0:1234:1111::1", ipv6=True)

            # Check that the external bird instance bird-a1 has a route for an IPAM block from the default IP pool
            if ipv4:
                result = self._checkRouteInExternalBird("bird-a1", "node1", "192\.168\.\d+\.\d+/\d+", "172.31.11.4")
            if ipv6:
                result = self._checkRouteInExternalBird("bird-a1", "node1", "fd00:10:244:.*/\d+", "fd00:0:1234:1111::4", ipv6=True)

            # Add BGPFilter with export rule
            if ipv4:
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
            if ipv6:
                kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-export-v6-1
spec:
  exportV6:
  - cidr: fd00:10:244::/64
    matchOperator: In
    action: Reject
EOF
""")
                self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-export-v6-1"))


            if ipv4:
                self._patchPeerFilters("peer-a1", ["test-filter-export-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1", []))
            if ipv6:
                self._patchPeerFilters("peer-a1-v6", ["test-filter-export-v6-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1-v6", []))

            if ipv4:
                result = self._checkRouteInExternalBird("bird-a1", "node1", "192\.168\.\d+\.\d+/\d+", "172.31.11.4", present=False)
            if ipv6:
                result = self._checkRouteInExternalBird("bird-a1", "node1", "fd00:10:244:.*/\d+", "fd00:0:1234:1111::4", ipv6=True, present=False)

            # Add BGPFilter with import rule
            if ipv4:
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
            if ipv6:
                kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-import-v6-1
spec:
  importV6:
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: Equal
    action: Reject
EOF
""")
                self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-import-v6-1"))

            if ipv4:
                self._patchPeerFilters("peer-a1", ["test-filter-import-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1", []))
            if ipv6:
                self._patchPeerFilters("peer-a1-v6", ["test-filter-import-v6-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1-v6", []))


            if ipv4:
                result = self._checkRouteInClusterBird(calicoPod, "10.111.111.0/24", "172.31.11.1", present=False)
            if ipv6:
                result = self._checkRouteInClusterBird(calicoPod, "fd00:1111:1111:1111::/64", "fd00:0:1234:1111::1", ipv6=True, present=False)

    def test_bgp_filter_basic_v4(self):
        self._test_bgp_filter_basic(True, False)

    def test_bgp_filter_basic_v6(self):
        self._test_bgp_filter_basic(False, True)

    def test_bgp_filter_basic_v4v6(self):
        self._test_bgp_filter_basic(True, True)

    def test_bgp_filter_validation(self):
        with DiagsCollector():
            # Filter with various invalid fields
            output = kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-invalid-filter
spec:
  importV4:
  - cidr: 10.111.111.0/24
    matchOperator: notin
    action: accept
  - cidr: 10.222.222.0/24
    matchOperator: equal
    action: Retecj
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: Equal
    action: Accept
  exportV4:
  - cidr: 10.111.111.0/24
    matchOperator: notin
    action: Accetp
  - cidr: 10.222.222.0/24
    matchOperator: in
    action: Accept
  - cidr: IPv4Address
    matchOperator: In
    action: Accept
  importV6:
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: Eqaul
    action: accept
  - cidr: 10.111.111.0/24
    matchOperator: In
    action: Accept
  exportV6:
  - cidr: fd00:2222:2222:2222::/64
    matchOperator: notequal
    action: reject
  - cidr: ipv6Address
    matchOperator: Equal
    action: Reject
EOF
""", allow_fail=True, returnerr=True)

            if output is not None:
                output = output.strip()

            expectedOutput = """The BGPFilter "test-invalid-filter" is invalid: 
* MatchOperator: Invalid value: "notin": Reason: failed to validate Field: MatchOperator because of Tag: matchOperator 
* Action: Invalid value: "Accetp": Reason: failed to validate Field: Action because of Tag: filterAction 
* MatchOperator: Invalid value: "in": Reason: failed to validate Field: MatchOperator because of Tag: matchOperator 
* CIDR: Invalid value: "IPv4Address": Reason: failed to validate Field: CIDR because of Tag: netv4 
* Action: Invalid value: "accept": Reason: failed to validate Field: Action because of Tag: filterAction 
* MatchOperator: Invalid value: "equal": Reason: failed to validate Field: MatchOperator because of Tag: matchOperator 
* Action: Invalid value: "Retecj": Reason: failed to validate Field: Action because of Tag: filterAction 
* CIDR: Invalid value: "fd00:1111:1111:1111::/64": Reason: failed to validate Field: CIDR because of Tag: netv4 
* MatchOperator: Invalid value: "notequal": Reason: failed to validate Field: MatchOperator because of Tag: matchOperator 
* Action: Invalid value: "reject": Reason: failed to validate Field: Action because of Tag: filterAction 
* CIDR: Invalid value: "ipv6Address": Reason: failed to validate Field: CIDR because of Tag: netv6 
* MatchOperator: Invalid value: "Eqaul": Reason: failed to validate Field: MatchOperator because of Tag: matchOperator 
* CIDR: Invalid value: "10.111.111.0/24": Reason: failed to validate Field: CIDR because of Tag: netv6"""
            assert output == expectedOutput


    def _test_bgp_filter_ordering(self, ipv4, ipv6):
        """Test multiple rules per filter and multiple filters per peer, as well as
        exhaust matchOperators and actions"""
        with DiagsCollector():
            # Add static route bird config
            if ipv4:
                run("""cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird/static-route.conf"
protocol static static1 {
    route 10.111.111.0/24 via 172.31.11.2;
    export all;
}
EOF
""")
                self.add_cleanup(lambda: run("docker exec bird-a1 sh -c 'rm /etc/bird/static-route.conf; birdcl configure'"))

                run("docker exec bird-a1 birdcl configure")
            if ipv6:
                run("""cat <<EOF | docker exec -i bird-a1 sh -c "cat > /etc/bird6/static-route.conf"
protocol static static1 {
    route fd00:1111:1111:1111::/64 via fd00:0:1234:1111::2;
    export all;
}
EOF
""")
                self.add_cleanup(lambda: run("docker exec bird-a1 sh -c 'rm /etc/bird6/static-route.conf; birdcl6 configure'"))

                run("docker exec bird-a1 birdcl6 configure")

            # Figure out which calico-node pod runs on kind-worker node
            calicoPod = self._getCalicoNodePod("kind-worker")

            # Add filters with multiple rules that should result in the routes being accepted
            if ipv4:
                kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-import-1
spec:
  importV4:
  - cidr: 10.111.0.0/16
    matchOperator: In
    action: Accept
  - cidr: 10.111.111.0/24
    matchOperator: Equal
    action: Reject
EOF
""")
                self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-import-1"))
            if ipv6:
                kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-import-v6-1
spec:
  importV6:
  - cidr: fd00:1111:1111::/48
    matchOperator: In
    action: Accept
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: Equal
    action: Reject
EOF
""")
                self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-import-v6-1"))

            if ipv4:
                self._patchPeerFilters("peer-a1", ["test-filter-import-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1", []))
            if ipv6:
                self._patchPeerFilters("peer-a1-v6", ["test-filter-import-v6-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1-v6", []))

            if ipv4:
                result = self._checkRouteInClusterBird(calicoPod, "10.111.111.0/24", "172.31.11.1", present=True)
            if ipv6:
                result = self._checkRouteInClusterBird(calicoPod, "fd00:1111:1111:1111::/64", "fd00:0:1234:1111::1", ipv6=True, present=True)

            # Add additional filters with multiple rules that should result in the routes being rejected
            if ipv4:
                kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-import-2
spec:
  importV4:
  - cidr: 10.111.0.0/16
    matchOperator: NotIn
    action: Accept
  - cidr: 10.111.111.0/24
    matchOperator: NotEqual
    action: Accept
  - cidr: 10.111.111.0/24
    matchOperator: Equal
    action: Reject
EOF
""")
                self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-import-2"))
            if ipv6:
                kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-import-v6-2
spec:
  importV6:
  - cidr: fd00:1111:1111::/48
    matchOperator: NotIn
    action: Accept
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: NotEqual
    action: Accept
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: Equal
    action: Reject
EOF
""")
                self.add_cleanup(lambda: kubectl("delete bgpfilter test-filter-import-v6-2"))

            if ipv4:
                self._patchPeerFilters("peer-a1", ["test-filter-import-2", "test-filter-import-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1", []))
            if ipv6:
                self._patchPeerFilters("peer-a1-v6", ["test-filter-import-v6-2", "test-filter-import-v6-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1-v6", []))

            if ipv4:
                result = self._checkRouteInClusterBird(calicoPod, "10.111.111.0/24", "172.31.11.1", present=False)
            if ipv6:
                result = self._checkRouteInClusterBird(calicoPod, "fd00:1111:1111:1111::/64", "fd00:0:1234:1111::1", ipv6=True, present=False)

            # Add an additional filter with both IPv4 and IPv6 rules that should result in the routes being accepted
            if ipv4 and ipv6:
                kubectl("""apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPFilter
metadata:
  name: test-filter-import-v4-v6
spec:
  importV4:
  - cidr: 10.111.111.0/24
    matchOperator: Equal
    action: Accept
  - cidr: 10.111.0.0/16
    matchOperator: NotIn
    action: Accept
  - cidr: 10.111.111.0/24
    matchOperator: NotEqual
    action: Accept
  - cidr: 10.111.111.0/24
    matchOperator: Equal
    action: Reject
  importV6:
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: Equal
    action: Accept
  - cidr: fd00:1111:1111::/48
    matchOperator: NotIn
    action: Accept
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: NotEqual
    action: Accept
  - cidr: fd00:1111:1111:1111::/64
    matchOperator: Equal
    action: Reject
EOF
""")

                self._patchPeerFilters("peer-a1", ["test-filter-import-v4-v6", "test-filter-import-2", "test-filter-import-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1", []))

                self._patchPeerFilters("peer-a1-v6", ["test-filter-import-v4-v6", "test-filter-import-v6-2", "test-filter-import-v6-1"])
                self.add_cleanup(lambda: self._patchPeerFilters("peer-a1-v6", []))

                result = self._checkRouteInClusterBird(calicoPod, "10.111.111.0/24", "172.31.11.1", present=True)

                result = self._checkRouteInClusterBird(calicoPod, "fd00:1111:1111:1111::/64", "fd00:0:1234:1111::1", ipv6=True, present=True)

    def test_bgp_filter_ordering_v4(self):
        self._test_bgp_filter_ordering(True, False)

    def test_bgp_filter_ordering_v6(self):
        self._test_bgp_filter_ordering(False, True)

    def test_bgp_filter_ordering_v4v6(self):
        self._test_bgp_filter_ordering(True, True)
