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

1. Use the following commands to pull the required {{site.prodname}} images.

   ```bash
   {% for component in site.data.versions[page.version].first.components -%}
   {% if component[1].image and component[0] != "flannel" -%}
   {% if component[1].registry %}{% assign registry = component[1].registry | append: "/" %}{% else %}{% assign registry = page.registry -%}{% endif -%}
   docker pull {{ registry }}{{ component[1].image }}:{{component[1].version}}
   {% endif -%}
   {% endfor -%}
   ```
   {: #load-docker-pull-cmds}

1. Retag the images with the name of your private registry.

   ```bash
   {% for component in site.data.versions[page.version].first.components -%}
   {% if component[1].image and component[0] != "flannel" -%}
   {% if component[1].registry %}{% assign registry = component[1].registry | append: "/" %}{% else %}{% assign registry = page.registry -%}{% endif -%}
   docker tag {{ registry }}{{ component[1].image }}:{{component[1].version}} <YOUR-REGISTRY>/{{ component[1].image }}:{{component[1].version}}
   {% endif -%}
   {% endfor -%}
   ```
   {: #load-docker-tag-cmds}
   > **Note**: We recommend changing just the name of the registry (`<YOUR-REGISTRY>`)
   > when retagging the images, as shown above and below.
   {: .alert .alert-info}

1. Push the images to your private registry.

   ```bash
   {% for component in site.data.versions[page.version].first.components -%}
   {% if component[1].image and component[0] != "flannel" -%}
   docker push <YOUR-REGISTRY>/{{ component[1].image }}:{{component[1].version}}
   {% endif -%}
   {% endfor -%}
   ```
   {: #load-docker-push-cmds}

   > **Important**: Do not push the private {{site.prodname}} images to a public registry.
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
1. The manifest will need to be modified if you are using {{site.prodname}} federation and need to mount in secrets to
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
