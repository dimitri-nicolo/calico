---
title: Installing Tigera Secure EE on Kubernetes
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/
---

We provide a number of manifests to get you up and running with {{site.tseeprodname}} in
just a few steps. Refer to the section that corresponds to your desired networking
for instructions.

- [Installing {{site.tseeprodname}} for policy and networking](calico)

- [Installing {{site.tseeprodname}} for policy](other): recommended for those on AWS who wish to
  [federate clusters](../../../usage/federation/index).

After installing {{site.tseeprodname}}, you can [enable application layer policy](app-layer-policy).
Enabling application layer policy also secures workload-to-workload communications with mutual
TLS authentication.

Should you wish to modify the manifests before applying them, refer to
[Customizing the Calico manifests](config-options) and
[Customizing the {{site.tseeprodname}} manifests](hosted/cnx/cnx).

If you prefer not to use Kubernetes to start the {{site.tseeprodname}} services, refer to the
[Integration guide](integration).
