---
title: Enabling VPC members to access pods
redirect_from: latest/security/aws-security-group-integration/vpc-member-access
canonical_url: https://docs.tigera.io/v2.3/usage/aws-security-group-integration/vpc-member-access
---

> **Note**: For pods in the Kubernetes cluster to be available to VPC member resources those
> resources must be members of appropriate security groups.  See
> [Interconnecting your VPC and cluster](/{{page.version}}/security/aws-security-group-integration/interconnection)
> for more information.
{: .alert .alert-info}

## About allowing VPC members to access pods

Pods selected by a network policy will deny ingress traffic from VPC members by default.

To allow VPC members to connect to pods, modify the NetworkPolicy.

Modify the manifest YAML or edit the network policy directly with  `kubectl edit <network-policy-name>`

 - Annotate the NetworkPolicy with `rules.networkpolicy.tigera.io/match-security-groups: "true"`
 - Add a selector for the security group of the VPC member.

The following  example allows members of security group `sg-01010101010101010` to access pods with the label `role = frontend`


````
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
````


