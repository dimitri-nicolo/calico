---
title: Overview
canonical_url: https://docs.tigera.io/v2.3/usage/aws-security-group-integration/
---



{{site.prodname}} integrates AWS security groups and network policy,
enforcing granular access control between Kubernetes pods and AWS VPC resources.

If you have
[enabled the AWS Security Group integration](/{{page.version}}/reference/other-install-methods/kubernetes/installation/aws-sg-integration),
{{site.prodname}} allows you to control communications between
[VPC members and pods](/{{page.version}}/security/aws-security-group-integration/vpc-member-access) and between
[pods and VPC members](/{{page.version}}/security/aws-security-group-integration/pod-access).


By default Kubernetes pods in the cluster along with EC2 and RDS instances in the VPC
are placed in security groups allowing communication between them.  See
[Interconnecting your VPC and cluster](/{{page.version}}/security/aws-security-group-integration/interconnection)
for more details.


VPC members must have a network interface and be able to be added to a
[security group](https://docs.aws.amazon.com/vpc/latest/userguide/VPC_SecurityGroups.html).
This includes but is not limited to [interface VPC endpoints](https://docs.aws.amazon.com/vpc/latest/userguide/vpce-interface.html) as well as
[RDS](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.html)
and [EC2](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/Instances.html)
instances.


 If enabled, {{site.prodname}} checks the rules of the pod’s security groups first to see if a connection should be blocked.
 If the connection is not blocked by the security group rules, the traffic is processed by the Calico
 and Kubernetes network policy.

> **Note**: The Security Group policy only blocks or passes traffic to the next tier, it does not allow it explicitly.
{: .alert .alert-info}



