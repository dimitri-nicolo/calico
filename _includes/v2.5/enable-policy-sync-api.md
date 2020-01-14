{% if include.feature == "ingress-flows" %}
  {% assign feature = "Ingress flow log correlation" %}
{% else %}
  {% assign feature = "Application layer policy" %}
{% endif %}

**Prerequisites**:

 - [{{site.tseeprodname}} installed](/{{page.version}}/getting-started/kubernetes/installation/)

{{feature}} requires the Policy Sync API to be enabled on Felix. To do this cluster-wide, modify the `default`
FelixConfiguration to set the field `policySyncPathPrefix` to `/var/run/nodeagent`.

```bash
kubectl patch felixconfiguration default --type='merge' -p '{"spec":{"policySyncPathPrefix":"/var/run/nodeagent"}}'
```
