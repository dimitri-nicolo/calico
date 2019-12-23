{% comment %}NOTE: the #load-docker-CMD-cmds attrs are required to automate testing{% endcomment %}
{% if include.yaml == "calico" %}
## Setting up access to the required images
{% endif %}

> **Note**: These instructions assume that you have your own private registry,
> its credentials in a local `~/.docker/config.json` file, and
> [root privileges](https://docs.docker.com/install/linux/linux-postinstall/).
> If your user is not root or in the `docker` group, add `sudo` to the `docker` commands that follow.
{: .alert .alert-info}

1. From a terminal prompt, open the `~/.docker/config.json` file in your favorite editor.

   ```bash
   vi ~/.docker/config.json
   ```

1. Depending on the existing contents of the file, edit it in one of the following ways.

   - **No quay.io object**: Add the following lines from the `config.json` inside the `"auth"` object.

     ```json
     "quay.io": {
       "auth": "<ROBOT-TOKEN-VALUE>",
       "email": ""
     }
     ```

   - **Existing quay.io object**: Add the following lines from the `config.json` inside the `"quay.io"` object.

     ```json
     "auth": "<ROBOT-TOKEN-VALUE>",
     "email": ""
     ```

1. Save and close the file.

{% if include.yaml == "calico" %}

1. Use the following commands to pull the required {{site.tseeprodname}} images.

   ```bash
   docker pull docker.elastic.co/elasticsearch/elasticsearch:{{site.data.versions[page.version].first.components["elasticsearch"].version}}
   docker pull {{page.registry}}{{site.imageNames["kibana"]}}:{{site.data.versions[page.version].first.components["kibana"].version}}
   docker pull {{page.registry}}{{site.imageNames["calicoctl"]}}:{{site.data.versions[page.version].first.components["calicoctl"].version}}
   docker pull {{page.registry}}{{site.imageNames["calicoq"]}}:{{site.data.versions[page.version].first.components["calicoq"].version}}
   docker pull {{page.registry}}{{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker pull {{page.registry}}{{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker pull {{page.registry}}{{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker pull {{page.registry}}{{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker pull {{page.registry}}{{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker pull {{page.registry}}{{site.imageNames["es-proxy"]}}:{{site.data.versions[page.version].first.components["es-proxy"].version}}
   docker pull {{page.registry}}{{site.imageNames["fluentd"]}}:{{site.data.versions[page.version].first.components["fluentd"].version}}
   docker pull {{page.registry}}{{site.imageNames["kubeControllers"]}}:{{site.data.versions[page.version].first.components["cnx-kube-controllers"].version}}
   docker pull {{page.registry}}{{site.imageNames["cloudControllers"]}}:{{site.data.versions[page.version].first.components["cloud-controllers"].version}}
   docker pull {{page.registry}}{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}}
   docker pull {{page.registry}}{{site.imageNames["elastic-tsee-installer"]}}:{{site.data.versions[page.version].first.components["elastic-tsee-installer"].version}}
   docker pull {{page.registry}}{{site.imageNames["intrusion-detection-controller"]}}:{{site.data.versions[page.version].first.components["intrusion-detection-controller"].version}}
   docker pull {{page.registry}}{{site.imageNames["es-curator"]}}:{{site.data.versions[page.version].first.components["es-curator"].version}}
   docker pull {{page.registry}}{{site.imageNames["compliance-controller"]}}:{{site.data.versions[page.version].first.components["compliance-controller"].version}}
   docker pull {{page.registry}}{{site.imageNames["compliance-reporter"]}}:{{site.data.versions[page.version].first.components["compliance-reporter"].version}}
   docker pull {{page.registry}}{{site.imageNames["compliance-snapshotter"]}}:{{site.data.versions[page.version].first.components["compliance-snapshotter"].version}}
   docker pull {{page.registry}}{{site.imageNames["compliance-server"]}}:{{site.data.versions[page.version].first.components["compliance-server"].version}}
   docker pull quay.io/calico/cni:{{site.data.versions[page.version].first.components["calico/cni"].version}}
   docker pull quay.io/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker pull quay.io/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker pull quay.io/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker pull quay.io/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker pull quay.io/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   docker pull upmcenterprises/elasticsearch-operator:{{site.data.versions[page.version].first.components["elasticsearch-operator"].version}}
   docker pull busybox:{{site.data.versions[page.version].first.components["busybox"].version}}
   ```
   {: #load-docker-pull-cmds}

1. Retag the images with the name of your private registry.

   ```bash
   docker tag docker.elastic.co/elasticsearch/elasticsearch:{{site.data.versions[page.version].first.components["elasticsearch"].version}} <YOUR-REGISTRY>/elasticsearch/elasticsearch:{{site.data.versions[page.version].first.components["elasticsearch"].version}}
   docker tag {{page.registry}}{{site.imageNames["kibana"]}}:{{site.data.versions[page.version].first.components["kibana"].version}} <YOUR-REGISTRY>/tigera/kibana:{{site.data.versions[page.version].first.components["kibana"].version}}
   docker tag {{page.registry}}{{site.imageNames["calicoctl"]}}:{{site.data.versions[page.version].first.components["calicoctl"].version}} <YOUR-REGISTRY>/{{site.imageNames["calicoctl"]}}:{{site.data.versions[page.version].first.components["calicoctl"].version}}
   docker tag {{page.registry}}{{site.imageNames["calicoq"]}}:{{site.data.versions[page.version].first.components["calicoq"].version}} <YOUR-REGISTRY>/{{site.imageNames["calicoq"]}}:{{site.data.versions[page.version].first.components["calicoq"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}} <YOUR-REGISTRY>/{{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}} <YOUR-REGISTRY>/{{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}} <YOUR-REGISTRY>/{{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker tag {{page.registry}}{{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}} <YOUR-REGISTRY>/{{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}} <YOUR-REGISTRY>/{{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker tag {{page.registry}}{{site.imageNames["es-proxy"]}}:{{site.data.versions[page.version].first.components["es-proxy"].version}} <YOUR-REGISTRY>/{{site.imageNames["es-proxy"]}}:{{site.data.versions[page.version].first.components["es-proxy"].version}}
   docker tag {{page.registry}}{{site.imageNames["fluentd"]}}:{{site.data.versions[page.version].first.components["fluentd"].version}} <YOUR-REGISTRY>/{{site.imageNames["fluentd"]}}:{{site.data.versions[page.version].first.components["fluentd"].version}}
   docker tag {{page.registry}}{{site.imageNames["kubeControllers"]}}:{{site.data.versions[page.version].first.components["cnx-kube-controllers"].version}} <YOUR-REGISTRY>/{{site.imageNames["kubeControllers"]}}:{{site.data.versions[page.version].first.components["cnx-kube-controllers"].version}}
   docker tag {{page.registry}}{{site.imageNames["cloudControllers"]}}:{{site.data.versions[page.version].first.components["cloud-controllers"].version}} <YOUR-REGISTRY>/{{site.imageNames["cloudControllers"]}}:{{site.data.versions[page.version].first.components["cloud-controllers"].version}}
   docker tag {{page.registry}}{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}} <YOUR-REGISTRY>/{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}}
   docker tag {{page.registry}}{{site.imageNames["elastic-tsee-installer"]}}:{{site.data.versions[page.version].first.components["elastic-tsee-installer"].version}} <YOUR-REGISTRY>/{{site.imageNames["elastic-tsee-installer"]}}:{{site.data.versions[page.version].first.components["elastic-tsee-installer"].version}}
   docker tag {{page.registry}}{{site.imageNames["intrusion-detection-controller"]}}:{{site.data.versions[page.version].first.components["intrusion-detection-controller"].version}} <YOUR-REGISTRY>/{{site.imageNames["intrusion-detection-controller"]}}:{{site.data.versions[page.version].first.components["intrusion-detection-controller"].version}}
   docker tag {{page.registry}}{{site.imageNames["es-curator"]}}:{{site.data.versions[page.version].first.components["es-curator"].version}} <YOUR-REGISTRY>/{{site.imageNames["es-curator"]}}:{{site.data.versions[page.version].first.components["es-curator"].version}}
   docker tag {{page.registry}}{{site.imageNames["compliance-controller"]}}:{{site.data.versions[page.version].first.components["compliance-controller"].version}} <YOUR-REGISTRY>/{{site.imageNames["compliance-controller"]}}:{{site.data.versions[page.version].first.components["compliance-controller"].version}}
   docker tag {{page.registry}}{{site.imageNames["compliance-reporter"]}}:{{site.data.versions[page.version].first.components["compliance-reporter"].version}} <YOUR-REGISTRY>/{{site.imageNames["compliance-reporter"]}}:{{site.data.versions[page.version].first.components["compliance-reporter"].version}}
   docker tag {{page.registry}}{{site.imageNames["compliance-snapshotter"]}}:{{site.data.versions[page.version].first.components["compliance-snapshotter"].version}} <YOUR-REGISTRY>/{{site.imageNames["compliance-snapshotter"]}}:{{site.data.versions[page.version].first.components["compliance-snapshotter"].version}}
   docker tag {{page.registry}}{{site.imageNames["compliance-server"]}}:{{site.data.versions[page.version].first.components["compliance-server"].version}} <YOUR-REGISTRY>/{{site.imageNames["compliance-server"]}}:{{site.data.versions[page.version].first.components["compliance-server"].version}}
   docker tag quay.io/calico/cni:{{site.data.versions[page.version].first.components["calico/cni"].version}} <YOUR-REGISTRY>/calico/cni:{{site.data.versions[page.version].first    .components["calico/cni"].version}}
   docker tag quay.io/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}} <YOUR-REGISTRY>/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker tag quay.io/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}} <YOUR-REGISTRY>/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker tag quay.io/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}} <YOUR-REGISTRY>/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker tag quay.io/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}} <YOUR-REGISTRY>/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker tag quay.io/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}} <YOUR-REGISTRY>/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   docker tag upmcenterprises/elasticsearch-operator:{{site.data.versions[page.version].first.components["elasticsearch-operator"].version}} <YOUR-REGISTRY>/upmcenterprises/elasticsearch-operator:{{site.data.versions[page.version].first.components["elasticsearch-operator"].version}}
   docker tag busybox:{{site.data.versions[page.version].first.components["busybox"].version}} <YOUR-REGISTRY>/busybox:{{site.data.versions[page.version].first.components["busybox"].version}}
   ```
   {: #load-docker-tag-cmds}
   > **Note**: We recommend changing just the name of the registry (`<YOUR-REGISTRY>`)
   > when retagging the images, as shown above and below.
   {: .alert .alert-info}

1. Push the images to your private registry.

   ```bash
   docker push <YOUR-REGISTRY>/elasticsearch/elasticsearch:{{site.data.versions[page.version].first.components["elasticsearch"].version}}
   docker push <YOUR-REGISTRY>/tigera/kibana:{{site.data.versions[page.version].first.components["kibana"].version}}
   docker push <YOUR-REGISTRY>/tigera/calicoctl:{{site.data.versions[page.version].first.components["calicoctl"].version}}
   docker push <YOUR-REGISTRY>/tigera/calicoq:{{site.data.versions[page.version].first.components["calicoq"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-apiserver:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-manager:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-manager-proxy:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-node:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker push <YOUR-REGISTRY>/tigera/cnx-queryserver:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["es-proxy"]}}:{{site.data.versions[page.version].first.components["es-proxy"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["fluentd"]}}:{{site.data.versions[page.version].first.components["fluentd"].version}}
   docker push <YOUR-REGISTRY>/tigera/kube-controllers:{{site.data.versions[page.version].first.components["cnx-kube-controllers"].version}}
   docker push <YOUR-REGISTRY>/tigera/cloud-controllers:{{site.data.versions[page.version].first.components["cloud-controllers"].version}}
   docker push <YOUR-REGISTRY>/tigera/typha:{{site.data.versions[page.version].first.components["typha"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["elastic-tsee-installer"]}}:{{site.data.versions[page.version].first.components["elastic-tsee-installer"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["intrusion-detection-controller"]}}:{{site.data.versions[page.version].first.components["intrusion-detection-controller"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["es-curator"]}}:{{site.data.versions[page.version].first.components["es-curator"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["compliance-controller"]}}:{{site.data.versions[page.version].first.components["compliance-controller"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["compliance-reporter"]}}:{{site.data.versions[page.version].first.components["compliance-reporter"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["compliance-snapshotter"]}}:{{site.data.versions[page.version].first.components["compliance-snapshotter"].version}}
   docker push <YOUR-REGISTRY>/{{site.imageNames["compliance-server"]}}:{{site.data.versions[page.version].first.components["compliance-server"].version}}
   docker push <YOUR-REGISTRY>/calico/cni:{{site.data.versions[page.version].first.components["calico/cni"].version}}
   docker push <YOUR-REGISTRY>/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker push <YOUR-REGISTRY>/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker push <YOUR-REGISTRY>/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker push <YOUR-REGISTRY>/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker push <YOUR-REGISTRY>/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   docker push <YOUR-REGISTRY>/upmcenterprises/elasticsearch-operator:{{site.data.versions[page.version].first.components["elasticsearch-operator"].version}}
   docker push <YOUR-REGISTRY>/busybox:{{site.data.versions[page.version].first.components["busybox"].version}}
   ```
   {: #load-docker-push-cmds}

   > **Important**: Do not push the private {{site.tseeprodname}} images to a public registry.
   {: .alert .alert-danger}

{% else %}

1. Use the following commands to pull the `{{include.yaml}}` image from the Tigera
   registry.

   ```bash
   docker pull {{page.registry}}{{site.imageNames[include.yaml]}}:{{site.data.versions[page.version].first.components[include.yaml].version}}
   ```
   {: #load-docker-pull-cmds}

1. Retag the image with the name of your private registry.

   ```bash
   docker tag {{site.imageNames[include.yaml]}}:{{site.data.versions[page.version].first.components[include.yaml].version}} <YOUR-REGISTRY>/tigera/{{include.yaml}}:{{site.data.versions[page.version].first.components[include.yaml].version}}
   ```
   {: #load-docker-tag-cmds}
   > **Note**: We recommend changing just the name of the registry (`<YOUR-REGISTRY>`),
   > as shown above. This will make it easier to complete the instructions that follow.
   {: .alert .alert-info}

1. Push the images to your private registry.

   ```bash
   docker push <YOUR-REGISTRY>/tigera/{{include.yaml}}:{{site.data.versions[page.version].first.components[include.yaml].version}}
   ```
   {: #load-docker-push-cmds}

{% endif %}

{% unless include.upgrade %}

{% if include.orchestrator == "openshift" %}

1. Modify the Ansible inventory file to include [the location of the registry](https://docs.openshift.com/container-platform/3.11/install/configuring_inventory_file.html#advanced-install-configuring-registry-location) which
   contains the private images and [the credentials needed to authenticate to it](https://github.com/openshift/openshift-ansible/blob/master/inventory/hosts.example#L223).

{% endif %}

{% if include.yaml != "calico" %}

1. Use the YAML that matches your datastore type to download the `{{include.yaml}}` manifest.

   - **etcd**

     ```
     curl -O \
     {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/{{include.yaml}}.yaml
     ```

     > **Note**: You can also
     > [view the YAML in a new tab]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/{{include.yaml}}.yaml){:target="_blank"}.
     {: .alert .alert-info}

   - **Kubernetes API datastore**

     ```
     curl -O \
     {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/{{include.yaml}}.yaml
     ```

     > **Note**: You can also
     > [view the YAML in a new tab]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/{{include.yaml}}.yaml){:target="_blank"}.
     {: .alert .alert-info}

{% include {{page.version}}/cnx-cred-sed.md yaml=include.yaml %}

{% if include.yaml == "calicoq" %}
1. The manifest will need to be modified if you are using {{site.tseeprodname}} federation and need to mount in secrets to
   access the remote cluster datastores. Follow the instructions in the manifest to enable this feature.
{% endif %}

1. Apply the YAML file.

   ```bash
   kubectl apply -f {{include.yaml}}.yaml
   ```
{% if include.yaml == "calicoctl" or include.yaml == "calicoq" %}
1. Create an alias.

   ```bash
   alias {{include.yaml}}="kubectl exec -i -n kube-system {{include.yaml}} /{{include.yaml}} -- "
   ```

   **Note**: In order to use the `{{include.yaml}}` alias
   when reading manifests, redirect the file into stdin, for example:
   ```
   {{include.yaml}} create -f - < my_manifest.yaml
   ```
   {: .alert .alert-success}
{% endif %}

{% endif %}
{% endunless %}
