---
title: Enable pods to access VPC members
canonical_url: https://docs.tigera.io/v2.3/usage/aws-security-group-integration/pod-access
---

> **Note**: For VPC member resources to be available to pods in the Kubernetes cluster those
> resources must be members of appropriate security groups.  See
> [Interconnecting your VPC and cluster](/{{page.version}}/security/aws-security-group-integration/interconnection)
> for more information.
{: .alert .alert-info}


## About restricting pod access to VPC members
The following tutorial walks you through the steps to protect a VPC member so that only the desired pods have access. In this hypothetical example, we have a database running in the VPC and wish to restrict which Kubernetes pods can access it.

### Prerequisites
- *A security group* that allows inbound traffic from the node security group.
  - in this demo: `db-sg` with id `sg-01010101010101`
- *A security group* that allows outbound traffic to the first security group. *(db-sg)*
  - in this demo: `access-db` with id `sg-02020202020202`
- *A database in the VPC*: that belongs to the first security group. *(db-sg)*
  - in this demo either Postgres on an EC2 instance or an RDS Postgres instance
- *A pod* in this cluster that will be allowed to access the database
  - in this demo: `prod-app-64jh`
- *A pod* in this cluster that will not be allowed access to the database
  - in this demo `dev-app-87mf`


### Restricting pod access

Because the `db-sg` security group allows inbound traffic from the node security group, all pods have access to the database by default. To let one pod access the database and prevent others from accessing it,

1. Ensure that the `access-db` allows outboud traffic to `db-sg`

2. Add a rule to `db-sg` that allows inbound traffic from `access-db`

3. Add the `prod-app-64jh` pod to the `access-db` security group. To do this annotate it with the ID of the `access-db` security group.

   - Syntax:

     ````
kubectl annotate pod <pod-name> aws.tigera.io/security-groups='["<sg-id>, <sg-id>"]'
````

   - Example:

     ````
kubectl annotate pod prod-app-64jh aws.tigera.io/security-groups='["sg-02020202020202"]'
````

4. To disallow other Kubernetes pods from accessing the database, remove the inbound rule which allows all traffic from the node security group from the db-sg security group.


5. Confirm that the `prod-app-64jh` pod __can__ access the database

      ````
kubectl exec -ti prod-app-64jh bash
PGCONNECT_TIMEOUT=3 PGPASSWORD=<password> psql
--host=<endpoint-address> --port=5432 --username=<username> --dbname=dbname -c 'select 1'
exit
````


6. Confirm that the `dev-app-87mf` pod __cannot__ access the database.


      ````
kubectl exec -ti dev-app-87mf bash
PGCONNECT_TIMEOUT=3 PGPASSWORD=<password> psql
--host=<endpoint-address> --port=5432 --username=<username> --dbname=dbname -c 'select 1'
exit
````


