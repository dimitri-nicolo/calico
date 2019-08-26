> **Important**: These instructions have been tested using the latest build of master in https://github.com/openshift/installer (version v4.2+)
>                and only the bundled Elasticsearch operator.
{: .alert .alert-danger}
  
## Before you begin

- Ensure that you meet the {{site.prodname}} [system requirements](/{{page.version}}/getting-started/openshift/requirements).

- Ensure that you have the [private registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials)
  and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).

- **If installing on AWS and using IPI**, ensure that you have [configured an AWS account](https://docs.openshift.com/container-platform/4.1/installing/installing_aws/installing-aws-account.html) appropriate for OpenShift v4,
  and have [setup your AWS credentials](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/setup-credentials.html).
  Note that OpenShift Installer supports a subset of [AWS regions](https://docs.openshift.com/container-platform/4.1/installing/installing_aws/installing-aws-account.html#installation-aws-regions_installing-aws-account).

- Ensure that you have setup a [RedHat account](https://cloud.redhat.com/). A RedHat account is required to obtain the pull secret
  necessary to provision an OpenShift cluster.

- Ensure that you have downloaded and installed the OpenShift Installer v4.2 or later and OpenShift Command-Line Interface from [cloud.redhat.com](https://cloud.redhat.com/openshift/install/aws/installer-provisioned).

- Ensure that you have [generated a local SSH private key](https://docs.openshift.com/container-platform/4.1/installing/installing_aws/installing-aws-default.html#ssh-agent-using_installing-aws-default) and have added it to your ssh-agent

> **Note**: The operator-based installation currently only supports images from quay.io
{: .alert .alert-info}

## Installing {{site.prodname}} and OpenShift v4

While this page provides most of the steps needed to install {{site.prodname}} on OpenShift v4,
the [OpenShift documentation](https://cloud.redhat.com/openshift/install) should be followed for installation
and this page should be referenced during the applicable steps.

### Create a configuration file for OpenShift Installer

First, create a staging directory for the installation. This directory will contain the configuration file, along with cluster state files, that OpenShift Installer will create:

```
mkdir openshift-tigera-install && cd openshift-tigera-install
```

Now run OpenShift Installer to create a default configuration file:

```
openshift-install create install-config
```

> **Note**: Refer to the OpenShift Installer documentation found on [https://cloud.redhat.com/openshift/install](https://cloud.redhat.com/openshift/install) for more information
> about the installer and any configuration changes required for your platform.
{: .alert .alert-info}

Once the Installer has finished, your staging directory will contain the configuration file `install-config.yaml`.

## Update the configuration file for {{site.prodname}}

Override the OpenShift networking to use Calico and update the AWS instance types to meet the [system requirements](/{{page.version}}/getting-started/openshift/requirements):

```bash
sed -i 's/OpenShiftSDN/Calico/' install-config.yaml
sed -i 's/platform: {}/platform:\n    aws:\n      type: m4.xlarge/g' install-config.yaml
```

## Generate the manifests

Now generate the Kubernetes manifests using your config:

```bash
openshift-install create manifests
```

Download the Calico and operator manifests into the manifests directory:

```bash
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-crd-apiserver.yaml -o manifests/00-crd-apiserver.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-crd-compliance.yaml -o manifests/00-crd-compliance.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-crd-console.yaml -o manifests/00-crd-console.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-crd-installation.yaml -o manifests/00-crd-installation.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-crd-intrusiondetection.yaml -o manifests/00-crd-intrusiondetection.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-crd-monitoringconfiguration.yaml -o manifests/00-crd-monitoringconfiguration.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-crd-tigerastatus.yaml -o manifests/00-crd-tigerastatus.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-namespace.yaml -o manifests/00-namespace.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-operator.yaml -o manifests/00-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-role.yaml -o manifests/00-role.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-rolebinding.yaml -o manifests/00-rolebinding.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/calico/00-serviceaccount.yaml -o manifests/00-serviceaccount.yaml
```

Download the {{site.prodname}} manifests into the manifests directory:

```bash
curl {{site.url}}/{{page.version}}/manifests/ocp/tsee/00-role-extra.yaml -o manifests/00-role-extra.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tsee/00-rolebinding-extra.yaml -o manifests/00-rolebinding-extra.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tsee/02-apiserver.yaml -o manifests/02-apiserver.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tsee/02-compliance.yaml -o manifests/02-compliance.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tsee/02-console.yaml -o manifests/02-console.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tsee/02-installation.yaml -o manifests/02-installation.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tsee/02-intrusiondetection.yaml -o manifests/02-intrusiondetection.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tsee/02-ps.yaml -o manifests/02-ps.yaml
```

### Add private Docker registry override

If you are using a private Docker registry, add `registry: <YOUR_REGISTRY>` to the `Installation` resource in `manifests/02-installation.yaml`.
For example, on Linux you can run:

```bash
sed -i '/^  variant:.*/a \ \ registry: <YOUR_REGISTRY>' manifests/02-installation.yaml
```

### Add Docker config to pull secret

1. Strip the spaces, tabs, carriage returns, and newlines from the `config.json`
   file; base64 encode the string; and save it as an environment variable called `SECRET`.
   If you're on Linux, you can use the following command.

   ```bash
   SECRET=$(cat ~/.docker/config.json | tr -d '\n\r\t ' | base64 -w 0)
   ```
1. Replace the placeholder `SECRET` in the Tigera manifest:

   ```bash
   sed -i "s/SECRET/${SECRET}/" manifests/02-ps.yaml
   ```

Now, we are ready to create the cluster.

## Create the cluster

Start the cluster creation:

```bash
openshift-install create cluster
```

This will take a while. The output might look like this (it will vary depending on whether the infrastructure is installer-provided or user-provided):

```
INFO Consuming "Master Machines" from target directory
INFO Consuming "Worker Machines" from target directory
INFO Consuming "Openshift Manifests" from target directory
INFO Consuming "Common Manifests" from target directory
INFO Creating infrastructure resources...
INFO Waiting up to 30m0s for the Kubernetes API at https://api.demo.tigera.com:6443...
INFO API v1.14.0+334f918 up
INFO Waiting up to 30m0s for bootstrapping to complete...
INFO Destroying the bootstrap resources...
```

When you see a log similar to `INFO API v1.14.0+334f918 up`, view the status of the cluster with:

```bash
export KUBECONFIG=$(pwd)/auth/kubeconfig
oc get clusteroperator
```

Once you see the log `INFO Install complete!`, the initial OpenShift cluster creation has completed successfully.
The logs will include OpenShift console login details that might look like this:

```bash
INFO Destroying the bootstrap resources...
INFO Waiting up to 30m0s for the cluster at https://api.demo.openshift.tigera.com:6443 to initialize...
INFO Waiting up to 10m0s for the openshift-console route to be created...
INFO Install complete!
INFO To access the cluster as the system:admin user when using 'oc', run 'export KUBECONFIG=/home/demo/openshift-tigera-install/auth/kubeconfig'
INFO Access the OpenShift web-console here: https://console-openshift-console.apps.demo.openshift.tigera.com
INFO Login to the console with user: kubeadmin, password: T3KTh-tSvPt-wgv8G-kpaTt
```

Take note of the login details and proceed to the [apply network policies](#apply-network-policies-to-enable-tigera-secure-ee-manager).

## Apply network policies to enable Tigera Secure EE Manager

Issue the following commands to download and apply the network policies to allow traffic to the Tigera Secure EE Manager.

```bash
curl {{site.url}}/{{page.version}}/manifests/ocp/manager-policy.yaml -O
oc apply -f manager-policy.yaml
```

## Enable privileged security context constraints

In order for {{site.prodname}} to mount host directories needed for node metrics and flow logs, the default user in the
`calico-monitoring` namespaces requires access to the privileged security context constraint.

```bash
oc adm policy add-scc-to-user --namespace=calico-monitoring privileged -z default
```

{% include {{page.version}}/apply-license.md init="openshift" cli="oc" %}

{% include {{page.version}}/cnx-monitor-install.md orch="openshift" installer="operator" elasticsearch="operator" %}

See the [Metrics](/{{page.version}}/security/metrics/) section for more information.

## Create {{site.prodname}} Manager serviceacounts

1. Create a serviceaccount for accessing the manager, replacing with `<USER>` and `<NAMESPACE>` with the user's name and namespace.

   ```bash
   oc create sa <USER> -n <NAMESPACE>
   ```

{% include {{page.version}}/cnx-grant-user-manager-permissions.md usertype="serviceaccount" installer="operator" %}

## Login to the {{site.prodname}} Manager

1. Get a configured serviceaccount's token, replacing with `<USER>` and `<NAMESPACE>` with the user's name and namespace.

   ```bash
   oc sa get-token <USER> -n <NAMESPACE>
   ```

1. Setup port-forwarding to the {{site.prodname}} Manager.

   ```bash
   oc port-forward "$(oc get po -l k8s-app=cnx-manager -n tigera-console -oname)" 9443:9443 -n tigera-console
   ```

1. Open your browser to https://localhost:9443/login/token and login with the token.
