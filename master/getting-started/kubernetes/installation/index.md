---
title: Installing Tigera Secure EE on Kubernetes
---

We provide a number of manifests to get you up and running with {{site.prodname}} in
just a few steps. Refer to the section that corresponds to your desired networking
for instructions.

If you are planning to use {{site.prodname}} Federated Endpoint Identity, please read the
[Federated Endpoint Identity Overview](/{{page.version}}/usage/federation/index) for additional
installation requirements. Some of the cluster configuration requirements for {{site.prodname}} Federation
cannot be easily changed on a running cluster.

- [Installing {{site.prodname}} for policy and networking (recommended)](calico)

- [Installing {{site.prodname}} for policy (advanced)](other)

After installing {{site.prodname}}, you can [enable application layer policy](app-layer-policy).
Enabling application layer policy also secures workload-to-workload communications with mutual
TLS authentication.

Should you wish to modify the manifests before applying them, refer to
[Customizing the manifests](config-options).

If you prefer not to use Kubernetes to start the {{site.prodname}} services, refer to the
[Integration guide](integration).
