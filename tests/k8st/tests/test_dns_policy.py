# Copyright (c) 2019 Tigera, Inc. All rights reserved.

import logging

from tests.k8st.test_base import TestBase
from tests.k8st.utils.utils import retry_until_success, kubectl, calicoctl

_log = logging.getLogger(__name__)


class TestDNSPolicy(TestBase):

    def setUp(self):
        super(TestDNSPolicy, self).setUp()

        # Create bgp test namespace
        self.ns = "dns-policy-test"
        self.create_namespace(self.ns)
        self.test1 = "test-1 -n " + self.ns

    def tearDown(self):
        super(TestDNSPolicy, self).tearDown()
        self.delete_and_confirm(self.ns, "ns")

        # Clean up policy.
        calicoctl("delete gnp default.allow-egress-to-domain || true")
        calicoctl("delete gnp default.deny-all-egress-except-dns || true")
        calicoctl("get gnp")

    def deny_all_egress_except_dns(self, selector):
        # Deny egress from selected pods, except for DNS.
        calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: deny-all-egress-except-dns
spec:
  selector: %s
  types:
  - Egress
  egress:
  - action: Allow
    protocol: UDP
    destination:
      ports:
      - 53
  - action: Deny
EOF
""" % selector)

    def allow_egress_to_domains(self, pod_selector, domains):
        domain_string = """
      domains:"""
        for domain in domains:
            domain_string = domain_string + """
      - %s""" % domain

        calicoctl("""apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: allow-egress-to-domain
spec:
  order: 1
  selector: "%s"
  types:
  - Egress
  egress:
  - action: Allow
    destination:%s
EOF
""" % (pod_selector, domain_string))

    def test_internet_service(self):
        kubectl("run " + self.test1 + " --generator=run-pod/v1 " +
                "--image=laurenceman/alpine --labels=\"egress=restricted\"")
        kubectl("wait --for=condition=ready pod/%s" % self.test1)

        def should_connect():
            kubectl("exec " + self.test1 + " -- " +
                    "curl --connect-timeout 3 -i -L microsoft.com")

        def should_not_connect():
            try:
                should_connect()
            except Exception:
                return
            raise Exception("should not have connected")

        # No policy.
        should_connect()

        # Deny all egress.
        self.deny_all_egress_except_dns("egress == 'restricted'")
        retry_until_success(should_not_connect, retries=2)

        # DNS policy.
        self.allow_egress_to_domains("egress == 'restricted'",
                                     ["microsoft.com", "www.microsoft.com"])
        retry_until_success(should_connect, retries=2)
