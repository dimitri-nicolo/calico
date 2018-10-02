1. If you are pulling from Tigera's private registry, skip to the next step. If you
   are pulling from another private registry, use the following commands to: set an environment
   variable called `REGISTRY` containing the location of the private registry and
   replace `{{site.data.versions[page.version].first.dockerRepo}}` in the manifest with the location of
   the private registry.

   ```bash
   REGISTRY=my-registry.com \
   sed -i -e "s?{{site.data.versions[page.version].first.dockerRepo}}?$REGISTRY?g" {{include.yaml}}.yaml
   ```

   > **Tip**: If you're hosting your own private registry, you may need to include
   > a port number. For example, `my-registry.com:5000`.
   {: .alert .alert-success}
