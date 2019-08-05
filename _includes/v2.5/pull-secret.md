{% if include.orch != "openshift" %}
  {% assign cli = "kubectl" %}
{% else %}
  {% assign cli = "oc" %}
{% endif %}
1. Issue the following command to create the `calico-monitoring` namespace.

   ```bash
   {{cli}} create namespace calico-monitoring
   ```

1. Strip the spaces, tabs, carriage returns, and newlines from the `config.json`
   file; base64 encode the string; and save it as an environment variable called `SECRET`.
   If you're on Linux, you can use the following command.

   ```bash
   SECRET=$(cat ~/.docker/config.json | tr -d '\n\r\t ' | base64 -w 0)
   ```

1. Use the following command to create a YML file called `cnx-pull-secret.yml`
   containing the base-64 encoded string.

   ```bash
   cat > cnx-pull-secret.yml <<EOF
   apiVersion: v1
   kind: Secret
   metadata:
     name: cnx-pull-secret
     namespace: kube-system
   data:
     .dockerconfigjson: ${SECRET}
   type: kubernetes.io/dockerconfigjson
   ---
   apiVersion: v1
   kind: Secret
   metadata:
     name: cnx-pull-secret
     namespace: calico-monitoring
   data:
     .dockerconfigjson: ${SECRET}
   type: kubernetes.io/dockerconfigjson
   EOF
   ```

1. Use the following command to add the secret to Kubernetes.

   ```bash
   {{cli}} create -f cnx-pull-secret.yml
   ```

   It should return the following.

   ```bash
   secret "cnx-pull-secret" created
   secret "cnx-pull-secret" created
   ```
   {: .no-select-button}

{% if include.platform == "docker-ee" %}
1. Continue to [Docker Enterprise/UCP Installation](#install-docker-ucp).
{% elsif include.orch != "openshift" %}

1. Continue to [Installing {{site.prodname}}](#install-cnx).
{% endif %}
