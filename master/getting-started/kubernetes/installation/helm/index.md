---
title: Installing TSEE Helm
---

## Before you begin

- Ensure that you have the [credentials for the Tigera private registry]({{ site.basuerl }}/{{ page.version }}/getting-started/#obtain-the-private-registry-credentials)
  and a [license key]({{ site.baseurl }}/{{ page.version }}/getting-started/#obtain-a-license-key).

- Ensure that Tiller is running, and your local helm CLI tool is configured to speak to it.

- Ensure you've acquired the {{ site.prodname }} Helm Artifacts from the Tigera support portal:

  - tigera-secure-core.tgz
  - tigera-secure-ee.tgz

## Step 1: Craft your values.yaml for Tigera Secure EE Core with Helm

- Before we install, we must build a values.yaml to configure {{ site.prodname }} for your environment.

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

To connect to an etcd secured by TLS, also pass your certs into `etcd.tls` as follows:

```
helm install ./tigera-secure-core \
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

If you are using a different pod CIDR, use the following commands to

Pod IPs will be chosen from this range. Changing this value after installation will have no effect. This should fall within `--cluster-cidr` configured for the cluster

```yaml
initialPool:
  cidr: 10.0.0.0/8
```

## Step 2: Install Tigera Secure EE Core with Helm

1. Install the chart

   ```
   helm install ./tigera-secure-ee-core.tgz -f my-values.yaml
     --set-file imagePullSecret=./tigera-pullsecret.json
   ```
   
2. Wait for the 'cnx-apiserver' pod to be running. 

3. Install your license:

   ```
   kubectl apply -f  ./license.yaml
   ```

4. Install some NetworkPolicy to secure the TSEE install:

   ```
   kubectl apply -f {{ site.url }}/{{ page.version }}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml
   ```

## Step 3: Craft your values.yaml Tigera Secure EE with Helm

### Use your own Elasticsearch

```yaml
elasticsearch:
  host: my.elasticsearch.co
  username: tigera-manager
  password: mypassword
```

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

## Step 4: Install Tigera Secure EE with Helm

0. Pre-install the CRDs.

   Due to [a bug in helm](https://github.com/helm/helm/issues/4925), it is possible for the CRDs that are created by this chart to fail to get fully deployed before Helm attempts to create resources that require them. This affects all versions of Helm with a potential fix pending. In order to work around this issue when installing the chart you will need to make sure all CRDs exist in the cluster first:

   ```
   kubectl apply -f {{ site.url }}/{{ page.version }}/getting-started/kubernetes/installation/helm/tigera-secure-lma/operator-crds.yaml
   ```

   >[Click to view this manifest directly]({{ site.url }}/{{ page.version }}/getting-started/kubernetes/installation/helm/tigera-secure-lma/operator-crds.yaml)


1. Install the tigera-secure-lma helm chart with custom resource provisioning disabled:

   ```
   helm install ./tigera-secure-lma.tgz --set createCustomResources=false
   ```

## Step 5: Grant a User Access to the Manager Install

include {{page.version}}/cnx-grant-user-manager-permissions.md

### Next Steps

- [Customize your Tigera Secure Enterprise Edition install](configuring)
