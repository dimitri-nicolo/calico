1. Ensure that you have [obtained the calicoctl image](/getting-started/#obtain-the-private-binaries).

1. Import the files into the local Docker engine. 

   ```
   docker load -i tigera_calicoctl_{{site.data.versions[page.version].first.components["tigera-calicoctl"].version}}.tar.xz.tar.xz
   ```
   
1. You can then extract the binary from the image or run `calicoctl` in a container.
