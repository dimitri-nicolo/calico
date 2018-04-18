1. If you are pulling from a quay.io registry, skip to the next step. If you 
   are not pulling from quay.io, use the following command to replace `quay.io` 
   in the manifest with the name of your private registry. 

   **Command**
   ```shell
   sed -i -e 's/quay.io/<REPLACE_ME>/g' {{include.yaml}}.yaml
   ```
   
   **Example**
   ```shell
   sed -i -e 's/quay.io/my-registry.com/g' {{include.yaml}}.yaml
   ```
   
   > **Tip**: If you're hosting your own private registry, you may need to include
   > a port number. For example, `my-registry.com:5000`.
   {: .alert .alert-success}