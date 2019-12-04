---
title: Interconnect your VPC and cluster
canonical_url: https://docs.tigera.io/v2.3/usage/aws-security-group-integration/interconnection
---


## About pod and VPC connections

By default, every pod in your Kubernetes cluster belongs the `TIGERA_DEFAULT_SECURITY_GROUP` this gives each pod the same access as the node on which it runs.

To override the default security group(s) for a particular pod, you can specify one
or more new security groups using an annotation. Once you annotate a pod, {{site.prodname}}
uses the security group(s) in the annotation instead of the default security group(s).

Adding this annotation does not remove the pod from the default pod security group.
This allows you to establish fine-grained control over communications from pods to
VPC members.

See [Enabling pods to access VPC members](/{{page.version}}/security/aws-security-group-integration/pod-access) for more information.


## Connecting VPC members to the Kubernetes cluster

Add VPC members that interoperate with your Kubernetes
cluster to the `tigera-trust-host-enforcement` security group.


> **Note**: {{site.prodname}} adds __EC2__ and __RDS__ instances to the tigera-trust-host-enforcement
security group automatically.
{: .alert .alert-info}


See
[Enabling VPC members to access pods](/{{page.version}}/security/aws-security-group-integration/vpc-member-access) for more information.



## Internal cluster connections

By default, every pod in your Kubernetes cluster belongs to the `TIGERA_POD_SECURITY_GROUP`
this allows pod to pod communication regardless of Security Group policies.

To manage pod to pod communication use regular network policies.





> **Important** {{site.prodname}} creates and uses three security groups internally:
>
> - _tigera-trust-host-enforcement_
> - _tigera-has-host-enforcement_
> - _tigera-cluster-{cluster-name}-TigeraPodDefault-id_
>
> Do not modify these security groups or reference them in any inbound or outbound rules.
{: .alert .alert-danger}


