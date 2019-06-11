---
title: calicoq host
---

`calicoq host <hostname>` shows you the endpoints that are hosted on
`<hostname>` and all the {{site.prodname}} policies and profiles that relate to those
endpoints.  It is equivalent to running `calicoq endpoint <endpoint-id>` for
each `<endpoint-id>` that is hosted on `<hostname>`.

## Options

```
-r --hide-rule-matches         Don't show the list of policies and profiles whose
                               rule selectors match each endpoint as an allowed or
                               disallowed source/destination.

-s --hide-selectors            Don't show the detailed selector expressions involved
                               (that cause each displayed profile or policy to match
                               each endpoint).

-o <OUTPUT> --output=<OUTPUT>  Set the output format. Should be one of yaml, json, or
                               ps. If nothing is set, defaults to ps.
```

## Example

```
$ DATASTORE_TYPE=kubernetes KUBECONFIG=/home/user/.kube/config calicoq host tigera-kubetest-01

Policies and profiles for each endpoint on host "tigera-kubetest-01":

Workload endpoint k8s/calico-monitoring.alertmanager-calico-node-alertmanager-0/eth0
  Policies:
  Profiles:
    Profile "ns.projectcalico.org/calico-monitoring"

Workload endpoint k8s/kube-system.kube-dns-3913472980-fgf9m/eth0
  Policies:
  Profiles:
    Profile "ns.projectcalico.org/kube-system"

Workload endpoint k8s/policy-demo.nginx-2371676037-j2vmh/eth0
  Policies:
  Profiles:
    Profile "ns.projectcalico.org/policy-demo"
  Rule matches:
    Policy "policy-demo/abcdefghijklmnopqrstuvwxyz" outbound rule 1 destination match; selector "projectcalico.org/namespace == 'policy-demo'"
```

## See also

-  [calicoq endpoint]({{site.baseurl}}/{{page.version}}/reference/calicoq/endpoint) for
   the related `calicoq endpoint` command.
-  [NetworkPolicy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/networkpolicy) and
   [GlobalNetworkPolicy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkpolicy)
   for more information about the {{site.prodname}} policy model.
