---
title: Installing Tigera Secure EE using Helm
---

## Before you begin

- Ensure that you have the [credentials for the Tigera private registry]({{ site.basuerl }}/{{ page.version }}/getting-started/#obtain-the-private-registry-credentials), referred in this doc as `config.json`.

- Ensure that you have a [license key]({{ site.baseurl }}/{{ page.version }}/getting-started/#obtain-a-license-key), referred to in this doc as `license.yaml`

- Ensure that Tiller is running, and your local helm CLI tool is configured to speak to it.

- Ensure you've acquired the {{ site.prodname }} Helm Artifacts from the [Tigera support portal](https://support.tigera.io):

  - tigera-secure-ee-core.tgz
  - tigera-secure-ee.tgz

## Step 1: Craft your values.yaml for {{ site.prodname }} Core

Before we install, we must build a helm values file to configure {{ site.prodname }} Core for your environment. We weill refer to this values file as `my-values.yaml` at the time of installation.

For the purposes of this install guide, we will cover options which must be set in order to achieve a functioning cluster. For a full reference of all available options, inspect the helm chart:

    helm inspect tigera-secure-ee-core.tgz

### Configure your Datastore Connection

To use Kubernetes as your datastore:

```yaml
datastore: kubernetes
```

To use etcd as your datastore

```yaml
datastore: etcd
etcd:
  endpoints: http://etcd.co
```

To connect to an etcd secured by TLS, also pass your certs into `etcd.tls` at install time with the following flags:

```
--set-file etcd.tls.crt=./etcd.crt \
--set-file etcd.tls.ca=./etcd.ca \
--set-file etcd.tls.key=./etcd.key
```

### Network settings

By default, {{ site.prodname }} installs with IPIP encapsulation enabled.

**Turn off IPIP**

```yaml
initialPool:
  ipIpMode: None
```

**Only use IPIP across subnets**

```yaml
initialPool:
  ipIpMode: CrossSubnet
```

### Default Pool CIDR

By default, {{ site.prodname }} creates an IPv4 Pool with CIDR `192.168.0.0/16` when it launches. To change this CIDR:

```yaml
initialPool:
  cidr: 10.0.0.0/8
```

>**Note**: This should fall within `--cluster-cidr` configured for the cluster
{: .alert .alert-info}

>**Warning**: Changing this value after installation will have no effect.
{: .alert .alert-info}

## Step 2: Install {{ site.prodname }} Core

1. Install the chart, passing in the `my-values.yaml` file you crafted from the previous section, an additionally passing your image pull secrets:

   ```
   helm install ./tigera-secure-ee-core.tgz -f my-values.yaml
     --set-file imagePullSecrets.cnx-pull-secret=./config.json
   ```

2. Wait for the 'cnx-apiserver' pod to become ready:

   ```
   kubectl rollout status -n kube-system deployment/cnx-apiserver
   ```

3. Install your {{ site.prodname }} license:

   ```
   kubectl apply -f ./license.yaml
   ```

4. Apply the following manifest to set network policy that secures access to {{ site.prodname }}:

   ```
   kubectl apply -f {{ site.url }}/{{ page.version }}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml
   ```

Now that the **{{ site.prodname }} Core** chart is installed, please move on to the next step to install the **{{ site.prodname }}** chart.

## Step 3: Craft your values.yaml for {{ site.prodname }}

Before we install, we must build a helm values file to configure {{ site.prodname }} for your environment. We weill refer to this values file as `my-values.yaml` at the time of installation.

For the purposes of this install guide, we will cover options which must be set in order to achieve a functioning cluster. For a full reference of all available options, inspect the helm chart:

    helm inspect tigera-secure-ee.tgz

### Connect to Elasticsearch

By default, {{ site.prodname }} launches Elasticsearch Operator to bootstrap an unsecured elasticsearch cluster for demonstrative purposes. To disable this behavior and instead connect to your own elasticsearch, define the address in your yaml:

```yaml
elasticsearch:
  host: my.elasticsearch.co
  port: 9200
```

Additionally, provide the CA and passwords for each of the roles:

```
--set-file elasticsearch.tls.ca=./elastic.ca \
--set elasticsearch.fluentd.password=$FLUENTD_PASSWORD \
--set elasticsearch.manager.password=$MANAGER_PASSWORD \
--set elasticsearch.curator.password=$CURATOR_PASSWORD \
--set elasticsearch.compliance.password=$COMPLIANCE_PASSWORD \
--set elasticsearch.intrusionDetection.password=$IDS_PASSWORD \
--set elasticsearch.elasticInstaller.password=$ELASTIC_INSTALLER_PASSWORD
```

See [information on the permissions of these roles]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/byo-elasticsearch#before-you-begin) for help setting up these roles.

### Setting an Auth Type

**Basic auth**

```yaml
manager:
  auth:
    type: basic
```

**OIDC**

```yaml
manager:
  auth:
    type: oidc
    authority: "https://accounts.google.com"
    clientID: "<oidc-client-id>"
```

**Oauth**

```yaml
manager:
  auth:
    type: oauth
    authority: "https://<oauth-authority>/oauth/authorize"
    clientID: "cnx-manager"
```

## Step 4: Install {{ site.prodname }}

0. Pre-install the CRDs.

   Due to [a bug in helm](https://github.com/helm/helm/issues/4925), it is possible for the CRDs that are created by this chart to fail to get fully deployed before Helm attempts to create resources that require them. This affects all versions of Helm with a potential fix pending. In order to work around this issue when installing the chart you will need to make sure all CRDs exist in the cluster first:

   ```
   kubectl apply -f {{ site.url }}/{{ page.version }}/getting-started/kubernetes/installation/helm/tigera-secure-lma/operator-crds.yaml
   ```

   >[Click to view this manifest directly]({{ site.url }}/{{ page.version }}/getting-started/kubernetes/installation/helm/tigera-secure-lma/operator-crds.yaml)

1. Install the tigera-secure-ee helm chart with custom resource provisioning disabled:

   ```
   helm install ./tigera-secure-ee.tgz --set createCustomResources=false \
     --set-file imagePullSecrets.cnx-pull-secret=./config.json
   ```

## Step 5: Grant a User Access to the Manager Install

{% include {{page.version}}/cnx-grant-user-manager-permissions.md %}

## Next steps

Consult the {{site.prodname}} for Kubernetes [demo](/{{page.version}}/security/simple-policy-cnx), which
demonstrates the main features.
