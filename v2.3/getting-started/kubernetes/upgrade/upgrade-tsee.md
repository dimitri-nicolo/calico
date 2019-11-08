---
title: Upgrading Calico Enterprise from an earlier release
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/upgrade/upgrade-tsee
---

## Prerequisite

Ensure that your Kubernetes cluster is already running an earlier version of {{site.prodname}}.

If your cluster has a Calico installation, follow the [Upgrading a Kubernetes cluster with Calico to {{site.prodname}} guide](./upgrade-to-tsee) 
instead.

## Upgrading to {{page.version}} {{site.prodname}}

If you used the manifests provided on the [Tigera documentation site](https://docs.tigera.io/) 
to install {{site.prodname}}, re-install using the {{page.version}} {{site.prodname}} instructions
and manifests for your specific installation type. To avoid unneccessary service impact, ensure you modify the various 
manifests to include any changes that were previously made for your current deployment *prior* to applying the new 
manifests.

If you are upgrading from a pre-v2.3 release of {{site.prodname}}, or you previously modified your manifests to use the
pre-v2.3 RBAC behavior, follow the additional instructions below to ensure RBAC behavior for tiered policy is as expected.

### <a name="upgrading-pre23"></a>Upgrading from a pre-v2.3 {{site.prodname}} release

The RBAC configuration model has changed between the v2.2 and v2.3 releases of {{site.prodname}}.
When upgrading from a pre-v2.3 release of {{site.prodname}} it is necessary to perform an action to either
maintain pre-v2.3 RBAC behavior for Calico tiered policy, or to migrate to the new RBAC behavior.
 
Pre-v2.3 RBAC behavior was as follows:
-  A user must have `get` access to a tier in order to do *any* management operations on the Calico policy types 
   (NetworkPolicy and GlobalNetworkPolicy)
-  The permissions a user has for Calico Network Policies or Calico Global Network Policies would extend across all
   tiers they have `get` access to, and they would have no access for these resource types in tiers that they do not
   have `get` access to
-  It was not possible to have different policy RBAC permissions across multiple tiers.

The new RBAC behavior is as follows:
-  A user must have `get` access to a tier in order to do *any* management operations on the Calico policy types 
   (NetworkPolicy and GlobalNetworkPolicy)
-  The permissions a user has for Calico Network Policies or Calico Global Network Policies can be granted across all
   tiers or on specific tiers. In both cases, the user still requires `get` access for the tier
-  The tiered network policy RBAC is specified using the new pseudo resource types `tier.globalnetworkpolicies` and 
   `tier.networkpolicies`

Prior to upgrading, follow one of the sections below to ensure minimal service impact for users during upgrade:
-  [Maintaining the pre-v2.3 behavior](#v23-rbac), or
-  [Migrating to the new tier-granularity RBAC](#per-tier-rbac)

#### <a name="v23-rbac"></a>Maintaining the pre-v2.3 behavior

##### During the upgrade to {{site.prodname}} {{page.version}}

If you are following the documented installation instructions, download the `cnx.yaml`, and before applying it modify 
`ClusterRole "ee-calico-tiered-policy-passthru"` to specify the resource kinds `tier.networkpolicies` and 
`tier.globalnetworkpolicies`. The resource definition should be:

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: ee-calico-tiered-policy-passthru
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies","tier.globalnetworkpolicies"]
  verbs: ["*"]
```

This modified ClusterRole bypasses the per-tier policy for the psuedo-resource kinds (`tier.networkpolicies` and
`tier.globlalnetworkpolicies`). RBAC for the tiered policy is then configured as per pre-v2.3 using the real resource
kinds `networkpolicies` and `globalnetworkpolicies`.

#### <a name="per-tier-rbac"></a>Migrating to the new per-tier-granularity RBAC

If you are upgrading from pre-v2.3 and you wish to start using the new per-tier granularity RBAC for Calico policy,
perform the following migration steps:

##### Before upgrading to {{site.prodname}} {{page.version}}

Modify any `Role` and `ClusterRole` that refer to Calico policy resource types. Update the resources to include the 
real Calico resource type (`networkpolicies` and `globalnetworkpolicies`) *and* the associated pseudo-resource types 
(`tier.networkpolicies` and `tier.globalnetworkpolicies`).

For example, the following ClusterRole:

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-only-globalnetworkpolicy
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["globalnetworkpolicies"]
  verbs: ["get","list","watch"]
```

would have the `resources` updated to include both `globalnetworkpolicies` and `tier.globalnetworkpolicies`:

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-only-globalnetworkpolicy
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["globalnetworkpolicies","tier.globalnetworkpolicies"]
  verbs: ["get","list","watch"]
```

##### Upgrading

When upgrading, ensure the cnx.yaml manifest has the {{page.version}} default settings for the `ClusterRole "ee-calico-tiered-policy-passthru"`.
This resource should have the `resources` set to include only the real resource kinds `networkpolicies` and `globalnetworkpolicies`.
These are the settings in the standard {{page.version}} {{site.prodname}} manifests, and so provided you have not previously modified them
no changes should be required.

##### Post upgrade

After upgrade, RBAC for Calico policy will be determined using the psuedo-resource kinds `tier.globalnetworkpolicies` and
`tier.networkpolicies`. At this point it is possible to update your RBAC definitions to utilize the per-tier granularity
that is available using the tier wildcard format of the resource names (`<tiername>.*`, e.g. use a resource name `net-sec.*`).

Using the new tiered policy RBAC, the real Calico resource kinds `networkpolicies` and `globalnetworkpolicies`, have
full permissions for all users, and access to the policy resources is controlled using the pseudo resource types. Therefore, after
upgrade you may remove the real resource kinds from the users `Role` and `ClusterRole` definitions to tidy up the resources. 
This is not necessary though, and omitting this step will not impact RBAC.

For example, the previous `ClusterRole` example may be further updated as follows to remove the resource kind `globalnetworkpolicies`:

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-only-globalnetworkpolicy
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tier.globalnetworkpolicies"]
  verbs: ["get","list","watch"]
```

