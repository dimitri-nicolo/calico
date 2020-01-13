---
title: Configure storage for logs and reports
redirect_from: latest/getting-started/create-storage
---

### Big picture

Before installing {{site.prodname}}, you must configure persistent storage for flow logs, DNS logs, audit logs, and compliance reports.

### Concepts

Before configuring a storage class for {{site.prodname}}, the following terms will help you understand storage interactions.

#### Persistent volume

Used by pods to persist storage within the cluster. Combined with **persistent volume claims**, pods can persist data across restarts and rescheduling.

#### Persistent volume claim

Used by pods to request and mount storage volumes. The claim specifies the volume requirements for the request: size, access rights, and storage class.

#### Dynamic provisioner

Provisions types of persistent volumes on demand. Although most managed public-cloud clusters provide a dynamic provisioner using cloud-specific storage APIs (for example, Amazon EBS or Google persistent disks), not all clusters have a dynamic provisioner.

When a pod makes a persistent volume claim from a storage class that uses a dynamic provisioner, the volume is automatically created. If the storage class does not use a dynamic provisioner (for example the local storage class), the volumes must be created in advance. For help, see the [Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/dynamic-provisioning/).

#### Storage class

The storage provided by the cluster. Storage classes can be used with dynamic provisioners to automatically provision persistent volumes on demand, or with manually-provisioned persistent volumes. Different storage classes provide different service levels.

### Before you begin...

**Determine storage support**

Determine the storage types that are available on your cluster. If you are using dynamic provisioning, verify it is supported.
If you are using local disks, you may find the [sig-storage local static provisioner](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner) useful. It creates and manages PersistentVolumes by watching for disks mounted in a configured directory.

  > **Note**: Do not use the host path storage provisioner. This provisioner is not suitable for production and results in scalability issues, instability, and data loss.
{: .alert .alert-info}

### How to

#### Create a storage class

Before installing {{site.prodname}}, create a storage class named, `tigera-elasticsearch`.

**Examples**

##### Pre-provisioned local disks

In the following example, we create a **StorageClass** to use when explicitly adding **PersistentVolumes** for local disks. This can be performed manually, or using the [sig-storage local static provisioner](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner).

```
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: tigera-elasticsearch
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Retain
```

##### AWS EBS disks

In the following example for an AWS cloud provider integration, the **StorageClass** tells {{site.prodname}} to use EBS disks for log storage.

```
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: tigera-elasticsearch
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp2
reclaimPolicy: Retain
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
```

##### AKS Azure Files storage

In the following example for an AKS cloud provider integration, the **StorageClass** tells {{site.prodname}} to use LRS disks for log storage.
   > **Note**: Premium Storage is recommended for databases greater than 100GiB and for production installations.
{: .alert .alert-info}

```
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: tigera-elasticsearch
provisioner: kubernetes.io/azure-file
mountOptions:
  - uid=1000
  - gid=1000
  - mfsymlinks
  - nobrl
  - cache=none
parameters:
  skuName: Standard_LRS
reclaimPolicy: Retain
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

### Above and beyond

- [Adjust size of Elasticsearch cluster]()
