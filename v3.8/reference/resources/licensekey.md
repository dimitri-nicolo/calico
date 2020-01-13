---
title: License key
canonical_url: https://docs.tigera.io/v2.3/reference/calicoctl/resources/licensekey
---

A License Key resource (`LicenseKey`) represents a user's license to use {{site.tseeprodname}}. Keys are
provided by Tigera support, and must be applied to the cluster to enable
{{site.tseeprodname}} features.

This resource is not supported in `kubectl`.

### Working with license keys

#### Applying or updating a license key

When you add {{site.tseeprodname}} to an existing Kubernetes cluster or create a
new OpenShift cluster, you must apply your license key to complete the installation
and gain access to the full set of {{site.tseeprodname}} features.

When your license key expires, you must update it to continue using {{site.tseeprodname}}.

To apply or update a license key use the following command, replacing `<customer-name>`
with the customer name in the file sent to you by Tigera.

**Command**
```
calicoctl apply -f <customer-name>-license.yaml
```

**Example**
```
calicoctl apply -f awesome-corp-license.yaml
```

#### Viewing information about your license key

To check information about the license key of your cluster, use `calicoctl get`.

```
calicoctl get licensekey -o wide
```

This is an example of the output of that command.

```
LICENSEID    EXPIRATION                      NODES   FEATURES
<UUID>       1999-03-14 23:59:59 -0700 PDT   999     [cnx all]
```
{: .no-select-button}

### Sample YAML

```yaml
apiVersion: projectcalico.org/v3
kind: LicenseKey
metadata:
  creationTimestamp: null
  name: default
spec:
  certificate: |
    -----BEGIN CERTIFICATE-----
    MII...n5
    -----END CERTIFICATE-----
  token: eyJ...zaQ
```

The data fields in the license key resource may change without warning.  The license key resource
is currently a singleton: the only valid name is `default`.

### Supported operations

| Datastore type        | Create | Delete | Update | Get/List | Notes
|-----------------------|--------|--------|--------|----------|------
| etcdv3                | Yes    |   No   | Yes    | Yes      |
| Kubernetes API server | Yes    |   No   | Yes    | Yes      |
