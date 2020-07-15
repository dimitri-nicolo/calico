---
title: Install an OpenShift 4 cluster with Calico Enterprise
description: Install Calico Enterprise on an OpenShift v4 cluster.
canonical_url: '/getting-started/openshift/installation'
---

### Big picture

Install an OpenShift 4 cluster with {{site.prodname}}.

### Value

Augments the applicable steps in the {% include open-new-window.html text='OpenShift documentation' url='https://cloud.redhat.com/openshift/install' %}
to install {{site.prodname}}.

### How to

#### Before you begin

- Ensure that your environment meets the {{site.prodname}} [system requirements]({{site.baseurl}}/getting-started/openshift/requirements)

- Ensure that you have the [private registry credentials and license key]({{site.baseurl}}/getting-started/calico-enterprise)

- **If installing on AWS**, ensure that you have {% include open-new-window.html text='configured an AWS account' url='https://docs.openshift.com/container-platform/4.2/installing/installing_aws/installing-aws-account.html' %} appropriate for OpenShift 4,
  and have {% include open-new-window.html text='set up your AWS credentials' url='https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/setup-credentials.html' %}.
  Note that the OpenShift installer supports a subset of {% include open-new-window.html text='AWS regions' url='https://docs.openshift.com/container-platform/4.3/installing/installing_aws/installing-aws-account.html#installation-aws-regions_installing-aws-account' %}.

- Ensure that you have a {% include open-new-window.html text='RedHat account' url='https://cloud.redhat.com/' %}. A RedHat account is required to obtain the pull secret necessary to provision an OpenShift cluster.

- Ensure that you have installed the OpenShift installer **v4.3 or later** and OpenShift command line interface from {% include open-new-window.html text='cloud.redhat.com' url='https://cloud.redhat.com/openshift/install/aws/installer-provisioned' %}

- Ensure that you have {% include open-new-window.html text='generated a local SSH private key' url='https://docs.openshift.com/container-platform/4.1/installing/installing_aws/installing-aws-default.html#ssh-agent-using_installing-aws-default' %} and have added it to your ssh-agent

> **Note**: OpenShift v4.2 installation currently only supports {{site.prodname}} images pulled from quay.io
{: .alert .alert-info}

#### Create a configuration file for the OpenShift installer

First, create a staging directory for the installation. This directory will contain the configuration file, along with cluster state files, that OpenShift installer will create:

```
mkdir openshift-tigera-install && cd openshift-tigera-install
```

Now run OpenShift installer to create a default configuration file:

```
openshift-install create install-config
```

> **Note**: Refer to the {% include open-new-window.html text='OpenShift installer documentation' url='https://cloud.redhat.com/openshift/install' %} for more information
> about the installer and any configuration changes required for your platform.
{: .alert .alert-info}

Once the installer has finished, your staging directory will contain the configuration file `install-config.yaml`.

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

{% include content/openshift-prometheus-operator.md %}

{% include content/openshift-resources.md %}

#### Add an image pull secret

1. Download the pull secret manifest template into the manifests directory.

   ```
   curl {{ "/manifests/ocp/02-pull-secret.yaml" | absolute_url }} -o manifests/02-pull-secret.yaml
   ```

1. Update the contents of the secret with the image pull secret provided to you by Tigera.

   For example, if the secret is located at `~/.docker/config.json`, run the following commands.

   ```bash
   SECRET=$(cat ~/.docker/config.json | tr -d '\n\r\t ' | base64 -w 0)
   sed -i "s/SECRET/${SECRET}/" manifests/02-pull-secret.yaml
   ```
#### Optionally provide additional configuration

You may want to provide {{site.prodname}} with additional configuration at install-time. For example, BGP configuration or peers. You can use a Kubernetes ConfigMap with your desired {{site.prodname}} resources in order to set configuration as part of the installation. If you do not need to provide additional configuration, you can skip this section.

To include [{{site.prodname}} resources]({{site.baseurl}}/reference/resources) during installation, edit `manifests/02-configmap-calico-resources.yaml` in order to add your own configuration.

> **Note**: If you have a directory with the {{site.prodname}} resources, you can create the file with the command:
> ```
> kubectl create configmap -n tigera-operator calico-resources \
>   --from-file=<resource-directory> --dry-run -o yaml \
>   > manifests/02-configmap-calico-resources.yaml
> ```
> With recent versions of kubectl it is necessary to have a kubeconfig configured or add `--server='127.0.0.1:443'`
> even though it is not used.

> **Note**: If you have provided a `calico-resources` configmap and the tigera-operator pod fails to come up with `Init:CrashLoopBackOff`,
> check the output of the init-container with `kubectl logs -n tigera-operator -l k8s-app=tigera-operator -c create-initial-resources`.
{: .alert .alert-info}

#### Create the cluster

Start the cluster creation with the following command and wait for it to complete.

```bash
openshift-install create cluster
```

#### Create storage class

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

You can now monitor progress with the following command:

```
watch oc get tigerastatus
```

When it shows all components with status `Available`, proceed to the next section.

#### Secure {{site.prodname}} with network policy

To secure the components which make up {{site.prodname}}, install the following set of network policies.

```
oc create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```

### Next steps

**Recommended**

- [Configure access to {{site.prodname}} Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Configure user authentication and log in]({{site.baseurl}}/getting-started/cnx/create-user-login)

**Recommended - Networking**

- If you are using the default BGP networking with full-mesh node-to-node peering with no encapsulation, go to [Configure BGP peering]({{site.baseurl}}/networking/bgp) to get traffic flowing between pods.
- If you are unsure about networking options, or want to implement encapsulation (overlay networking), see [Determine best networking option]({{site.baseurl}}/networking/determine-best-networking).

**Recommended - Security**

- [Get started with {{site.prodname}} tiered network policy]({{site.baseurl}}/security/tiered-policy)
