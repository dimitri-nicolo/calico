## Applying your license key

{% if include.init == "openshift" %}

1. [Install calicoctl]({{site.baseurl}}/{{page.version}}/usage/calicoctl/install#installing-calicoctl-as-a-kubernetes-pod).

   > **Note**: We highly recommend you install calicoctl as a Kubernetes pod in OpenShift.
   This ensures that you are using the latest version of calicoctl and its accompanying configuration.
   If you choose to [install calicoctl as a binary on a single host]({{site.baseurl}}/{{page.version}}/usage/calicoctl/install#installing-calicoctl-as-a-binary-on-a-single-host),
   we recommend you uninstall any versions of calicoctl that may have shipped alongside OpenShift with the following commands.
   ```
   rm /usr/local/bin/calicoctl
   rm /etc/calico/calicoctl.cfg
   ```
   {: .alert .alert-info}

{% else %}

1. Install `calicoctl`.  We recommend installing `calicoctl` as a pod -
   follow [these instructions]({{site.baseurl}}/{{page.version}}/usage/calicoctl/install) to do so.

   If you are not running `calicoctl` as a pod, [configure it to connect to your datastore]({{site.baseurl}}/{{page.version}}/usage/calicoctl/configure).

{% if include.platform == "eks" %}
   Please note that the EKS installation uses the Kubernetes Datastore Driver.
{% endif %}

{% endif %}

1. Use the following command to apply your [license key]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/licensekey).

   **Command**
   ```
   calicoctl apply -f - < <customer-name>-license.yaml
   ```

   **Example**
   ```
   calicoctl apply -f - < awesome-corp-license.yaml
   ```

1. Confirm that the license was applied:

   ```
   calicoctl get license
   ```

1. Continue to [Installing the {{site.tseeprodname}} Manager](#install-cnx-mgr).
