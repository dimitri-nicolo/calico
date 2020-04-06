---
title: Create tiered policies
description: Group policies together in a tier and apply RBAC to control user access.
canonical_url: /security/tiered-policy
---

### Big picture

Order tiers of **{{site.prodname}} network policies** and **Kubernetes network policies** for execution, and configure policy actions to handle traffic within policies.

### Value

Tiers allow delegation of authority over network policies. Platform and security operatives can configure policies that take precedence over policies configured by application or services owners. You can use flexible RBAC to prevent unauthorized viewing or modification of higher tiers, and still allow application or services owners to manage the detailed policies related to their workloads. This promotes self-service for modern CI/CD processes for containerization and microservices.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **Tier**
- **NetworkPolicy** and **GlobalNetwork** policy with action rules:
  - Allow
  - Deny
  - Pass
  - Log
- **Calico Enterprise API server**

### Concepts

#### Tiered policies 

{{site.prodname}} uses **tiered network policies** so platform teams can implement security guardrails for certain classes of applications. Each **{{site.prodname}} network policy** and **{{site.prodname}} global network policy** belongs to a tier. Although Kubernetes doesn’t have the concept of a tier, we automatically put Kubernetes network policies in your cluster into a default tier during installation -- so you can manage them as part of your workflow.  

#### Drag and move tiered policies in Calico Enterprise Manager 

If you are using the {{site.prodname}} Manager, you don’t have to worry about managing tier order; you just drag and move the tier left or right in the graphical sequence to order it. Because the UI also lets you preview and stage policies to test your workflow before enforcing them in production, using {{site.prodname}} Manager is recommended. (If you are not using our UI, you can still manage tier ordering and policy actions using `kubectl`). 

#### Policy evaluation order

Tiers are evaluated from left to right in the {{site.prodname}} Manager. If you are using the CLI, tiers are ordered using the **order:** field from lowest to highest value. (For example, a tier **order: 2** is evaluated before a tier **order: 50**.) Tiers without an order are placed at the end of the flow, and are treated as “infinite".

The following diagram shows how you can use RBAC to make your tiered policy workflow tamperproof, while allowing self-service that is appropropriate for Security, Ops, and Dev teams. 

![policy-evaluation]({{site.baseurl}}/images/policy-evaluation.png)

#### Policy rules in evaluation 

Rules in policies in a tier are chained together in order, and are executed one after the other. So policies in earlier tiers can take precedence over those in later tiers. Each tier can allow the traffic, deny the traffic, or pass it to the next tier for further evaluation using these **action** fields:

- **Pass** - skip to next tier
- **Allow** - allow traffic
- **Deny** - drop traffic
- **Log** - allow traffic and write iptable logs to syslog 

#### The default tier: always last

During installation, {{site.prodname}} creates a default tier with all your Kubernetes network policies, which is the last tier -- highest order with a value of nil (infinite). Although you cannot move Kubernetes network policies out of the default tier, you can add/remove {{site.prodname}} network policies from the default tier.

#### Managing network policies together

It is easy to manage both **Kubernetes network policies** alongside your **{{site.prodname}} policies** -- both follow the same deny/allow rules. Here are a few things to note about viewing and managing Kubernetes and {{site.prodname}} network policies. 

| **If you...**                                                | **Then...**                                                  |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| Use `kubectl` to create, modify, or delete Kubernetes network policies as you normally would | You will find them in the default tier with their RBAC preserved. They cannot be moved into a different tier. |
| View Kubernetes network policies in Calico Enterprise Manager | Policies names will look normal.                             |
| Create/update Kubernetes network policies using `calicoctl`  | Policies are prefixed with “knp.default” (to differentiate them from Calico Enterprise network policies). For example: `knp.default.default-deny` |
| View Calico Enterprise network policies through `kubectl`    | Calico Enterprise resource names are appended with `.p` to differentiate between Kubernetes network policies and/or the backing custom resource definition. For example: `networkpolicies.p -l` (short for projectcalico.org). |

### Before you begin...

**Required**

- A Kubernetes cluster installed and configured with [Calico Enterprise]({{site.baseurl}}/getting-started/) and [CLIs are installed and configured]({{site.baseurl}}/getting-started/clis/calicoctl/)
- If you use {{site.prodname}} Manager, users require authentication to log in [Configure user authentication to Calico Enterprise Manager]({{site.baseurl}}/getting-started/cnx/create-user-login)

### How to

- [Create, modify, or delete a tier](#create-modify-or-delete-a-tier)
- [Create, modify, or delete a Calico Enterprise global network policy in a tier](#create-modify-or-delete-a-calico-enterprise-global-network-policy-in-a-tier)
- [Create, modify, or delete Kubernetes network policies in a tier](#create-modify-or-delete-kubernetes-network-policies-in-a-tier)

#### Create, modify, or delete a tier

Use `kubectl` to create, modify or delete tiers. For example, to create a tier called “devops”:

1. Create a file called, `devops.yaml` with the following content.

   ```
   apiVersion: projectcalico.org/v3
   kind: Tier
   metadata:
     name: internal-access
   spec:
     order: 100
   ```
1. Apply the YAML using `kubectl`.

   ```
   kubectl apply -f devops.yaml
   ```

You can also create, modify, delete or reorder tiers in the {{site.prodname}} Manager web application.

#### Create, modify, or delete a Calico Enterprise global network policy in a tier

Use `kubectl` to create, modify, or delete {{site.prodname}} GlobalNetworkPolicy and {{site.prodname}} NetworkPolicy. The policy name must begin with the name of an existing tier, followed by a period “.” (as in `devops.isolate-production`).

For example, to create a {{site.prodname}} GlobalNetworkPolicy called “isolate-production” in the “devops” tier:

1. Create a file called, `isolate-production.yaml` with the following content.

   ```
   apiVersion: projectcalico.org/v3
   kind: GlobalNetworkPolicy
   metadata:
     name: devops.isolate-production
   spec:
     tier: devops
     selector: env == 'production'
     types:
     - Ingress
     ingress:
     - action: Deny
       source:
         selector: env == 'dev'
   ```      
1. Apply the YAML using `kubectl`.

   ```
   kubectl apply -f isolate-production.yaml
   ```

You can also create, modify, delete or reorder policies in the {{site.prodname}} Manager web application.

#### Create, modify, or delete Kubernetes network policies in a tier

Use `kubectl` to create, modify, or delete Kubernetes network policies as you normally would. Kubernetes network policies are always in the default tier, no further action is required.

### Above and beyond

- [Tier]({{site.baseurl}}/reference/resources/tier)
- [Configure RBAC for tiered policies]({{site.baseurl}}/security/rbac-tiered-policies)
- [Preview policy impacts]({{site.baseurl}}/security/policy-impact-preview)
- [Create staged policies]({{site.baseurl}}/security/staged-network-policies)
