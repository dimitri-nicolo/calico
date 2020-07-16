---
title: Configure fine-grained access control between VPC endpoints and pods 
description: Keep your AWS security groups, and configure fine-grained access control between VPC endpoints and pods. 
canonical_url: /security/aws-integration/tiers-and-policy
---

### Big picture

Configure ingress and egress from/to VPC resources to pods.

### Value

Keep your existing AWS security groups, and extend security for AWS/EKS Kubernetes clusters. 

### Features

- **AWS security groups integration**
- **NetworkPolicy** 

### Concepts

#### Using annotations with security groups

By default, every pod in your Kubernetes cluster belongs to the `K8S_NODE_SGS`, 
giving each pod the same access as the node on which it runs. To override the default security group(s) 
for a particular pod, you can specify one or more new security groups using an annotation. 
After you annotate a pod, {{site.prodname}} uses the security group(s) in the annotation instead 
of the default security group(s).

Note that adding an annotation does not remove the pod from the default pod security group; it just allows 
you to establish fine-grain control over communications from pods to VPC members.

#### Pod traffic

By default, all pod traffic is allowed. Every pod in your Kubernetes cluster belongs to an internal Tigera pod security group (POD_SG) for pod-to-pod communication, regardless of security group policies. {{site.prodname}} tiered policy processes traffic as follows: 
- Checks the rules of the pod’s security groups to see if a connection should be blocked 
- If the connection is not blocked by the security group rules, the traffic is processed by the {{site.prodname}} tiered policy or Kubernetes network policy. The security group policy only blocks or passes traffic to the next tier; it does not explicitly allow traffic. 

### Before you begin

**Required**

- [Configure {{site.prodname}} AWS security group integration]({{site.baseurl}}/security/aws-integration/aws-security-group-integration)

### How to

- [Restrict access to integration tiers](#restrict-access-to-integration-tiers)
- [Allow security group access to specific pods](#allow-security-group-access-to-specific-pods)
- [Deny pod access to VPC resource](#deny-pod-access-to-VPC-resource) 

#### Restrict access to integration tiers

Log in to {{site.prodname}} Manager, and verify that the following AWS integration tiers are present. These tiers are essential to operations and should never be modified. 

| Tier        | Order | Installed with ...         |
| ----------- | ----- | -------------------------- |
| `sg-remote` | 105   | AWS SG integration feature |
| `sg-local`  | 106   | AWS SG integration feature |
| `metadata`  | 107   | AWS SG integration feature |

Best practices are:
 
To avoid accidentally modifying the above tiers and associated network policies, do not allow admin and non-admin users to view and modify them. 
  - Although you cannot hide specific tiers from non-admin users, you can use [RBAC for tiered policies]({{site.baseurl}}/security/rbac-tiered-policies), to display only a subset of tiers in the {{site.prodname}} UI. For help, see [displaying only the net-sec tier]({{site.baseurl}}/security/rbac-tiered-policies#user-can-view-only-a-specific-tier).
  - To display tiers and associated network policies, but disable write access to those tiers, see [RBAC example fine-grained permissions]({{site.baseurl}}/security/rbac-tiered-policies#fine-grained-rbac-for-policies-and-tiers).
- Add any {{site.prodname}} tiers after order, 107.
- Do not modify the following internal pod security groups, or reference them in any ingress or egress rules. 
  - `tigera-trust-host-enforcement`
  - `tigera-has-host-enforcement`
  - `tigera-cluster-{cluster-name}-TigeraPodDefault-id`

#### Allow security group access to specific pods

The following example allows members of security group `sg-01010101010101010` to access pods with the label `role = frontend`. We annotate the NetworkPolicy with `rules.networkpolicy.tigera.io/match-security-groups: "true"`, and add a selector for the security group of the VPC endpoint.

```
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-network-policy
  namespace: default
  annotations:
    rules.networkpolicy.tigera.io/match-security-groups: "true"
spec:
  podSelector:
    matchLabels:
    role: frontend
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
          matchLabels:
               sg.aws.tigera.io/sg-01010101010101010: ""
```

#### Deny pod access to VPC resource

In the following example, we have the following resources:

- A security group that allows ingress traffic from the node security group.
  - `db-sg with id sg-01010101010101`
- A security group that allows egress traffic to the first security group, (db-sg).
  - `access-db with id sg-02020202020202`
- A database in the VPC that belongs to the first security group, (db-sg). 
  - Either Postgres on an EC2 instance, or an RDS Postgres instance.
- A pod in this cluster is allowed to access the database,`prod-app-64jh`
- A pod in this cluster is denied access to the database,`dev-app-87mf`

Because the `db-sg` security group allows ingress traffic from the node security group, all pods have access to the database by default. To let one pod access the database and prevent others from accessing it, we use these steps.

1. Ensure that the access-db allows egress traffic to `db-sg`
1. Add a rule to db-sg that allows ingress traffic from `access-db`
1. Add the `prod-app-64jh` pod to the `access-db` security group. To do this, annotate it with the ID of the `access-db` security group.

   Syntax:

   ```
   kubectl annotate pod <pod-name> aws.tigera.io/security-groups='["<sg-id>, <sg-id>"]'
   ```
   Example:

   ```
   kubectl annotate pod prod-app-64jh aws.tigera.io/security-groups='["sg-02020202020202"]'
   ```

1. To disallow other Kubernetes pods from accessing the database, remove the ingress rule that allows all traffic from the node security group from the `db-sg` security group.

1. Confirm that the `prod-app-64jh pod` can access the database.
   
   ```
   kubectl exec -ti prod-app-64jh bash
   PGCONNECT_TIMEOUT=3 PGPASSWORD=<password> psql
   --host=<endpoint-address> --port=5432 --username=<username> --dbname=dbname -c 'select 1'
   exit
   ```
1. Confirm that the `dev-app-87mf` pod cannot access the database.

   ```
   kubectl exec -ti dev-app-87mf bash
   PGCONNECT_TIMEOUT=3 PGPASSWORD=<password> psql
   --host=<endpoint-address> --port=5432 --username=<username> --dbname=dbname -c 'select 1'
   exit
   ```
### Above and beyond

- [Enable pod access to AWS metadata]({{site.baseurl}}/security/aws-integration/metadata-access)
