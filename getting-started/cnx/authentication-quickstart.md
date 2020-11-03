---
title: Authentication quickstart
description: Get started quickly and log in to Calico Enterprise Manager and Kibana. 
---

### Big picture

Get started quickly with our default authentication options, create a user with access to the manager UI, and log in to the {{site.prodname}} Manager UI and Kibana using predefined roles.

### Concepts

#### Authentication defaults
Token authentication is the default authentication option for {{site.prodname}} Manager. When a service account is created, an 
associated secret is created that contains a signed bearer token for that service account. Just copy the token for the service 
account in to the Manager UI and log in.

Use basic login for the default Kibana root user.

The default login methods are always available at:
- **{{site.prodname}} Manager:** `https://<host>:<port>/login/token`. 
- **Kibana:** `https://<host>:<port>/tigera-kibana/login`. 

#### Cluster roles

In Kubernetes, **cluster roles** specify cluster scoped permissions and are bound to users via **cluster role bindings**.
We provide the following predefined roles:

**tigera-ui-user**
- Allows basic UI access to {{site.prodname}} Manager.
- View various resources in the `projectcalico.org` and `networking.k8s.io` API groups.
- Grants viewer access to Kibana.

**tigera-network-admin**
- Allows full access to {{site.prodname}} Manager.
- Create and modify various resources in the `projectcalico.org` and `networking.k8s.io` API groups.
- Grants superuser access for Kibana (including Elastic user and license management). 

### Before you begin

Make sure you have installed {{site.prodname}} using one of the [installation guides]({{site.baseurl}}/getting-started/) and have set up [access to the Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager).

### How to

> Note: For OpenShift, replace `kubectl` in the commands below with `oc`.
{: .alert .alert-info}

**Logging in to {{site.prodname}} Manager**

First, create a service account in the desired namespace:

```bash
kubectl create sa <user> -n <namespace>
```

Give the service account permissions to access the {{site.prodname}} Manager UI, and a {{site.prodname}} cluster role:

```bash
kubectl create clusterrolebinding <binding_name> --clusterrole <role_name> --serviceaccount <namespace>:<service_account>
```

where:
- **binding_name** is a descriptive name for the rolebinding.
- **role_name** is one of the default cluster roles (or a custom cluster role) specifying {{site.prodname}} UI permissions.
- **namespace** is the service account's namespace.
- **service_account** is the service account that the permissions are being associated with.

For example, the following command gives the service account jane in the default namespace network admin permissions:

```bash
kubectl create clusterrolebinding jane-access --clusterrole tigera-network-admin --serviceaccount default:jane
```

Next, get the token from the service account.
Using the running example of a service account named jane in the default namespace:

```bash
{% raw %}kubectl get secret $(kubectl get serviceaccount jane -o jsonpath='{range .secrets[*]}{.name}{"\n"}{end}' | grep token) -o go-template='{{.data.token | base64decode}}' && echo{% endraw %}
```

Now that we have the token, we can proceed to login! Go to the {{site.prodname}} UI and submit the token.

  
**Logging in to Kibana**

Connect to Kibana with the `elastic` username. Use the following command to decode the password:	

{% raw %}
```	
kubectl -n tigera-elasticsearch get secret tigera-secure-es-elastic-user -o go-template='{{.data.elastic | base64decode}}' && echo
```
{% endraw %}
Once logged in, you can configure users and their privileges from the settings page.


### Above and beyond

- [Bring your own identity provider]({{site.baseurl}}/getting-started/cnx/configure-identity-provider)
- [Configure RBAC for tiered policies]({{site.baseurl}}/security/rbac-tiered-policies)
- [Configure RBAC for Elasticsearch]({{site.baseurl}}/security/logs/rbac-elasticsearch)
