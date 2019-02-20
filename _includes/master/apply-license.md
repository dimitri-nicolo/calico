{% if include.init != "openshift" %}
  {% assign cli = "kubectl" %}
{% else %}
  {% assign cli = "oc" %}
{% endif %}

## Applying your license key

1. Use the following command to apply your [license key]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/licensekey).

   **Command**
   ```
   {{cli}} apply -f - < <customer-name>-license.yaml
   ```

   **Example**
   ```
   {{cli}} apply -f - < awesome-corp-license.yaml
   ```
   {: .no-select-button}

1. Confirm that the license was applied:

   ```
   {{cli}} get licensekey
   ```

1. Continue to [Installing metrics and logs](#installing-metrics-and-logs)
