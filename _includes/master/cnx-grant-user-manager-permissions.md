1. Grant permission to access the {{site.prodname}} Manager to users in your cluster. Issue one of the following
   commands, replacing `<USER>` with the name of the user you wish to grant access.

   The ClusterRole `tigera-ui-user` grants permission to use the {{site.prodname}} Manager UI, view flow
   logs, audit logs, and network statistics, and access the default policy tier.

   ```
{%- if include.init == "openshift" %}
   oc adm policy add-cluster-role-to-user tigera-ui-user <USER>
{%- else %}
   kubectl create clusterrolebinding <USER>-tigera \
     --clusterrole=tigera-ui-user \
     --user=<USER>
{%- endif %}
   ```

   The ClusterRole `network-admin` grants permission to use the {{site.prodname}} Manager UI, view flow
   logs, audit logs, and network statistics, and administer all network policies and tiers.

   ```
{%- if include.init == "openshift" %}
   oc adm policy add-cluster-role-to-user network-admin <USER>
{%- else %}
   kubectl create clusterrolebinding <USER>-network-admin \
     --clusterrole=network-admin \
     --user=<USER>
{%- endif %}
   ```

   To grant access to additional tiers, or create your own roles consult the [RBAC documentation]({{site.baseurl}}/{{page.version}}/reference/cnx/rbac-tiered-policies){:target="_blank"}.
