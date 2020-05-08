---
title: Helm
description: Install Calico Enterprise using Helm application package manager.
---

### Big picture

Install {{ site.prodname }} on a deployed Kubernetes cluster using Helm.

### Before you begin...

**Required**

- Tiller **v2.14** is running, and your local helm CLI is configured to speak to it.
- [Credentials for the Tigera private registry and a license key]({{site.baseurl}}/getting-started/calico-enterprise)

### How to

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Get the Helm chart.
   ```
   curl -O -L https://s3.amazonaws.com/tigera-public/ee/charts/tigera-operator-{% include chart_version_name %}.tgz
   ```
1. Install the chart, passing in your image pull secrets.
   ```
   helm install ./tigera-operator-{% include chart_version_name %}.tgz \
     --name {{ site.prodnamedash }} \
     --set-file imagePullSecrets.tigera-pull-secret=<path/to/pull/secret>
   ```
2. Monitor progress, wait until `apiserver` shows a status of `Available`, then proceed to the next step.
   ```
   watch kubectl get tigerastatus
   ``` 
3. Install your {{ site.prodname }} license.
   ```
   kubectl apply -f </path/to/license.yaml>
   ```
4. Monitor progress, wait until all components show a status of `Available`, then proceed to the next step.
   ```
   watch kubectl get tigerastatus
   ```
5. Apply the following manifest to secure {{site.prodname}} with network policy:
   ```
   kubectl apply -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
   ```

### Next steps

**Required**

- [Install and configure CLIs]({{site.baseurl}}/getting-started/clis/)

**Recommended**

- [Configure access to {{site.prodname}} Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Configure user authentication and log in]({{site.baseurl}}/getting-started/cnx/create-user-login)

**Recommended - Networking**

- If you are using the default BGP networking with full-mesh node-to-node peering with no encapsulation, go to [Configure BGP peering]({{site.baseurl}}/networking/bgp) to get traffic flowing between pods.
- If you are unsure about networking options, or want to implement encapsulation (overlay networking), see [Determine best networking option]({{site.baseurl}}/networking/determine-best-networking).

**Recommended - Security**

- [Get started with {{site.prodname}} tiered network policy]({{site.baseurl}}/security/tiered-policy)
