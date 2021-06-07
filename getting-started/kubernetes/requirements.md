---
title: System requirements
description: Review requirements before installing Calico Enterprise to ensure success.
canonical_url: '/getting-started/kubernetes/requirements'
---

{% include content/reqs-sys.md orch="Kubernetes" %}

### Kubernetes requirements

#### Supported Kubernetes versions

We test {{site.prodname}} {{page.version}} against the following Kubernetes versions.

- 1.19
- 1.20
- 1.21

Other versions may work, but we do not actively test {{site.prodname}}
{{page.version}} against them.

#### Supported managed Kubernetes versions

We test {{site.prodname}} {{page.version}} against the following managed Kubernetes versions.

- AKS: 1.20 
- GKE: 1.19
- EKS: 1.19

Other versions may work, but we do not actively test {{site.prodname}}
{{page.version}} against them.

#### Supported Docker Enterprise versions

We test {{site.prodname}} {{page.version}} against the following Docker Enterprise versions.

{% include content/docker-ee.md %}

Other versions may work, but we do not actively test {{site.prodname}}
{{page.version}} against them.

#### CNI plug-in enabled

{{site.prodname}} is installed as a CNI plugin. The kubelet must be configured
to use CNI networking by passing the `--network-plugin=cni` argument. (On
kubeadm, this is the default.)

#### Other network providers

{{site.prodname}} must be the only network provider in each cluster. We do
not currently support migrating a cluster with another network provider to
use {{site.prodname}} networking.

#### Supported kube-proxy modes

{{site.prodname}} supports the following kube-proxy modes:
- `iptables` (default)

#### IP pool configuration

The IP range selected for pod IP addresses cannot overlap with any other
IP ranges in your network, including:

- The Kubernetes service cluster IP range
- The range from which host IPs are allocated

### Application layer policy requirements

- {% include open-new-window.html text='MutatingAdmissionWebhook' url='https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook' %} enabled
- Istio {% include open-new-window.html text='v1.6' url='https://archive.istio.io/v1.6/' %}, or {% include open-new-window.html text='v1.7' url='https://archive.istio.io/v1.7/' %}

Note that Kubernetes version 1.16 requires Istio version 1.2 or greater.
Note that Istio version 1.7 requires Kubernetes version 1.16+.

{% include content/reqs-kernel.md %}
