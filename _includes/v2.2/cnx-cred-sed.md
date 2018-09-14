1. If you are pulling from the Tigera private registry, skip to the next step. If you
   are not pulling from the Tigera private registry, use the following commands to set an
   environment  variable called `REGISTRY` containing the location of your private registry
   and replace `{{site.data.versions[page.version].first.dockerRepo}}` in the manifest with the location of your private registry.

   ```bash
   REGISTRY=my-registry.com \
   sed -i -e "s?{{site.data.versions[page.version].first.dockerRepo}}?$REGISTRY?g" {{include.yaml}}.yaml
   ```

   > **Tip**: If you're hosting your own private registry, you may need to include
   > a port number. For example, `my-registry.com:5000`.
   {: .alert .alert-success}
