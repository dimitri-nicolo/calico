---
title: Installing Calico Enterprise on Kubernetes
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/
---

We provide a number of manifests to get you up and running with {{site.prodname}} in
just a few steps. Refer to the section that corresponds to your desired networking
for instructions.

- [Installing {{site.prodname}} for policy and networking](calico)

- [Installing {{site.prodname}} for policy](other): recommended for those on AWS who wish to
  [federate clusters](../../../usage/federation/index).

After installing {{site.prodname}}, you can [enable application layer policy](app-layer-policy).
Enabling application layer policy also secures workload-to-workload communications with mutual
TLS authentication.

Should you wish to modify the manifests before applying them, refer to
[Customizing the Calico manifests](config-options) and
[Customizing the {{site.prodname}} manifests](hosted/cnx/cnx).

If you prefer not to use Kubernetes to start the {{site.prodname}} services, refer to the
[Integration guide](integration).
