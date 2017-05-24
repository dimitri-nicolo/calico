---
title: calicoq host
---

`calicoq host <hostname>` shows you the endpoints that are hosted on
`<hostname>` and all the Calico profiles and policies that relate to those
endpoints.  It is equivalent to running `calicoq endpoint <endpoint-id>` for
each `<endpoint-id>` that is hosted on `<hostname>`.

## Options

```
-r --hide-rule-matches     Don't show the list of profiles and policies whose
                           rule selectors match <endpoint-id> as an allowed or
                           disallowed source/destination.

-s --hide-selectors        Don't show the detailed selector expressions involved
                           (that cause each displayed profile or policy to match
                           <endpoint-id>).
```

## Example

```
$ DATASTORE_TYPE=kubernetes KUBECONFIG=/home/user/.kube/config calicoq host tigera-kubetest-01
Policies that match each endpoint:

Workload endpoint k8s/calico-monitoring.alertmanager-calico-node-alertmanager-0/eth0
  Policies:
  Profiles:
    Profile ns.projectcalico.org/calico-monitoring

Workload endpoint k8s/kube-system.kube-dns-3913472980-fgf9m/eth0
  Policies:
  Profiles:
    Profile ns.projectcalico.org/kube-system

Workload endpoint k8s/policy-demo.nginx-2371676037-j2vmh/eth0
  Policies:
  Profiles:
    Profile ns.projectcalico.org/policy-demo
```

## See also

-  [calicoq endpoint]({{site.baseurl}}/{{page.version}}/reference/calicoq/endpoint) for
   more detail about the profiles and policies that are shown for each
   endpoint.
-  [Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy) for
   more information about the Calico policy model.
