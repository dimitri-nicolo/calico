## Applying your license key

1. [Install calicoctl]({{site.baseurl}}/{{page.version}}/getting-started/calicoctl/install#installing-calicoctl-as-a-kubernetes-pod).

1. Use the following command to apply your [license key]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/licensekey).

   **Command**
   ```
   calicoctl apply -f - < <customer-name>-license.yaml
   ```

   **Example**
   ```
   calicoctl apply -f - < awesome-corp-license.yaml
   ```
   {: .no-select-button}

1. Confirm that the license was applied:

   ```
   calicoctl get license
   ```

1. Continue to [Installing metrics and logs](#installing-metrics-and-logs)
