## Install {{include.cli}} as a kubectl plugin on a single host

1. Log in to the host, open a terminal prompt, and navigate to the location where
you want to install the binary.

   > **Note**: In order to install `{{include.cli}}` as a kubectl plugin, the binary must be located in your `PATH`. For example,
   > `/usr/local/bin/`.
   {: .alert .alert-info}

1. Ensure that you have the [`config.json` file with the private Tigera registry credentials]({{site.baseurl}}/getting-started/calico-enterprise#get-private-registry-credentials-and-license-key).

1. From a terminal prompt, use the following command to either create or open the `~/.docker/config.json` file.

   ```bash
   vi ~/.docker/config.json
   ```

1. Depending on the existing contents of the file, edit it in one of the following ways.

   - **New file**: If you have created a new file, paste in the entire contents of the
   `config.json` file from Tigera.

   - **Existing file without quay.io object**: If you have opened an existing file that does not contain an existing `"quay.io"` object, add the following lines from the `config.json` inside the `"auth"` object.

     ```json
     "quay.io": {
       "auth": "<ROBOT-TOKEN-VALUE>",
       "email": ""
     }
     ```

   - **Existing file with quay.io object**: If you have opened an existing file that already contains a `"quay.io"` entry, add the following lines from the `config.json` inside the `"quay.io"` object.

     ```json
     "auth": "<ROBOT-TOKEN-VALUE>",
     "email": ""
     ```

1. Save and close the file.

1. Use the following commands to pull the {{include.cli}} image from the Tigera
   registry.

   ```bash
   docker pull {{page.registry}}{% include component_image component=include.cli %}
   ```

1. Confirm that the image has loaded by typing `docker images`.
{%- assign c = site.data.versions.first.components[include.cli] %}

   ```bash
   REPOSITORY                TAG               IMAGE ID       CREATED         SIZE
   {{ c.image }}  {{ c.version }}            e07d59b0eb8a   2 minutes ago   42MB
   ```
   {: .no-select-button}

1. Create a copy of the container.

   ```bash
   docker create --name {{include.cli}}-copy {{page.registry}}{% include component_image component=include.cli %}
   ```

1. Copy the {{include.cli}} file from the container to the local file system, while naming it `kubectl-calico`. The name follows kubectl plugin naming convention and is required.

   ```bash
   docker cp {{include.cli}}-copy:{{include.codepath}} kubectl-calico
   ```

1. Use the following command to delete the copy of the {{include.cli}} container.

   ```bash
   docker rm {{include.cli}}-copy
   ```

1. Set the file to be executable.

   ```bash
   chmod +x kubectl-calico
   ```

   > **Note**: If the location of `kubectl-calico` is not already in your `PATH`, move the file
   > to one that is or add its location to your `PATH`. This is required in order for
   > kubectl to detect the plugin and allow you to use it.
   {: .alert .alert-info}

1. Verify the plugin works.

   ```
   kubectl calico -h
   ```

You can now run any `{{include.cli}}` subcommands through `kubectl calico`.

> **Note**: If you run these commands from your local machine (instead of a host node), some of
> the node related subcommands will not work (like node status).
{: .alert .alert-info}

**Next step**:

[Configure `calicoctl` to connect to your datastore](configure).
