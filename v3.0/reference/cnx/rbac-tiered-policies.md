---
title: RBAC on tiered Policies
---

The {{site.prodname}} API server adds the ability to manage tiered
policies as Kubernetes resources. This allows administrators to apply
Kubernetes RBAC Authorization APIs to CNX resources.

### Admin permissions

By default, admin users have sufficient permission to access all the resources
in the {{site.prodname}} UI.

### Granting new users {{site.prodname}} UI permissions

Under Kubernetes, newly created users are not granted RBAC permissions. In
order to give a user sufficient permissions to use the {{site.prodname}} UI,
bind a user to the `cluster-admin` role:

```
kubectl create clusterrolebinding permissive-binding \
    --clusterrole=cluster-admin \
    --user=<USER>
```

As result of this operation, the user now has full permission to access all
resources in the {{site.prodname}} UI.

### Fine-grained permissions

#### Policies and tiers

In {{site.prodname}}, `GlobalNetworkPolicy` and `NetworkPolicy` resources
are associated with a specific tier. In addition to the permissions associated
with resources, {{site.prodname}} also gives you the ability to set permissions
for tiers.

Giving a non-admin user 'read' permission for a tier is a prerequisite for
performing any operations on Global/NetworkPolicies in that tier. Once a
non-admin user is given 'read' tier permission, their ability to operate on
Global/NetworkPolicies is limited by their resource permissions (e.g.
`apiGroup: "networking.k8s.io"`, `resource: "networkpolicies"`).

#### The default tier

Policies created by the underlying orchestration integration such as Kubernetes
are placed in the `default` tier. In order to use the {{site.prodname}} UI,
users must be given 'read' access to the `default` tier.

#### Associating a resource with a tier

For details on creating a [tier]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier)
resource and adding a Global/NetworkPolicy to that tier, refer to the
[Tiered Policy Demo]({{site.baseurl}}/{{page.version}}/getting-started/cnx/tiered-policy-cnx/).

#### Restricting {{site.prodname}} permissions

In order to limit permissions for non-admin users for resources
outside the `default` tier, use the following steps as a starting point
to implement your security model:

1. Download the [`min-rbac.yaml` manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/demo-manifests/min-rbac.yaml).

1. Run the following command to replace `<USER>` with the `name/email` of
   the non-admin user you are providing permissions to:

   ```
   sed -i -e 's/<USER>/<name/email>/g' min-rbac.yaml
   ```

1. Use the following command to install the bindings:

   ```
   kubectl apply -f min-rbac.yaml
   ```

Add to the roles and bindings according to your specific security requirements.

### Example fine-grained permissions

```
# Users:
- jane (non-admin)
- kubernetes-admin (admin)
```

User 'jane' is forbidden from reading policies in any tier (default, and
net-sec in this case):

```
# kubectl get networkpolicies.p
Error from server: (Forbidden) Policy operation is associated with tier default. User "jane" cannot list tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)

# kubectl get networkpolicies.p -l projectcalico.org/tier==net-sec
Error from server: (Forbidden) Policy operation is associated with tier net-sec. User "jane" cannot list tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)
```

> **Note**: The appended '.p' with the networkpolicies resource in the kubectl
  command. That is short for "networkpolicies.projectcalico.org" and is needed
  to differentiate from the Kubernetes namesake NetworkPolicy resource.
{: .alert .alert-info}

> **Note**: Currently, the tier collection on a Policy resource through the
  kubectl client (pre 1.9) of the APIs is implemented using labels because
  kubectl lacks field selector support. The label used for tier collection
  is "projectcalico.org/tier". When a label selection is not specified, the
  server defaults the collection to the `default` tier. Field selection based
  policy collection is enabled at API level. spec.tier is the field to select
  on for the purpose.
{: .alert .alert-info}

kubernetes-admin gives user 'jane' permission to read tier 'default':

```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  # "namespace" omitted since ClusterRoles are not namespaced
  name: tier-default-reader
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["default"]
  verbs: ["get"]

---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-tier-default-global
subjects:
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-default-reader
  apiGroup: rbac.authorization.k8s.io
```

With the above user jane is able to read policies in default tier
```
# kubectl get networkpolicies.p
No resources found.
```
But still not net-sec
```
# kubectl get networkpolicies.p -l projectcalico.org/tier==net-sec
Error from server: (Forbidden) Policy operation is associated with tier net-sec. User "jane" cannot list tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)
```

To provide permission to user 'jane' to read policies under 'net-sec' tier,
use the following clusterrole and clusterrolebindings.

kubernetes-admin gives user 'jane' the permission to read tier 'net-sec':
```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  # "namespace" omitted since ClusterRoles are not namespaced
  name: tier-net-sec-reader
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["net-sec"]
  verbs: ["get"]

---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-tier-net-sec-global
subjects:
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-net-sec-reader
  apiGroup: rbac.authorization.k8s.io
```
