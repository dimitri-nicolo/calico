---
title: Create storage for logs and reports
---

### Big picture

Before installing {{site.prodname}}, you must configure persistent storage for flow logs, DNS logs, audit logs, and compliance reports.


### Concepts

Before configuring a storage class for {{site.prodname}}, these terms will help you understand storage interactions:

- **Persistent volume**
  Used by pods to persist storage within the cluster. Combined with **persistent volume claims**, pods can persist data across restarts and rescheduling.

- **Storage class**
  The storage provided by the cluster. Storage classes can be used with dynamic provisioners to automatically provision persistent volumes on demand, or with manually-provisioned persistent volumes. Different storage classes provide different service levels. 

- **Persistent volume claim**
  Used by pods to request and mount storage volumes. The claim specifies the volume requirements for the request: size, access rights, and storage class.

- **Dynamic provisioner**
  Provisions types of persistent volumes on demand. Although most managed public-cloud clusters provide a dynamic provisioner using cloud-specific storage APIs (for example, Amazon EBS or Google persistent disks), not all clusters have a dynamic provisioner. Pods used with a dynamic provisioner automatically get the persistent volumes they require if they are configured with persistent volume claims with a storage class (and volumes do not already exist).

### Before you begin...

**Determine storage support**
  Determine the storage types that are available on your cluster. If using dynamic provisioning, verify it is supported.

  > **Note**: Do not use the host path storage provisioner. This provisioner is not suitable for production and results in scalability issues, instability, and data loss. {: .alert .alert-info}

**Manually provision persistent volumes**
  If your provider does not support dynamic provisioning or it is disabled, manually provision the persistent volumes. 
  For help, see the [Sizing guide](TBD).

### How to

#### Create a storage class

Before installing {{site.prodname}}, create a storage class named, `tigera-elasticsearch` using the [Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/dynamic-provisioning/).

### Above and beyond

- [Scale the {{site.prodname}} cluster]({{site.baseurl}}/{{page.version}}/TBD)