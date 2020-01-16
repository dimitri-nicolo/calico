## Installing Calico Enterprise

Ensure that you have the following:

- [credentials for the Tigera private registry]({{ site.baseurl }}/getting-started/calico-enterprise#obtain-the-private-registry-credentials), (`config.json`)

- A [license key]({{ site.baseurl }}/getting-started/calico-enterprise#obtain-a-license-key) (`license.yaml`)

{%- if include.method == "full" %}

- Tiller **v2.14+** is running, and your local helm CLI tool is configured to speak to it.

{% endif %}

The high-level steps to a functioning cluster with access to the user interface are:

- [Acquire the helm charts](#acquire-the-helm-charts)

{%- if include.method == "full" %}

- [Create values-core.yaml for {{ site.prodname }} Core](#create-values-coreyaml-for-calico-enterprise-core)

{% endif %}

- [Install {{ site.prodname }} Core](#install-calico-enterprise-core)

{%- if include.method == "full" %}

- [Create values.yaml for {{ site.prodname }}](#create-valuesyaml-for-calico-enterprise)

{% endif %}

- [Install {{ site.prodname }}](#install-calico-enterprise)

- [Grant access to user interface](#grant-access-to-user-interface)

- [Log in to the Manager UI](#log-in-to-the-manager-ui)

### Acquire the Helm charts

```
curl -O -L https://s3.amazonaws.com/tigera-public/ee/charts/tigera-secure-ee-core-{% include chart_version_name %}.tgz
curl -O -L https://s3.amazonaws.com/tigera-public/ee/charts/tigera-secure-ee-{% include chart_version_name %}.tgz
```

{%- if include.method == "full" %}

### Configure Elastic storage

The bundled ElasticSearch operator is configured to use a `StorageClass` called `tigera-elasticsearch`.

Create a StorageClass with that name providing persistent storage that meets the requirements.

### Create values-core.yaml for {{ site.prodname }} Core

In this step, you create a values-core.yaml file with your configuration values to build a running cluster.

#### Configure your Datastore Connection

**Kubernetes Datastore**

```yaml
datastore: kubernetes
```

**Etcd datastore**

```yaml
datastore: etcd
etcd:
  endpoints: http://etcd.co
```

**Etcd secured by TLS**

Set the following flags to specify TLS certs to use when connecting to etcd:

```
--set-file etcd.tls.crt=./etcd.crt \
--set-file etcd.tls.ca=./etcd.ca \
--set-file etcd.tls.key=./etcd.key
```

#### Network settings

By default, {{ site.prodname }} runs with Calico networking.

```yaml
network: calico
```

**Azure Kubernetes Service (AKS)**

```yaml
platform: aks
network: aks
```

**Google Kubernetes Engine (GKE)**

```yaml
platform: gke
network: gke
```

**Amazon Elastic Kubernetes Service (EKS)**

```yaml
platform: eks
network: ecs
```

**AWS VPC CNI plugin**

To run {{ site.prodname }} in policy-only mode using Elastic Network Interfaces on AWS via the AWS VPC CNI plugin outside of EKS, omit the `platform` flag:

```
network: ecs
```

**Policy-only Mode**

To run Calico in policy-only mode for all other networking methods, set:

```
network: none
```

**MTU**

The default {{ site.prodname }} MTU is 1440. To change the MTU:

```yaml
mtu: 1500
```

**Default Pool CIDR**

By default, {{ site.prodname }} creates an IPv4 Pool with CIDR `192.168.0.0/16` when it launches. To change this CIDR:

```yaml
initialPool:
  cidr: 10.0.0.0/8
```

>**Warning**: Changing any network settings in the `initialPool` block after installation will have no effect. For information on changing IP Pools after installation, see [configuring IP Pools]({{site.baseurl}}/networking/migrate-pools)
{: .alert .alert-warning}

>**Note**: This should fall within `--cluster-cidr` configured for the cluster
{: .alert .alert-info}

{% endif %}

### Install {{ site.prodname }} Core

1. Install the chart, passing in the `values-core.yaml` file you created from the previous section, an additionally passing your image pull secrets:

{%- if include.method == "full" %}
   ```
   helm install ./tigera-secure-ee-core-{% include chart_version_name %}.tgz \
     -f values-core.yaml \
     --set-file imagePullSecrets.cnx-pull-secret=./config.json
   ```
{% else %}
   ```
   helm install ./tigera-secure-ee-core-{% include chart_version_name %}.tgz \
     --set-file imagePullSecrets.cnx-pull-secret=./config.json
   ```
{% endif %}

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
   kubectl apply -f {{ "/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml" | absolute_url }}
   ```

Now that the **{{ site.prodname }} Core** chart is installed, please move on to the next step to install the **{{ site.prodname }}** chart.

{%- if include.method == "full" %}

### Create values.yaml for {{ site.prodname }}

Before we install, we must build a helm values file to configure {{ site.prodname }} for your environment. We will refer to this values file as `values.yaml` at the time of installation.

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
    type: OIDC
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

{% endif %}

### Install {{ site.prodname }}

0. Pre-install the CRDs.

   Due to [a bug in helm](https://github.com/helm/helm/issues/4925), it is possible for the CRDs that are created by this chart to fail to get fully deployed before Helm attempts to create resources that require them. This affects all versions of Helm with a potential fix pending. In order to work around this issue when installing the chart you will need to make sure all CRDs exist in the cluster first:

   ```
   kubectl apply -f {{ "/reference/other-install-methods/kubernetes/installation/helm/calico-enterprise/operator-crds.yaml" | absolute_url }}
   ```

   >[Click to view this manifest directly]({{ site.baseurl }}/reference/other-install-methods/kubernetes/installation/helm/calico-enterprise/operator-crds.yaml)

1. Install the tigera-secure-ee helm chart with custom resource provisioning disabled:

   ```
   helm install ./tigera-secure-ee-{% include chart_version_name %}.tgz \
     --namespace calico-monitoring \
     --set createCustomResources=false \
     --set-file imagePullSecrets.cnx-pull-secret=./config.json
   ```

   >Note: This version of the Calico Enterprise Helm chart **must** be installed with `--namespace calico-monitoring`.

   >Note: If you have not chosen to use a preexisting elasticsearch cluster, some pods may crashloop several times until the elasticsearch pods converge.

### Grant access to user interface

In this step, we are going to grant a user permission to access the Calico Enterprise Manager in your cluster. For instructions on how to create a user, please consult our ["Configuring user authentication to Calico Enterprise Manager" document]({{site.baseurl}}/reference/cnx/authentication#basic-authentication). Once you have a user, you can run the following commands, replacing `<USER>` with the name of the user you wish to grant access.

**User manager**

The `tigera-ui-user` role grants permission to use the Calico Enterprise Manager UI, view flow logs, audit logs, and network statistics, and access the default policy tier.

```
kubectl create clusterrolebinding <USER>-tigera \
  --clusterrole=tigera-ui-user \
  --user=<USER>
```

**Network Admin**

The `tigera-network-admin` role grants permission to use the Calico Enterprise Manager UI, view flow logs, audit logs, and network statistics, and administer all network policies and tiers.

```
kubectl create clusterrolebinding <USER>-network-admin \
  --clusterrole=tigera-network-admin \
  --user=<USER>
```

To grant access to additional tiers, or create your own roles, see the RBAC documentation.

### Log in to the Manager UI

```
kubectl port-forward -n calico-monitoring svc/cnx-manager 9443
```

Sign in by navigating to https://localhost:9443 and login.

#### Kibana authentication

Connect to Kibana with the `elastic` username. Decode the Kibana password with the following command:

```
kubectl -n calico-monitoring get secret tigera-elasticsearch-es-elastic-user -o yaml |  awk '/elastic:/{print $2}' | base64 --decode
```

