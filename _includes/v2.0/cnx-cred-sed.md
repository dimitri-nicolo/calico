
1. Use the following `sed` command to quickly replace `<YOUR_PRIVATE_DOCKER_REGISTRY>`
   in the manifest with the name of your private registry. Since the manifest 
   already contains the names of the images and their version numbers, you
   just need to replace `<REPLACE_ME>` with the name of the private
   repository before issuing the command.

   **Command**
   ```shell
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/<REPLACE_ME>/g' calico.yaml
   ```
   
   **Example**

   ```shell
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/my-repo/g' calico.yaml
   ```
   
   > **Tip**: If you're hosting your own private repository, you may need to include
   > a port number. For example, `my-repo:5000`.
   {: .alert .alert-success}