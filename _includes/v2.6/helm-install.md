## Installing Tigera Secure EE

Ensure that you have the following:

- [credentials for the Tigera private registry]({{ site.basuerl }}/{{ page.version }}/getting-started/#obtain-the-private-registry-credentials), (`config.json`)

- A [license key]({{ site.baseurl }}/{{ page.version }}/getting-started/#obtain-a-license-key) (`license.yaml`)

{%- if include.method == "full" %}

- Tiller **v2.14+** is running, and your local helm CLI tool is configured to speak to it.

{% endif %}

The high-level steps to a functioning cluster with access to the user interface are:

- [Acquire the helm charts](#acquire-the-helm-charts)

{%- if include.method == "full" %}

- [Create values.yaml for {{ site.prodname }} Core](#create-valuesyaml-for-tigera-secure-ee-core)

{% endif %}

- [Install {{ site.prodname }} Core](#install-tigera-secure-ee-core)

{%- if include.method == "full" %}

- [Create values.yaml for {{ site.prodname }}](#create-valuesyaml-for-tigera-secure-ee)

{% endif %}

- [Install {{ site.prodname }}](#install-tigera-secure-ee)

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

### Create values.yaml for {{ site.prodname }} Core

In this step, you create a values.yaml file with your configuration values to build a running cluster.

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

**Default Pool CIDR**

By default, {{ site.prodname }} creates an IPv4 Pool with CIDR `192.168.0.0/16` when it launches. To change this CIDR:

```yaml
initialPool:
  cidr: 10.0.0.0/8
```

>**Warning**: Changing any network settings in the `initialPool` block after installation will have no effect. For information on changing IP Pools after installation, see [configuring IP Pools]({{site.url}}/{{page.version}}/networking/migrate-pools)
{: .alert .alert-warning}

>**Note**: This should fall within `--cluster-cidr` configured for the cluster
{: .alert .alert-info}

{% endif %}

### Install {{ site.prodname }} Core

1. Install the chart, passing in the `values.yaml` file you created from the previous section, an additionally passing your image pull secrets:

{%- if include.method == "full" %}
   ```
   helm install ./tigera-secure-ee-core-{% include chart_version_name %}.tgz \
     -f values.yaml \
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
   kubectl apply -f {{ site.url }}/{{ page.version }}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml
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
   kubectl apply -f {{ site.url }}/{{ page.version }}/reference/other-install-methods/kubernetes/installation/helm/tigera-secure-ee/operator-crds.yaml
   ```

   >[Click to view this manifest directly]({{ site.baseurl }}/{{ page.version }}/reference/other-install-methods/kubernetes/installation/helm/tigera-secure-ee/operator-crds.yaml)

1. Install the tigera-secure-ee helm chart with custom resource provisioning disabled:

   ```
   helm install ./tigera-secure-ee-{% include chart_version_name %}.tgz \
     --namespace calico-monitoring \
     --set createCustomResources=false \
     --set-file imagePullSecrets.cnx-pull-secret=./config.json
   ```

   >Note: This version of the Tigera Secure EE Helm chart **must** be installed with `--namespace calico-monitoring`.

   >Note: If you have not chosen to use a preexisting elasticsearch cluster, some pods may crashloop several times until the elasticsearch pods converge.

### Grant access to user interface

In this step, we are going to grant a user permission to access the Tigera Secure EE Manager in your cluster. For instructions on how to create a user, please consult our ["Configuring user authentication to Tigera Secure EE Manager" document](/{{page.version}}/reference/cnx/authentication#basic-authentication). Once you have a user, you can run the following commands, replacing `<USER>` with the name of the user you wish to grant access.

**User manager**

The `tigera-ui-user` role grants permission to use the Tigera Secure EE Manager UI, view flow logs, audit logs, and network statistics, and access the default policy tier.

```
kubectl create clusterrolebinding <USER>-tigera \
  --clusterrole=tigera-ui-user \
  --user=<USER>
```

**Network Admin**

The `tigera-network-admin` role grants permission to use the Tigera Secure EE Manager UI, view flow logs, audit logs, and network statistics, and administer all network policies and tiers.

```
kubectl create clusterrolebinding <USER>-network-admin \
  --clusterrole=tigera-network-admin \
  --user=<USER>
```

To grant access to additional tiers, or create your own roles, see the RBAC documentation.

### Log in to the Manager UI

```
kubectl port-forward -n calico-monitoring svc/cnx-manager 9443 & \
  kubectl port-forward -n calico-monitoring svc/tigera-kibana 5601
```

Sign in by navigating to https://localhost:9443 and login.

#### Connect to Elasticsearch & Kibana

You can connect to ElasticSearch and Kibana using the default user `elastic` that is used for installation.
In order to change the user to ElasticSearch and Kibana you need to provide the passwords for each of the roles:

```
helm upgrade $DEPLOYMENT $CHART_NAME
--set elasticsearch.fluentd.password=$FLUENTD_PASSWORD \
--set elasticsearch.manager.password=$MANAGER_PASSWORD \
--set elasticsearch.curator.password=$CURATOR_PASSWORD \
--set elasticsearch.compliance.benchmarker.password=$COMPLIANCE_BENCHMARKER_PASSWORD \
--set elasticsearch.compliance.controller.password=$COMPLIANCE_CONTROLLER_PASSWORD \
--set elasticsearch.compliance.reporter.password=$COMPLIANCE_REPORTER_PASSWORD \
--set elasticsearch.compliance.snapshotter.password=$COMPLIANCE_SNAPSHOTTER_PASSWORD \
--set elasticsearch.compliance.server.password=$COMPLIANCE_SERVER_PASSWORD \
--set elasticsearch.intrusionDetection.password=$IDS_PASSWORD \
--reuse-values --recreate-pods
```

If you're using the Elasticsearch X-Pack security then you may wish to use the following roles. You should
use the [Kibana Role Management API](https://www.elastic.co/guide/en/kibana/7.3/role-management-api.html),
since some roles include permissions on both Kibana and Elasticsearch.

They may also be useful as a reference for defining alternative security configuration.

1. fluentd role for creating indices and sending logs to Elasticsearch  (`tigera-ee-fluentd`)

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor", "manage_index_templates" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "create_index", "write" ]
         }
       ]
     }
   }
   ```

1. {{site.prodname}} Manager role for querying Elasticsearch (`tigera-ee-manager`)

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*", ".kibana" ],
           "privileges": [ "read"]
         }
       ]
     }
   }
   ```

1. {{site.prodname}} role for installing machine learning jobs, Watcher jobs, and Kibana dashboards (`tigera-ee-installer`)

   ```json
   {
     "elasticsearch": {
       "cluster": [ "manage_ml", "manage_watcher" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "read"]
         }
       ]
     },
     "kibana": [
       {
         "privileges": ["all"]
       }
     ]
   }
   ```

1. {{site.prodname}} Curator role for deleting indices older than retention period in Elasticsearch (`tigera-ee-curator`)

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor", "manage_index_templates" ],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "all"]
         }
       ]
     }
   }
   ```

1. {{site.prodname}} intrusion detection controller role for processing threat feeds, flow logs and security events. (`tigera-ee-intrusion-detection`)

   ```json
   {
     "elasticsearch": {
       "cluster": [ "monitor", "manage_index_templates"],
       "indices": [
         {
           "names": [ "tigera_secure_ee_*" ],
           "privileges": [ "read"]
         },
         {
           "names": [ ".tigera.ipset.*", "tigera_secure_ee_events.*", ".tigera.domainnameset.*" ],
           "privileges": [ "all"]
         }
       ]
     }
   }
   ```

1. {{site.prodname}} compliance report and dashboard for assessing the compliance posture of the cluster.

    1. Compliance benchmarker role for storing benchmark results  (`tigera-ee-compliance-benchmarker`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_benchmark_results.*" ],
               "privileges": [ "create_index", "write", "view_index_metadata", "read" ]
             }
           ]
         }
       }
       ```

    1. Compliance controller role for querying last archived reports  (`tigera-ee-compliance-controller`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_compliance_reports.*" ],
               "privileges": [ "read" ]
             }
           ]
         }
       }
       ```

    1. Compliance reporter role for querying archived audit information and storing reports  (`tigera-ee-compliance-reporter`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_audit_*" ],
               "privileges": [ "read" ]
             },
             {
               "names": [ "tigera_secure_ee_snapshots.*" ],
               "privileges": [ "read" ]
             },
             {
               "names": [ "tigera_secure_ee_benchmark_results.*" ],
               "privileges": [ "read" ]
             },
             {
               "names": [ "tigera_secure_ee_compliance_reports.*" ],
               "privileges": [ "create_index", "write", "view_index_metadata", "read" ]
             }
           ]
         }
       }
       ```

    1. Compliance snapshotter role for recording daily configuration audits  (`tigera-ee-compliance-snapshotter`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_snapshots.*" ],
               "privileges": [ "create_index", "write", "view_index_metadata", "read" ]
             }
           ]
         }
       }
       ```

    1. Compliance server role for querying archived reports (`tigera-ee-compliance-server`)

       ```json
       {
         "elasticsearch": {
           "cluster": [ "monitor", "manage_index_templates"],
           "indices": [
             {
               "names": [ "tigera_secure_ee_compliance_reports.*" ],
               "privileges": [ "read" ]
             }
           ]
         }
       }
