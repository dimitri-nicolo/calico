1. Use the following commands to set an environment variable called `REGISTRY` containing
   the location of your private registry and replace the paths in the manifest to refer to it.

   ```bash
   REGISTRY=my-registry.com \
   sed -i -e "s?{{page.registry}}?$REGISTRY?g" {{include.yaml}}.yaml {% if include.yaml == "monitor-calico" %}\
   sed -i -e "s?docker.elastic.co?$REGISTRY/upmcenterprises?$REGISTRY?g" {{include.yaml}}.yaml{% endif %}
   ```

   > **Tip**: If you're hosting your own private registry, you may need to include
   > a port number. For example, `my-registry.com:5000`.
   {: .alert .alert-success}
