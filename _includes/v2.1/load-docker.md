{% if include.yaml == "calico" and include.orchestrator == "kubernetes" %}
### Pulling the images from another private registry
{% endif %}

**Prerequisite**: Ensure that you have the [`config.json` file with the private Tigera registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials).

1. From a terminal prompt, use the following command to either create or open the `~/.docker/config.json` file.

   ```bash
   vi ~/.docker/config.json
   ```

1. Depending on the existing contents of the file, edit it in one of the following ways.

   - **New file**: Paste in the entire contents of the `config.json` file from Tigera.

   - **Existing file without quay.io object**: Add the following lines from the `config.json` inside the `"auth"` object.

     ```json
     "quay.io": {
       "auth": "<ROBOT-TOKEN-VALUE>",
       "email": ""
     }
     ```

   - **Existing file with quay.io object**: Add the following lines from the `config.json` inside the `"quay.io"` object.

     ```json
     "auth": "<ROBOT-TOKEN-VALUE>",
     "email": ""
     ```

1. Save and close the file.

{% if include.yaml == "calico" %}

1. Use the following commands to pull the {{site.prodname}} images from the Tigera
   registry and to pull the Prometheus images.

   ```bash
   docker pull {{page.registry}}{{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker pull {{page.registry}}{{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker pull {{page.registry}}{{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker pull {{page.registry}}{{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker pull {{page.registry}}{{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker pull {{page.registry}}{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}}
   docker pull quay.io/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker pull quay.io/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker pull quay.io/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker pull quay.io/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker pull quay.io/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   ```

1. Retag the images with the name of your private registry.

   ```bash
   docker tag {{page.registry}}{{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}} <YOUR-REGISTRY>/tigera/cnx-apiserver:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}} <YOUR-REGISTRY>/tigera/cnx-queryserver:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}} <YOUR-REGISTRY>/tigera/cnx-manager:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}} <YOUR-REGISTRY>/tigera/cnx-manager-proxy:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker tag {{page.registry}}{{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}} <YOUR-REGISTRY>/tigera/cnx-node:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker tag {{page.registry}}{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}} <YOUR-REGISTRY>/tigera/typha:{{site.data.versions[page.version].first.components["typha"].version}}
   docker tag quay.io/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}} <YOUR-REGISTRY>/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker tag quay.io/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}} <YOUR-REGISTRY>/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker tag quay.io/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}} <YOUR-REGISTRY>/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker tag quay.io/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}} <YOUR-REGISTRY>/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker tag quay.io/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}} <YOUR-REGISTRY>/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   ```
   > **Note**: We recommend changing just the name of the registry (`<YOUR-REGISTRY>`)
   > when retagging the images, as shown above and below.
   {: .alert .alert-info}

1. Push the images to your private registry.

   ```bash
   docker push <YOUR-REGISTRY>/tigera/cnx-apiserver:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-queryserver:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-manager:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-manager-proxy:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-node:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker push <YOUR-REGISTRY>/tigera/typha:{{site.data.versions[page.version].first.components["typha"].version}}
   docker push <YOUR-REGISTRY>/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker push <YOUR-REGISTRY>/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker push <YOUR-REGISTRY>/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker push <YOUR-REGISTRY>/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker push <YOUR-REGISTRY>/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   ```

   > **Important**: Do not push the private {{site.prodname}} images to a public registry.
   {: .alert .alert-danger}

{% else %}

1. Use the following commands to pull the `{{include.yaml}}` image from the Tigera
   registry.

   ```bash
   docker pull {{site.imageNames[include.yaml]}}:{{site.data.versions[page.version].first.components[include.yaml].version}}
   ```

1. Retag the image with the name of your private registry.

   ```bash
   docker tag {{site.imageNames[include.yaml]}}:{{site.data.versions[page.version].first.components[include.yaml].version}} <YOUR-REGISTRY>/tigera/{{include.yaml}}:{{site.data.versions[page.version].first.components[include.yaml].version}}
   ```
   > **Note**: We recommend changing just the name of the registry (`<YOUR-REGISTRY>`),
   > as shown above. This will make it easier to complete the instructions that follow.
   {: .alert .alert-info}

1. Push the images to your private registry.

   ```bash
   docker push <YOUR-REGISTRY>/tigera/{{include.yaml}}:{{site.data.versions[page.version].first.components[include.yaml].version}}
   ```

{% endif %}

{% if include.orchestrator == "kubernetes" and include.yaml == "calico" %}

1. Push the credentials of your private repository up to Kubernetes as a [secret](https://kubernetes.io/docs/concepts/containers/images/#creating-a-secret-with-a-docker-config)
named `cnx-pull-secret` in the `kube-system` namespace.

1. Continue to [Installing {{site.prodname}}](#install-cnx).

{% endif %}

{% if include.orchestrator == "openshift" %}

1. Modify the Ansible inventory file to include [the location of the registry](https://docs.openshift.com/container-platform/3.10/install/configuring_inventory_file.html#advanced-install-configuring-registry-location) which
   contains the private images and [the credentials needed to authenticate to it](https://github.com/openshift/openshift-ansible/blob/master/inventory/hosts.example#L223).

{% endif %}

{% if include.yaml != "calico" %}

1. Open the YAML that matches your datastore type in a new tab.

   - [etcd]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/{{include.yaml}}.yaml){:target="_blank"}

   - [Kubernetes API datastore]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/{{include.yaml}}.yaml){:target="_blank"}

1. Copy the contents, paste them into a new file, and save the file as {{include.yaml}}.yaml. This is what subsequent instructions will refer to.

{% include {{page.version}}/cnx-cred-sed.md yaml=include.yaml %}

1. Apply the YAML file.

   ```bash
   kubectl apply -f {{include.yaml}}.yaml
   ```

{% endif %}
