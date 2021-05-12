---
title: Install an OpenShift 4 cluster with Calico Enterprise
description: Install Calico Enterprise on an OpenShift v4 cluster.
canonical_url: '/getting-started/openshift/installation'
---

### Big picture

Install an OpenShift 4 cluster with {{site.prodname}}.

### Value

Augments the applicable steps in the {% include open-new-window.html text='OpenShift documentation' url='https://cloud.redhat.com/openshift/install' %} to install {{site.prodname}}.

### Before you begin

**Required**

- Your environment meets the {{site.prodname}} [system requirements]({{site.baseurl}}/getting-started/openshift/requirements)

- [Private registry credentials and license key]({{site.baseurl}}/getting-started/calico-enterprise)

- **If installing on AWS**, a {% include open-new-window.html text='configured an AWS account' url='https://docs.openshift.com/container-platform/4.2/installing/installing_aws/installing-aws-account.html' %} appropriate for OpenShift 4,
  and have {% include open-new-window.html text='set up your AWS credentials' url='https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/setup-credentials.html' %}. Note that the OpenShift installer supports a subset of {% include open-new-window.html text='AWS regions' url='https://docs.openshift.com/container-platform/4.3/installing/installing_aws/installing-aws-account.html#installation-aws-regions_installing-aws-account' %}.

- A {% include open-new-window.html text='RedHat account' url='https://cloud.redhat.com/' %} for the pull secret to provision an OpenShift cluster.

- OpenShift installer **v4.6 or later** and OpenShift command line interface from {% include open-new-window.html text='cloud.redhat.com' url='https://cloud.redhat.com/openshift/install/aws/installer-provisioned' %}

- A {% include open-new-window.html text='generated a local SSH private key' url='https://docs.openshift.com/container-platform/4.1/installing/installing_aws/installing-aws-default.html#ssh-agent-using_installing-aws-default' %} that is added to your ssh-agent

### How to

The geeky details of what you get:
{% include geek-details.html details='Policy:Calico,IPAM:Calico,CNI:Calico,Overlay:IPIP,Routing:BGP,Datastore:Kubernetes' %}

1. [Create a configuration file for the OpenShift installer](#create-a-configuration-file-for-the-openshift-installer)
1. [Update the configuration file to use {{site.prodname}}](#update-the-configuration-file-to-use-calico-enterprise)
1. [Generate the install manifests](#generate-the-install-manifests)
1. [Add an image pull secret](#add-an-image-pull-secret)
1. [Provide additional configuration](#provide-additional-configuration)
1. [Create the cluster](#create-the-cluster)
1. [Create a storage class](#create-a-storage-class)
1. [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
1. [Secure {{site.prodname}} components with network policy](#secure-calico-enterprise-components-with-network-policy)


{% include content/install-openshift.md clusterType="standalone" %}

### Next steps

**Recommended**

- [Configure access to {{site.prodname}} Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Authentication quickstart]({{site.baseurl}}/getting-started/cnx/authentication-quickstart)
- [Configure your own identity provider]({{site.baseurl}}/getting-started/cnx/configure-identity-provider)

**Recommended - Networking**

- The default networking uses IP in IP encapsulation with BGP routing. For all networking options, see [Determine best networking option]({{site.baseurl}}/networking/determine-best-networking).

**Recommended - Security**

- [Get started with {{site.prodname}} tiered network policy]({{site.baseurl}}/security/tiered-policy)
