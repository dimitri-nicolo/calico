{% if include.feature == "ingress-flows" %}
  {% assign feature = "Ingress flow log correlation" %}
{% else %}
  {% assign feature = "Application layer policy" %}
{% endif %}

**Prerequisites**:

 - [{{site.prodname}} installed](/{{page.version}}/getting-started/kubernetes/installation/)
 - [calicoctl installed](/{{page.version}}/getting-started/calicoctl/install) & [configured](/{{page.version}}/getting-started/calicoctl/configure/)

{{feature}} requires the Policy Sync API to be enabled on Felix. To do this cluster-wide, modify the `default`
FelixConfiguration to set the field `policySyncPathPrefix` to `/var/run/nodeagent`.  The following example uses `sed` to modify your
existing default config before re-applying it.

```bash
calicoctl get felixconfiguration default --export -o yaml | \
sed -e '/  policySyncPathPrefix:/d' \
    -e '$ a\  policySyncPathPrefix: /var/run/nodeagent' > felix-config.yaml
calicoctl apply -f felix-config.yaml
```
