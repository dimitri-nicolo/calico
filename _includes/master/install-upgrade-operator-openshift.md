> **Important**: These instructions have been tested using the latest build of master in https://github.com/openshift/installer (version v4.2+)
>                and only the bundled Elasticsearch operator.
{: .alert .alert-danger}

### Big picture

Install an OpenShift v4 cluster with {{site.prodname}}.

### Value

Augments the applicable steps in the [OpenShift documentation](https://cloud.redhat.com/openshift/install)
to install {{site.prodname}}.

### How to

#### Before you begin

- Ensure that your environment meets the {{site.prodname}} [system requirements](/{{page.version}}/getting-started/openshift/requirements).

- Ensure that you have the [private registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials)
  and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).

- **If installing on AWS**, ensure that you have [configured an AWS account](https://docs.openshift.com/container-platform/4.1/installing/installing_aws/installing-aws-account.html) appropriate for OpenShift v4,
  and have [set up your AWS credentials](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/setup-credentials.html).
  Note that the OpenShift installer supports a subset of [AWS regions](https://docs.openshift.com/container-platform/4.1/installing/installing_aws/installing-aws-account.html#installation-aws-regions_installing-aws-account).

- Ensure that you have a [RedHat account](https://cloud.redhat.com/). A RedHat account is required to obtain the pull secret necessary to provision an OpenShift cluster.

- Ensure that you have installed the OpenShift installer **v4.2 or later** and OpenShift command line interface from [cloud.redhat.com](https://cloud.redhat.com/openshift/install/aws/installer-provisioned).

- Ensure that you have [generated a local SSH private key](https://docs.openshift.com/container-platform/4.1/installing/installing_aws/installing-aws-default.html#ssh-agent-using_installing-aws-default) and have added it to your ssh-agent

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

> **Note**: Refer to the OpenShift installer documentation found on [https://cloud.redhat.com/openshift/install](https://cloud.redhat.com/openshift/install) for more information
> about the installer and any configuration changes required for your platform.
{: .alert .alert-info}

Once the installer has finished, your staging directory will contain the configuration file `install-config.yaml`.

#### Update the configuration file to use {{site.prodname}}

Override the OpenShift networking to use Calico and update the AWS instance types to meet the [system requirements](/{{page.version}}/getting-started/openshift/requirements):

```bash
sed -i 's/OpenShiftSDN/Calico/' install-config.yaml
sed -i 's/platform: {}/platform:\n    aws:\n      type: m4.xlarge/g' install-config.yaml
```

#### Generate the install manifests

Now generate the Kubernetes manifests using your configuration file:

```bash
openshift-install create manifests
```

Download the {{site.prodname}} manifests for OpenShift and add them to the generated manifests directory:

```bash
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-alertmanager.yaml -o manifests/01-crd-alertmanager.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-apiserver.yaml -o manifests/01-crd-apiserver.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-compliance.yaml -o manifests/01-crd-compliance.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-console.yaml -o manifests/01-crd-console.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-elasticsearchcluster.yaml -o manifests/01-crd-elasticsearchcluster.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-installation.yaml -o manifests/01-crd-installation.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-intrusiondetection.yaml -o manifests/01-crd-intrusiondetection.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-monitoringconfiguration.yaml -o manifests/01-crd-monitoringconfiguration.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-logcollector.yaml -o manifests/01-crd-logcollector.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-prometheusrule.yaml -o manifests/01-crd-prometheusrule.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-prometheus.yaml -o manifests/01-crd-prometheus.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-servicemonitor.yaml -o manifests/01-crd-servicemonitor.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/crds/01-crd-tigerastatus.yaml -o manifests/01-crd-tigerastatus.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tigera-operator/00-namespace-tigera-operator.yaml -o manifests/00-namespace-tigera-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tigera-operator/02-rolebinding-tigera-operator.yaml -o manifests/02-rolebinding-tigera-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tigera-operator/02-role-tigera-operator.yaml -o manifests/02-role-tigera-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tigera-operator/02-serviceaccount-tigera-operator.yaml -o manifests/02-serviceaccount-tigera-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/tigera-operator/02-tigera-operator.yaml -o manifests/02-tigera-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/elasticsearch-operator/03-clusterrolebinding-elasticsearch-operator.yaml -o manifests/03-clusterrolebinding-elasticsearch-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/elasticsearch-operator/03-clusterrole-elasticsearch-operator.yaml -o manifests/03-clusterrole-elasticsearch-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/elasticsearch-operator/03-deployment-elasticsearch-operator.yaml -o manifests/03-deployment-elasticsearch-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/elasticsearch-operator/03-serviceaccount-elasticsearch-operator.yaml -o manifests/03-serviceaccount-elasticsearch-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/misc/00-namespace-tigera-prometheus.yaml -o manifests/00-namespace-tigera-prometheus.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/prometheus-operator/04-clusterrolebinding-prometheus-operator.yaml -o manifests/04-clusterrolebinding-prometheus-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/prometheus-operator/04-clusterrolebinding-prometheus.yaml -o manifests/04-clusterrolebinding-prometheus.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/prometheus-operator/04-clusterrole-prometheus-operator.yaml -o manifests/04-clusterrole-prometheus-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/prometheus-operator/04-clusterrole-prometheus.yaml -o manifests/04-clusterrole-prometheus.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/prometheus-operator/04-deployment-prometheus-operator.yaml -o manifests/04-deployment-prometheus-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/prometheus-operator/04-serviceaccount-prometheus-operator.yaml -o manifests/04-serviceaccount-prometheus-operator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/prometheus-operator/04-serviceaccount-prometheus.yaml -o manifests/04-serviceaccount-prometheus.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/misc/00-namespace-calico-monitoring.yaml -o manifests/00-namespace-calico-monitoring.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/misc/99-alertmanager-secret.yaml -o manifests/99-alertmanager-secret.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/misc/99-alertmanager-service.yaml -o manifests/99-alertmanager-service.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/misc/99-elastic-curator.yaml -o manifests/99-elastic-curator.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/misc/99-kibana-service.yaml -o manifests/99-kibana-service.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/misc/99-prometheus-service.yaml -o manifests/99-prometheus-service.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/misc/99-tigera-es-config.yaml -o manifests/99-tigera-es-config.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-installation.yaml -o manifests/01-cr-installation.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-apiserver.yaml -o manifests/01-cr-apiserver.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-console.yaml -o manifests/01-cr-console.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-compliance.yaml -o manifests/01-cr-compliance.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-intrusiondetection.yaml -o manifests/01-cr-intrusiondetection.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-alertmanager.yaml -o manifests/01-cr-alertmanager.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-elasticsearchcluster.yaml -o manifests/01-cr-elasticsearchcluster.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-monitoringconfiguration.yaml -o manifests/01-cr-monitoringconfiguration.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-logcollector.yaml -o manifests/01-cr-logcollector.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-prometheus.yaml -o manifests/01-cr-prometheus.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-prometheusrule.yaml -o manifests/01-cr-prometheusrule.yaml
curl {{site.url}}/{{page.version}}/manifests/ocp/01-cr-servicemonitor.yaml -o manifests/01-cr-servicemonitor.yaml
```

> **Note**: The Tigera operator manifest downloaded above includes an initialization container which configures Amazon AWS
> security groups for {{site.prodname}}. If not running on AWS, you should remove the init container from `manifests/02-tigera-operator.yaml`.
{: .alert .alert-info}

#### Add an image pull secret

1. Download the pull secret manifest template into the manifests directory.

   ```
   curl {{site.url}}/{{page.version}}/manifests/ocp/02-pull-secret.yaml -o manifests/02-pull-secret.yaml
   ```

1. Update the contents of the secret with the image pull secret provided to you by Tigera.

   For example, if the secret is located at `~/.docker/config.json`, run the following commands.

   ```bash
   SECRET=$(cat ~/.docker/config.json | tr -d '\n\r\t ' | base64 -w 0)
   sed -i "s/SECRET/${SECRET}/" manifests/02-pull-secret.yaml
   ```

#### Create the cluster

Start the cluster creation with the following command and wait for it to complete.

```bash
openshift-install create cluster
```

#### Install the {{site.prodname}} license

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

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
oc create -f {{site.url}}/master/manifests/tigera-policies.yaml
```
