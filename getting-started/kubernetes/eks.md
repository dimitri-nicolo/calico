---
title: Amazon Elastic Kubernetes Service (EKS)
description: Enable Calico network policy in EKS.
canonical_url: '/getting-started/kubernetes/eks'
---

### Big picture

Install {{site.prodname}} in EKS managed Kubernetes service.

### Value

You can use {{site.prodname}} with Amazon VPC CNI networking on EKS, or with Calico CNI networking in place of the default AWS VPC networking without the need to use IP addresses from the underlying VPC. This allows you to take
advantage of the full set of {{site.prodname}} networking features, including {{site.prodname}}'s flexible IP address management capabilities.

### Before you begin

**Required**

- [Credentials for the Tigera private registry and a license key]({{site.baseurl}}/getting-started/calico-enterprise)

- If using a private registry, familiarize yourself with this guide on [using a private registry]({{site.baseurl}}/getting-started/private-registry)

- Review [network requirements]({{site.baseurl}}/getting-started/kubernetes/requirements#network-requirements) to ensure network access is properly configured for {{site.prodname}} components

### How to

1. [Option A: Install with Amazon VPC networking](#install-eks-with-amazon-vpc-networking)
1. [Option B: Install with Calico CNI networking](#install-eks-with-calico-networking)
1. [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
1. [Secure {{site.prodname}} with network policy](#secure-calico-enterprise-with-network-policy)

{% include content/install-eks.md clusterType="standalone" %}

### Next steps

- [Configure access to {{site.prodname}} Manager]({{site.baseurl}}/getting-started/cnx/access-the-manager)

**Recommended**
- {% include open-new-window.html text='Video: Everything you need to know about Kubernetes pod networking on AWS' url='https://www.projectcalico.org/everything-you-need-to-know-about-kubernetes-pod-networking-on-aws/' %}
- [Get started with {{site.prodname}} network policy]({{ site.baseurl }}/security/calico-enterprise-policy)
- [Enable default deny for Kubernetes pods]({{ site.baseurl }}/security/kubernetes-default-deny)
