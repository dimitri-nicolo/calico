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
   docker pull docker.elastic.co/elasticsearch/elasticsearch-oss:{{site.data.versions[page.version].first.components["elasticsearch"].version}}
   docker pull docker.elastic.co/kibana/kibana-oss:{{site.data.versions[page.version].first.components["kibana"].version}}
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
   docker pull {{page.registry}}{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}}
   docker pull quay.io/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker pull quay.io/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker pull quay.io/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker pull quay.io/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker pull quay.io/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   docker pull giantswarm/tiny-tools:{{site.data.versions[page.version].first.components["tiny-tools"].version}}
   docker pull upmcenterprises/elasticsearch-operator:{{site.data.versions[page.version].first.components["elasticsearch-operator"].version}}
   ```

1. Retag the images with the name of your private registry.

   ```bash
   docker tag docker.elastic.co/elasticsearch/elasticsearch-oss:{{site.data.versions[page.version].first.components["elasticsearch"].version}} <YOUR-REGISTRY>/docker.elastic.co/elasticsearch/elasticsearch-oss:{{site.data.versions[page.version].first.components["elasticsearch"].version}}
   docker tag docker.elastic.co/kibana/kibana-oss:{{site.data.versions[page.version].first.components["kibana"].version}} <YOUR-REGISTRY>/docker.elastic.co/kibana/kibana-oss:{{site.data.versions[page.version].first.components["kibana"].version}}
   docker tag {{page.registry}}{{site.imageNames["calicoctl"]}}:{{site.data.versions[page.version].first.components["calicoctl"].version}} <YOUR-REGISTRY>/{{site.imageNames["calicoctl"]}}:{{site.data.versions[page.version].first.components["calicoctl"].version}}
   docker tag {{page.registry}}{{site.imageNames["calicoq"]}}:{{site.data.versions[page.version].first.components["calicoq"].version}} <YOUR-REGISTRY>/{{site.imageNames["calicoq"]}}:{{site.data.versions[page.version].first.components["calicoq"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}} <YOUR-REGISTRY>/{{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}} <YOUR-REGISTRY>/{{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}} <YOUR-REGISTRY>/{{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker tag {{page.registry}}{{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}} <YOUR-REGISTRY>/{{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker tag {{page.registry}}{{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}} <YOUR-REGISTRY>/{{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker tag {{page.registry}}{{site.imageNames["es-proxy"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}} <YOUR-REGISTRY>/{{site.imageNames["es-proxy"]}}:{{site.data.versions[page.version].first.components["es-proxy"].version}}
   docker tag {{page.registry}}{{site.imageNames["fluentd"]}}:{{site.data.versions[page.version].first.components["fluentd"].version}} <YOUR-REGISTRY>/{{page.registry}}{{site.imageNames["fluentd"]}}:{{site.data.versions[page.version].first.components["fluentd"].version}}
   docker tag {{page.registry}}{{site.imageNames["kubeControllers"]}}:{{site.data.versions[page.version].first.components["cnx-kube-controllers"].version}} <YOUR-REGISTRY>/{{site.imageNames["kubeControllers"]}}:{{site.data.versions[page.version].first.components["cnx-kube-controllers"].version}}
   docker tag {{page.registry}}{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}} <YOUR-REGISTRY>/{{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}}
   docker tag quay.io/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}} <YOUR-REGISTRY>/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker tag quay.io/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}} <YOUR-REGISTRY>/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker tag quay.io/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}} <YOUR-REGISTRY>/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker tag quay.io/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}} <YOUR-REGISTRY>/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker tag quay.io/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}} <YOUR-REGISTRY>/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   docker tag giantswarm/tiny-tools:{{site.data.versions[page.version].first.components["tiny-tools"].version}} <YOUR-REGISTRY>/giantswarm/tiny-tools:{{site.data.versions[page.version].first.components["tiny-tools"].version}}
   docker tag upmcenterprises/elasticsearch-operator:{{site.data.versions[page.version].first.components["elasticsearch-operator"].version}} <YOUR-REGISTRY>/upmcenterprises/elasticsearch-operator:{{site.data.versions[page.version].first.components["elasticsearch-operator"].version}}
   ```
   > **Note**: We recommend changing just the name of the registry (`<YOUR-REGISTRY>`)
   > when retagging the images, as shown above and below.
   {: .alert .alert-info}

1. Push the images to your private registry.

   ```bash
   docker push <YOUR-REGISTRY>/docker.elastic.co/elasticsearch/elasticsearch-oss:{{site.data.versions[page.version].first.components["elasticsearch"].version}}
   docker push <YOUR-REGISTRY>/docker.elastic.co/kibana/kibana-oss:{{site.data.versions[page.version].first.components["kibana"].version}}
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
   docker push <YOUR-REGISTRY>/tigera/typha:{{site.data.versions[page.version].first.components["typha"].version}}
   docker push <YOUR-REGISTRY>/coreos/configmap-reload:{{site.data.versions[page.version].first.components["configmap-reload"].version}}
   docker push <YOUR-REGISTRY>/coreos/prometheus-config-reloader:{{site.data.versions[page.version].first.components["prometheus-config-reloader"].version}}
   docker push <YOUR-REGISTRY>/coreos/prometheus-operator:{{site.data.versions[page.version].first.components["prometheus-operator"].version}}
   docker push <YOUR-REGISTRY>/prometheus/alertmanager:{{site.data.versions[page.version].first.components["alertmanager"].version}}
   docker push <YOUR-REGISTRY>/prometheus/prometheus:{{site.data.versions[page.version].first.components["prometheus"].version}}
   docker push <YOUR-REGISTRY>/giantswarm/tiny-tools:{{site.data.versions[page.version].first.components["tiny-tools"].version}}
   docker push <YOUR-REGISTRY>/upmcenterprises/elasticsearch-operator:{{site.data.versions[page.version].first.components["elasticsearch-operator"].version}}
   ```

   > **Important**: Do not push the private {{site.tseeprodname}} images to a public registry.
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

1. Issue the following command to create the `calico-monitoring` namespace.

   ```bash
   kubectl create namespace calico-monitoring
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
   kubectl create -f cnx-pull-secret.yml
   ```

   It should return the following.

   ```bash
   secret "cnx-pull-secret" created
   secret "cnx-pull-secret" created
   ```

1. Continue to [Installing {{site.tseeprodname}}](#install-cnx).

{% endif %}

{% if include.orchestrator == "openshift" %}

1. Modify the Ansible inventory file to include [the location of the registry](https://docs.openshift.com/container-platform/3.10/install/configuring_inventory_file.html#advanced-install-configuring-registry-location) which
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
     > [view the YAML in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/{{include.yaml}}.yaml){:target="_blank"}.
     {: .alert .alert-info}

   - **Kubernetes API datastore**

     ```
     curl -O \
     {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/{{include.yaml}}.yaml
     ```

     > **Note**: You can also
     > [view the YAML in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/{{include.yaml}}.yaml){:target="_blank"}.
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
{% endif %}

{% endif %}
