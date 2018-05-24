{% if include.yaml == "calico" %}
### Pulling the images from Tigera's private registry
{% endif %}

**Prerequisite**: Ensure that you have the [`config.json` file with the private Tigera registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials).

1. From a terminal prompt, navigate to the location of the `config.json` file.

1. Strip the spaces, tabs, carriage returns, and newlines from the `config.json`
   file and base64 encode the string. If you're on Linux, you can use the
   following command.

   ```bash
   cat config.json | tr -d '\n\r\t ' | base64 -w 0
   ```

1. Open a new file in your favorite editor called `cnx-pull-secret.yml`.

   ```bash
   vi cnx-pull-secret.yml
   ```

1. Paste in the following YAML.

   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: cnx-pull-secret
     namespace: kube-system
   data:
     .dockerconfigjson: <BASE64-STRING>
   type: kubernetes.io/dockerconfigjson
   ```

1. Replace `<BASE64-STRING>` with the base64-encoded config.json returned to your
   shell previously.

1. Save and close the file.

1. Use the following command to add the secret to Kubernetes.

   ```bash
   kubectl create -f cnx-pull-secret.yml
   ```

   It should return the following.

   ```bash
   secret "cnx-pull-secret" created
   ```
   
{% if include.yaml == "calico" %}
1. Continue to [Installing {{site.prodname}}](#install-cnx).
{% endif %}

{% if include.yaml != "calico" %}

1. Use the YAML that matches your datastore type to deploy the `{{include.yaml}}` container to your nodes.

   - **etcd**

     ```
     kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/{{include.yaml}}.yaml
     ```

     > **Note**: You can also
     > [view the YAML in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/{{include.yaml}}.yaml){:target="_blank"}.
     {: .alert .alert-info}

   - **Kubernetes API datastore**

     ```
     kubectl apply -f \
     {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/{{include.yaml}}.yaml
     ```

     > **Note**: You can also
     > [view the YAML in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/{{include.yaml}}.yaml){:target="_blank"}.
     {: .alert .alert-info}

**Next step**:

[Configure `{{include.yaml}}` to connect to your datastore](/{{page.version}}/usage/{{include.yaml}}/configure/).

{% endif %}
