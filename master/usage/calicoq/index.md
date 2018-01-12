---
title: Installing calicoq
---

This document outlines how to install `calicoq`.

### Where to run calicoq

You can run `calicoq` from any host with network access to the
datastore.

### Installing calicoq

1. Ensure that you have [obtained the calicoq image](../../getting-started/#obtain-the-private-binaries-and-images).

1. Import the files into the local Docker engine.

   ```
   docker load -i tigera_calicoq_{{site.data.versions[page.version].first.components["calicoq"].version}}.tar.xz.tar.xz
   ```

1. You can then extract the binary from the image or run `calicoq` in a container.

> **Note**: Move `calicoq` to a directory in your `PATH` or add the directory it is in to
> your `PATH` to avoid having to prepend the path to invocations of `calicoq`.
{: .alert .alert-info}
