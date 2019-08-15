1. Issue the following command to create the `calico-monitoring` namespace.

   ```bash
   kubectl create namespace calico-monitoring
   ```

1. Use the following command to add the secret to Kubernetes.

   ```bash
   kubectl create secret generic cnx-pull-secret -n calico-monitoring --from-file=.dockerconfigjson=$HOME/.docker/config.json --type kubernetes.io/dockerconfigjson
   kubectl create secret generic cnx-pull-secret -n kube-system --from-file=.dockerconfigjson=$HOME/.docker/config.json --type kubernetes.io/dockerconfigjson
   ```

   It should return the following.

   ```bash
   secret "cnx-pull-secret" created
   secret "cnx-pull-secret" created
   ```
   {: .no-select-button}

{% if include.platform == "docker-ee" %}
1. Continue to [Docker Enterprise/UCP Installation](#install-docker-ucp).
{% else %}

1. Continue to [Installing {{site.prodname}}](#install-cnx).
{% endif %}