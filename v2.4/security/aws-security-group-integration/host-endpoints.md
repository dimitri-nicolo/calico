---
title: Host endpoints
canonical_url: https://docs.tigera.io/v2.3/usage/aws-security-group-integration/host-endpoints
---

User host endpoints are not supported.
For each endpoint resource in the VPC, the cloud-controller will create a Host Endpoint resource.
These host endpoints are managed by the cloud-controller and cannot be changed. Because the
cloud-controller manages host endpoints, additional host endpoints are not supported.


> **Important**: If a user creates additional host endpoints only one of the host endpoints will be used.
> This means that all policies enforced by the cloud-controller could be effectively bypassed.
{: .alert .alert-danger}

> **Note**:
> By default {{site.tseeprodname}} configures RBAC to prevent creation or modification of host
> endpoints.  We strongly recommend you do not modify these settings.
{: .alert .alert-info}


Depending on intent and the targeted resource there are several alternatives
to host endpoints available:

- Configure the security groups that are applied to the instances to police traffic to nodes.
- For AWS resources: apply labels to host endpoints for selecting  by AWS Security Group
- For other resources create
[GlobalNetworkSets](/{{page.version}}/reference/calicoctl/resources/globalnetworkset)
and use custom labels for selection in policy.

