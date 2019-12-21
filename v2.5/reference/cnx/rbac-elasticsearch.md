---
title: Configuring Tigera Secure EE RBAC for Elasticsearch
---

The {{site.prodname}} allows administrators the ability to manage access to Elasticsearch indices from the UI.
This effectively allows administrators to manage a user's UI access to flow logs, audit logs, and intrusion detection events. If a user
does not have access to a specific Elasticsearch index, then when they navigate to a page that uses Elasticsearch queries on that index,
the page will not display any data from that index.

### Elasticsearch indexes and RBAC

In {{site.prodname}}, resources are associated with the Kubernetes API group `lma.tigera.io`.

| Elasticsearch Index          | Kubernetes RBAC resource name | Description                                                                                                                     |
|------------------------------|-------------------------------|---------------------------------------------------------------------------------------------------------------------------------|
| tigera_secure_ee_flows       | flows                         | Access to flow logs                                                                                                             |
| tigera_secure_ee_audit*      | audit*                        | Access both EE and K8s audit logs. The UI currently uses this query for searching both Kube and EE audit logs at the same time. |
| tigera_secure_ee_audit_ee    | audit_ee                      | Access to EE Audit logs                                                                                                         |
| tigera_secure_ee_audit_kube  | audit_kube                    | Access to K8s Audit logs                                                                                                        |
| tigera_secure_ee_events      | events                        | Access to intrusion detection events                                                                                            |

Each Elasticsearch index used within Tigera Secure EE is mapped to a specific RBAC resource name within the `lma.tigera.io` API group.

> **Note**: The `lma.tigera.io` API group is only used for RBAC and is not backed by an actual API.
{: .alert .alert-info}

### Users with custom permissions

To apply custom Elasticsearch index permissions to a user, create a `ClusterRole` that lists the RBAC resource names corresponding
to the Elasticsearch indexes you want that user to access. For example, the `ClusterRole` below allows access to Tigera Secure audit logs only.

```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: audit-ee-only
rules:
- apiGroups: ["lma.tigera.io"]
  resources: ["index"]
  resourceNames: ["audit_ee"]
  verbs: ["get"]
```

Once you have a `ClusterRole` with the right Elasticsearch index access permissions, create a `ClusterRoleBinding` with it, binding
the role to the desired user. Below is an example `ClusterRoleBinding` to attach the above cluster role to a user:

```
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: bob-es-access
subjects:
- kind: User
  name: bob
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: audit-ee-only
  apiGroup: rbac.authorization.k8s.io
```

Note: if you want to provision a user with the minimal Tigera Secure UI permissions, please refer to [Minimum permissions for all UI users](/{{page.version}}/reference/cnx/rbac-tiered-policies#minimum-permissions-for-all-ui-users).

### Verify user access to Elasticsearch indexes

Once you have `ClusterRole`s and `ClusterRoleBinding`s setup, you can verify user access to Elasticsearch indexes by creating a `SubjectAccessReview`.
Creating a `SubjectAccessReview` returns YAML output that tells you whether the user is authorized.

In the `SubjectAccessReview` spec:
- `group` should be set to `lma.tigera.io`
- `resource` should be set to `index`
- `verb` should be set to `get`
- and `resource` should be set to a Kubernetes RBAC resource name (as defined in the table above)

Continuing with our running example `ClusterRoleBinding` that allows the user bob access only to the `audit_ee` resource (i.e., Tigera Secure audit logs),
we can verify that bob has that access with the following:

```
kubectl create -oyaml -f - <<EOF
apiVersion: authorization.k8s.io/v1
kind: SubjectAccessReview
spec:
  resourceAttributes:
    group: lma.tigera.io
    resource: index
    name: audit_ee
    verb: get
  user: bob
EOF
```

Running that verifies the `ClusterRoleBinding` is doing what we expect:

```
apiVersion: authorization.k8s.io/v1
kind: SubjectAccessReview
metadata:
  creationTimestamp: null
spec:
  resourceAttributes:
    group: lma.tigera.io
    name: audit_ee
    resource: index
    verb: get
  user: bob
status:
  allowed: true
  reason: 'RBAC: allowed by ClusterRoleBinding "bob-es-access" of ClusterRole "audit-ee-only"
    to User "bob"'
```

You would need to create a `SubjectAccessReview` resource for each Elasticsearch index. The one exception is if you wanted
to verify whether the user had access to all indexes. In a `SubjectAccessReview` resource, setting the `name` field to the empty string means "all".
We could check whether bob has access to all indexes by running:

```
kubectl create -oyaml -f - <<EOF
apiVersion: authorization.k8s.io/v1
kind: SubjectAccessReview
spec:
  resourceAttributes:
    group: lma.tigera.io
    resource: index
    name: ""
    verb: get
  user: bob
EOF
```

And this verifies that bob does not have access to all indexes:

```
apiVersion: authorization.k8s.io/v1
kind: SubjectAccessReview
metadata:
  creationTimestamp: null
spec:
  resourceAttributes:
    group: lma.tigera.io
    resource: index
    verb: get
  user: bob
status:
  allowed: false
  reason: no RBAC policy matched
```

> **Note**: If a user is given access to the `audit*` RBAC resource name, a `SubjectAccessReview`
on either `audit_ee` or `audit_kube` will return "allowed: false" due to the way the authorization is implemented.
However, a `SubjectAccessReview` on `audit*` will return "allowed: true".
{: .alert .alert-info}

### Give access to all Elasticsearch indexes

In a `ClusterRole` resource, setting the `resourceNames` field to an empty array means that everything is allowed.
For example, the following `ClusterRole` gives a bound user access to all Elasticsearch indexes:

```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: allow-all-es
rules:
- apiGroups: ["lma.tigera.io"]
  resources: ["index"]
  resourceNames: []
  verbs: ["get"]
```

Then we could allow the user `jane` all Elasticsearch indexes by creating this binding:

```
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: jane-allow-all-es
subjects:
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: allow-all-es
  apiGroup: rbac.authorization.k8s.io
```
