---
title: CNX Manager with non-admin user
---

{{site.prodname}} Manager is working with both core and {{site.prodname}} apiserver registered Kubernetes resources. In Kubernetes, when a new user is created, the user does not come with any RBAC permissions. Quickstart way of going about accessing the UI has been to bind the user with the `cluster-admin` role.

But, if you will like to set up a more controlled workflow for a new user, where-in, say, the user does not have access to CRUDing of networpolicies resources outside of the `default` namespace and outside of the `default` tier you may consider the following instructions:

1. Download the [`min-rbac.yaml` manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/demo-manifests/min-rbac.yaml). 

1. Run the following command to replace the `<USER>` with the `name/email` of the non-admin user you are providing permissions to. 
   ```
   sed -i -e 's/<USER>/<name/email>/g' min-rbac.yaml
   ```

1. Use the following command to install the bindings.
   ```
   kubectl apply -f min-rbac.yaml
   ```

Depending upon how you/admin would like to open up the system resources to the respective user, you may now add to the set of roles and bindings.

**[Here's]({{site.baseurl}}/{{page.version}}/reference/essentials/rbac-tiered-policies)** an overview on how RBAC with tiered policies work in {{site.prodname}}.