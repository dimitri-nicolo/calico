---
title: Using domain names in policy rules
canonical_url: 'https://docs.projectcalico.org/master/security/domain-based-policy'
---

### Big Picture

Cluster operators can use domain names in global {{site.prodname}}
policy, to allow connection to services with those domain names
outside the cluster.

### Value

Identifying an external service by its domain name is usually more
convenient than specifying its IP.  Especially if the IP can change
while the domain name stays the same, as is often the case.

> **Note**: For services *within* the cluster, Kubernetes labels
> provide similar convenience.  Using domain names for services
> *within* the cluster is not supported by the feature described here,
> because Kubernetes labels can and should be used for those instead.
{: .alert .alert-info}

### Features

-  Domain names can be used to specify destinations outside the
   cluster to which traffic should be allowed.  (They don't work for
   *denying* traffic to particular destinations, or for services
   *inside* the cluster.)

   > **Note**: This makes sense when egress traffic from the workloads
   > concerned (i.e. that the policy applies to) is denied by default.
   > In other words, so that the GlobalNetworkPolicy with domain names is
   > poking holes in a generally closed wall around the cluster.
   {: .alert .alert-info}

-  Supported domain names are those in DNS A, AAAA and CNAME records.

-  We rely on the client workload connecting to a service by name, and
   therefore doing a DNS lookup for the name.  {{site.prodname}}
   snoops the DNS exchange and uses that information to open a pinhole
   to the service's current IP address.

-  {{site.prodname}} only trusts DNS information that comes from a
   configurable set of DNS servers.  Otherwise an attacker could use a
   malicious workload to poison the system by sending DNS lookups to a
   fake DNS server under their own control.

-  By default, {{site.prodname}} trusts the Kubernetes cluster's DNS
   service (kube-dns or CoreDNS), which means that this feature will
   work out of the box unless pods take particular steps to override
   their DNS config.

### How to

Allowed destination domain names can be configured in two ways.

1. A
   [GlobalNetworkPolicy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkpolicy)
   can have egress rules with `action: Allow` and a
   `destination.domains` field specifying the domain names to which
   egress traffic is allowed.

2. Allowed destination domain names can be specified in the `allowedEgressDomains` field of a
   [GlobalNetworkSet]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkset)
   resource, and GlobalNetworkPolicy can then use a
   `destination.selector` expression that matches that
   GlobalNetworkSet.

### Tutorial

To configure a default egress policy for all pods, that denies all
outbound connections except for DNS:

```
calicoctl apply -f - <<EOF
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
EOF
```

Then, to allow particular pods to connect out to particular domains:

```
calicoctl apply -f - <<EOF
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
EOF
```

Obviously the domain names should be changed to those that you need,
and the pod selector `my-pod-label == 'my-value'` should be whatever
makes sense in your cluster for selecting the pods that you want to be
able to make those outbound connections.

Alternatively configure allowed domains in a GlobalNetworkSet like this:

```
calicoctl apply -f - <<EOF
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
EOF
```

and then reference that GlobalNetworkSet via a destination label selector:

```
calicoctl apply -f - <<EOF
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
EOF
```

> **Note**: Using a GlobalNetworkSet, instead of specifying domains
> directly in GlobalNetworkPolicy rules, would make sense when
> multiple policies need to reference the same set of domains, or when
> you want the allowed destinations for a rule to be a mix of domains
> and IPs from GlobalNetworkSets, as well, perhaps, as IPs from
> WorkloadEndpoints and HostEndpoints.  A single destination selector
> expression can potentially match all of those.
{: .alert .alert-info}

### Above and Beyond

This feature is configured by [Felix configuration
parameters]({{site.baseurl}}/{{page.version}}/reference/felix/configuration)
beginning with "DNS", but you normally won't need to change these, as
the defaults are chosen to work out of the box with standard
Kubernetes installs.
