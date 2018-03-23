{% if include.yaml == "calico" %}

## Setting up access to the private {{site.prodname}} images

### About setting up access to the private {{site.prodname}} images

{{site.prodname}} includes several private images. You can set up access to these
in either of the following ways.

- [Pull the images from Tigera's private registry](#pulling-the-images-from-tigeras-private-registry).

- [Pull the images from another private registry](#pulling-the-images-from-another-private-registry). (Advanced)

{% endif %}

### Pulling the images from Tigera's private registry

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

[Configure `{{include.yaml}}` to connect to your datastore](/{{page.version}}/usage/include.yaml/configure/).

{% endif %}

### Pulling the image from another private registry


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
   docker pull {{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker pull {{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker pull {{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker pull {{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker pull {{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker pull {{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}}
   docker pull quay.io/coreos/prometheus-operator:v0.15.0
   docker pull quay.io/prometheus/alertmanager:v0.11.0
   docker pull quay.io/prometheus/prometheus:v2.0.0
   docker pull quay.io/coreos/prometheus-config-reloader:v0.0.2
   docker pull quay.io/coreos/configmap-reload:v0.0.1
   ```

1. Retag the images with the name of your private registry.

   ```bash
   docker tag {{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}} <YOUR-REGISTRY>/tigera/cnx-apiserver:{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}
   docker tag {{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}} <YOUR-REGISTRY>/tigera/cnx-queryserver:{{site.data.versions[page.version].first.components["cnx-queryserver"].version}}
   docker tag {{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.components["cnx-manager"].version}} <YOUR-REGISTRY>/tigera/cnx-manager:{{site.data.versions[page.version].first.components["cnx-manager"].version}}
   docker tag {{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}} <YOUR-REGISTRY>/tigera/cnx-manager-proxy:{{site.data.versions[page.version].first.components["cnx-manager-proxy"].version}}
   docker tag {{site.imageNames["node"]}}:{{site.data.versions[page.version].first.components["cnx-node"].version}} <YOUR-REGISTRY>/tigera/cnx-node:{{site.data.versions[page.version].first.components["cnx-node"].version}}
   docker tag {{site.imageNames["typha"]}}:{{site.data.versions[page.version].first.components["typha"].version}} <YOUR-REGISTRY>/tigera/typha:{{site.data.versions[page.version].first.components["typha"].version}}
   docker tag quay.io/coreos/prometheus-operator:v0.15.0 <YOUR-REGISTRY>/prometheus-operator:v0.15.0
   docker tag quay.io/prometheus/alertmanager:v0.11.0 <YOUR-REGISTRY>/alertmanager:v0.11.0
   docker tag quay.io/prometheus/prometheus:v2.0.0 <YOUR-REGISTRY>/prometheus:v2.0.0
   docker tag quay.io/coreos/prometheus-config-reloader:v0.0.2 <YOUR-REGISTRY>/prometheus-config-reloader:v0.0.2
   docker tag quay.io/coreos/configmap-reload:v0.0.1 <YOUR-REGISTRY>/configmap-reload:v0.0.1
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
   docker push <YOUR-REGISTRY>/prometheus-operator:v0.15.0
   docker push <YOUR-REGISTRY>/alertmanager:v0.11.0
   docker push <YOUR-REGISTRY>/prometheus:v2.0.0
   docker push <YOUR-REGISTRY>/prometheus-config-reloader:v0.0.2
   docker push <YOUR-REGISTRY>/configmap-reload:v0.0.1
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

1. Push the credentials of your private repository up to Kubernetes as a [secret](https://kubernetes.io/docs/concepts/containers/images/#creating-a-secret-with-a-docker-config)
named `cnx-pull-secret` in the `kube-system` namespace.

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

