## Applying your license key

1. [Install calicoctl]({{site.baseurl}}/{{page.version}}/usage/calicoctl/install).

1. [Configure calicoctl to connect to your datastore]({{site.baseurl}}/{{page.version}}/usage/calicoctl/configure).

1. Use the following command to apply your [license key]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/licensekey).

   **Command**
   ```
   calicoctl apply -f <customer-name>-license.yaml
   ```

   **Example**
   ```
   calicoctl apply -f awesome-corp-license.yaml
   ```

1. Confirm that the license was applied:

   ```
   calicoctl get license
   ```
