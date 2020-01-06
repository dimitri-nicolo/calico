---
title: Adjust the log storage size
---

### Big picture

Adjust the size of the {{site.prodname}} log storage during or after installation.

### Value

By default, {{site.prodname}} creates the log storage with a single node. This makes it easy to get started using {{site.prodname}}. 
Generally, a single node for logs is fine for test or development purposes. Before going to production, you should scale 
the number of nodes, replicas, CPU, and memory to reflect a production environment.

### Features

This how-to guide uses the following {{site.prodname}} features:

-  **LogStorage** resource

### Concepts

#### Log storage terms

| Term    | Description                                                  |
| ------- | ------------------------------------------------------------ |
| node    | A running instance of the log storage.                  |
| cluster | A collection of nodes. Multiple nodes protect the cluster from any single node failing, and lets you scale resources (CPU, memory, storage space) . |
| replica | A copy of data. Replicas protect against data loss if a node fails. The number of replicas must be less than the number of nodes. |

### Before you begin...

**Review Log storage requirements**

Review [Log storage requirements]() for guidance on the number of nodes and resources to configure for your environment.

### How to

- [Adjust the number of nodes](#adjust-the-number-of-nodes)
- [Adjust the number of replicas](#adjust-the-number-of-replicas)
- [Adjust the volume size](#adjust-the-volume-size)
- [Adjust the CPU and memory](#adjust-the-cpu-and-memory)

In the following examples, you can set **LogStorage** resource values during {{site.prodname}} installation, or after by applying kubectl to the manifest.

#### Adjust the number of nodes

In the following example, {{site.prodname}} is configured to install two log storage nodes.

```
apiVersion: operator.tigera.io/v1
kind: LogStorage
metadata:
  name: tigera-secure
spec:
  nodes:
    count: 2
```

#### Adjust the number of replicas

In the following example, {{site.prodname}} is configured to install two replicas. We recommend creating at least one replica to protect against data loss in case a log storage node goes down. The number of replicas must be less than the number of nodes.

```
apiVersion: operator.tigera.io/v1
kind: LogStorage
metadata:
  name: tigera-secure
Spec:
  nodes:
    count: 3
  indices:
    replicas: 2
```

#### Adjust the volume size

In the following example, {{site.prodname}} is configured to install nodes that have 30Gi for storage.

```
apiVersion: operator.tigera.io/v1
kind: LogStorage
metadata:
  name: tigera-secure
spec:
  nodes:
    resourceRequirements:
      requests:
        storage: 30Gi
```

#### Adjust the CPU and memory

In the following example, {{site.prodname}} is configured to install nodes with 5Gi for CPU and memory.

```
apiVersion: operator.tigera.io/v1
kind: LogStorage
metadata:
  name: tigera-secure
spec:
  nodes:
    resourceRequirements:
      limits:
        cpu: 5Gi
        memory: 5Gi
      requests:
        cpu: 5Gi
        memory: 5Gi
```
