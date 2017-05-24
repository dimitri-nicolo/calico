---
title: calicoq host
---

`calicoq host <hostname>` is used to list the endpoints that are hosted on
`<hostname>`, and to show the Calico profiles and policies that apply to each
of those endpoints.

### Example

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

## Options

```
  -s --hide-selectors        Hide selectors from output.
  -r --include-rule-matches  Show policies whose rules match endpoints on the host.
```

## See also

-  [Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy) for
   more information about the Calico policy model.
