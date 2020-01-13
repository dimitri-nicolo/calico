1. Log into the host, open a terminal prompt, and navigate to the location where
you want to install the binary.

   > **Tip**: Navigate to a location that's in your `PATH`. For example,
   > `/usr/local/bin/`.
   {: .alert .alert-success}

1. Ensure that you have the [`config.json` file with the private Tigera registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials).

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
   docker pull {{page.registry}}{{site.imageNames[include.cli]}}:{{site.data.versions[page.version].first.components[include.cli].version}}
   ```

1. Confirm that the image has loaded by typing `docker images`.

   ```bash
   REPOSITORY                TAG               IMAGE ID       CREATED         SIZE
   {{site.imageNames[include.cli]}}  {{site.data.versions[page.version].first.components[include.cli].version}}            e07d59b0eb8a   2 minutes ago   42MB
   ```

1. Create a copy of the container.

   ```bash
   docker create --name {{include.cli}}-copy {{page.registry}}{{site.imageNames[include.cli]}}:{{site.data.versions[page.version].first.components[include.cli].version}}
   ```

1. Copy the {{include.cli}} file from the container to the local file system. The following command copies it to a common `$PATH` location.

   ```bash
   docker cp {{include.cli}}-copy:{{include.codepath}} {{include.cli}}
   ```

1. Use the following command to delete the copy of the {{include.cli}} container.

   ```bash
   docker rm {{include.cli}}-copy
   ```

1. Set the file to be executable.

   ```
   chmod +x {{include.cli}}
   ```
