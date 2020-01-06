{% if include.orch != "openshift" %}
  {% assign cli = "kubectl" %}
{% else %}
  {% assign cli = "oc" %}
{% endif %}
1. Issue the following command to create the `calico-monitoring` namespace.

   ```bash
   {{cli}} create namespace calico-monitoring
   ```

1. Use the following command to add the secret to Kubernetes.

   ```bash
   {{cli}} create secret generic cnx-pull-secret -n calico-monitoring --from-file=.dockerconfigjson=$HOME/.docker/config.json --type kubernetes.io/dockerconfigjson
   {{cli}} create secret generic cnx-pull-secret -n kube-system --from-file=.dockerconfigjson=$HOME/.docker/config.json --type kubernetes.io/dockerconfigjson
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

1. Continue to [Installing {{site.tseeprodname}}](#install-cnx).
{% endif %}
