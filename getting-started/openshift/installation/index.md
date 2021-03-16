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

#### Create a configuration file for the OpenShift installer

First, create a staging directory for the installation. This directory will contain the configuration file, along with cluster state files, that OpenShift installer will create:

```
mkdir openshift-tigera-install && cd openshift-tigera-install
```

Now run OpenShift installer to create a default configuration file:

```
openshift-install create install-config
```

> **Note**: See the {% include open-new-window.html text='OpenShift installer documentation' url='https://cloud.redhat.com/openshift/install' %} for more information
> about the installer and any configuration changes required for your platform.
{: .alert .alert-info}

After the installer finishes, your staging directory will contain the configuration file `install-config.yaml`.

#### Update the configuration file to use {{site.prodname}}

Override the OpenShift networking to use {{site.prodname}} and update the AWS instance types to meet the [system requirements]({{site.baseurl}}/getting-started/openshift/requirements):

```bash
sed -i 's/OpenShiftSDN/Calico/' install-config.yaml
sed -i 's/platform: {}/platform:\n    aws:\n      type: m4.xlarge/g' install-config.yaml
```

#### Generate the install manifests

Now generate the Kubernetes manifests using your configuration file:

```bash
openshift-install create manifests
```

{% include content/openshift-manifests.md %}

#### Add an image pull secret

{% include content/openshift-pull-secret.md %}

#### Provide additional configuration

You may want to provide {{site.prodname}} with additional configuration at install-time. For example, BGP configuration or peers. You can use a Kubernetes ConfigMap with your desired {{site.prodname}} resources in order to set configuration as part of the installation. If you do not need to provide additional configuration, you can skip this section.

To include [{{site.prodname}} resources]({{site.baseurl}}/reference/resources) during installation, edit `manifests/02-configmap-calico-resources.yaml` in order to add your own configuration.

> **Notes**: If you have a directory with the {{site.prodname}} resources, you can create the file with the command:
> ```
> kubectl create configmap -n tigera-operator calico-resources \
>   --from-file=<resource-directory> --dry-run -o yaml \
>   > manifests/02-configmap-calico-resources.yaml
> ```
> With recent versions of kubectl it is necessary to have a kubeconfig configured or add `--server='127.0.0.1:443'`
> even though it is not used.

> If you have provided a `calico-resources` configmap and the tigera-operator pod fails to come up with `Init:CrashLoopBackOff`,
> check the output of the init-container with `kubectl logs -n tigera-operator -l k8s-app=tigera-operator -c create-initial-resources`.
{: .alert .alert-info}

#### Create the cluster

Start the cluster creation with the following command and wait for it to complete.

```bash
openshift-install create cluster
```

#### Create a storage class

{{site.prodname}} requires storage for logs and reports. Before finishing the installation, you must [create a StorageClass for {{site.prodname}}]({{site.baseurl}}/getting-started/create-storage).

#### Install the {{site.prodname}} license

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.
Before applying the license, wait until the Tigera API server is ready with the following command:

```
watch oc get tigerastatus
```

Wait until the `apiserver` shows a status of `Available`.

Once the Tigera API server is ready, apply the license:

```
oc create -f </path/to/license.yaml>
```

#### Install {{site.prodname}} resources 
Apply the custom resources for enterprise features.

```bash
oc apply -f {{ "/manifests/ocp/tigera-enterprise-resources.yaml" | absolute_url }} 
```

{% include content/openshift-prometheus-operator.md %}

You can now monitor progress with the following command:

```
watch oc get tigerastatus
```

When it shows all components with status `Available`, proceed to the next section.

#### Secure {{site.prodname}} components with network policy

To secure the components which make up {{site.prodname}}, install the following set of network policies.

```
oc create -f {{ "/manifests/ocp/tigera-policies.yaml" | absolute_url }}
```

### Next steps

**Recommended**

- [Configure access to {{site.prodname}} Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Authentication quickstart]({{site.baseurl}}/getting-started/cnx/authentication-quickstart)
- [Configure your own identity provider]({{site.baseurl}}/getting-started/cnx/configure-identity-provider)

**Recommended - Networking**

- The default networking uses IP in IP encapsulation with BGP routing. For all networking options, see [Determine best networking option]({{site.baseurl}}/networking/determine-best-networking).

**Recommended - Security**

- [Get started with {{site.prodname}} tiered network policy]({{site.baseurl}}/security/tiered-policy)
