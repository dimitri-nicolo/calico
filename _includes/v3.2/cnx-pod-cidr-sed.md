1. If you are using pod CIDR `192.168.0.0/16`, skip to the next step. If you
   are using a different pod CIDR, use the following commands to: set an environment
   variable called `POD_CIDR` containing your pod CIDR and
   replace `192.168.0.0/16` in the manifest with your pod CIDR.

   ```bash
   POD_CIDR="{your pod CIDR}" \
   sed -i -e "s?192.168.0.0/16?$POD_CIDR?g" {{include.yaml}}.yaml
   ```
