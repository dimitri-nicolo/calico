---
title: Using domain names in policy rules
canonical_url: 'https://docs.projectcalico.org/master/security/domain-based-policy'
---

### Big Picture

Use domain names to specify destinations outside the cluster to which
traffic should be allowed.

### Value

Because IPs often change or are hard to predict, using a domain name
is more convenient for identifying an external service.  And, because
{{site.prodname}} only trusts DNS information from a configurable set
of DNS servers, attackers cannot subvert this feature to gain access
to arbitrary external IPs.

### Features

This article covers the following {{site.prodname}} features:

- Allow egress traffic for specific domains using a [global network
  policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkpolicy).

- Use a [global network
  set]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkset)
  for allowed domains, and reference the network set in a [global
  network
  policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkpolicy).

- Support for DNS types: DNS A, AAAA, and CNAME records.

### Concepts

#### Deny all, allow some

Domain names can be used to specify destinations outside the cluster
to which traffic should be allowed.  This makes sense when egress
traffic from workloads is denied by default.  When you use domain
names in a global network policy, you are poking holes for allowed
traffic in an otherwise firewalled cluster.

> **Note:** Domain names do not work for *denying* traffic to
> particular destinations, or for services *inside* the cluster.
{: .alert .alert-info}

> **Note**: Kubernetes labels provide a similar convenience for
> services *within* the cluster.  {{site.prodname}} does not support
> using domain names for services *within* the cluster, because
> Kubernetes labels can and should be used for those instead.
{: .alert .alert-info}

#### How it works

When a client workload connects to a service by name, it does a DNS
lookup for the name.  {{site.prodname}} snoops the DNS exchange and
uses that information to open a pinhole to the service's current IP
address.

{{site.prodname}} trusts DNS information that comes only from a
configurable set of DNS servers.  Otherwise, attackers could use a
malicious workload to poison the system by sending DNS lookups to a
fake DNS server under their own control.

By default, {{site.prodname}} trusts the Kubernetes cluster's DNS
service (kube-dns or CoreDNS).  Thus domain name usage works out of
the box, unless you override pod DNS configurations.

The domain name feature is configured by [Felix configuration
parameters]({{site.baseurl}}/{{page.version}}/reference/felix/configuration)
(with "DNS" prefix).  The defaults are chosen to work out of the box
with standard Kubernetes installs, so you won’t normally change them.

### How to

There are two options for configuring allowed destination domain names:

- A
  [GlobalNetworkPolicy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkpolicy)
  can have egress rules with `action: Allow` and a
  `destination.domains` field specifying the domain names to which
  egress traffic is allowed.

- Allowed destination domain names can be specified in the
  `allowedEgressDomains` field of a
  [GlobalNetworkSet]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkset)
  resource, and GlobalNetworkPolicy can then use a
  `destination.selector` expression that matches that
  GlobalNetworkSet.

### Tutorial

The following example configures a default **global network policy**
for all pods, that denies all **egress** connections except for DNS:

```
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: deny-all-egress-except-dns
spec:
  selector: all()
  types:
  - Egress
  egress:
  - action: Allow
    protocol: UDP
    destination:
      ports:
      - 53
  - action: Deny
```

Then, in the following example, we allow pods with “my-value” to
connect out to domains **alice.com** and **bob.example.com**.

```
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: allow-egress-to-domain
spec:
  order: 1
  selector: "my-pod-label == 'my-value'"
  types:
  - Egress
  egress:
  - action: Allow
    destination:
      domains:
      - alice.com
      - bob.example.com
```

Alternatively, you can configure allowed domains in a GlobalNetworkSet:

```
apiVersion: projectcalico.org/v3
kind: GlobalNetworkSet
metadata:
  name: allowed-domains-1
  labels:
  - name: destination-set
    value: alice-and-bob
spec:
  allowedEgressDomains:
  - alice.com
  - bob.example.com
```

and then reference that GlobalNetworkSet using a destination label
selector:

```
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: allow-egress-to-domain
spec:
  order: 1
  selector: "my-pod-label == 'my-value'"
  types:
  - Egress
  egress:
  - action: Allow
    destination:
      selector: "destination-set == 'alice-and-bob'"
```

> **Note**: Using a GlobalNetworkSet, instead of specifying domains
> directly in GlobalNetworkPolicy rules, makes sense when
> multiple policies need to reference the same set of domains, or when
> you want the allowed destinations for a rule to be a mix of domains
> and IPs from GlobalNetworkSets, as well, perhaps, as IPs from
> WorkloadEndpoints and HostEndpoints.  A single destination selector
> expression can potentially match all of those.
{: .alert .alert-info}

### Above and Beyond

To change the default DNS parameters, see [Felix configuration
parameters]({{site.baseurl}}/{{page.version}}/reference/felix/configuration).
