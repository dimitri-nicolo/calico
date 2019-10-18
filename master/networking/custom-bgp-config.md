---
title: Custom BGP Configuration
canonical_url: https://docs.tigera.io/v2.3/usage/custom-bgp-config
---

In {{site.prodname}}, BGP is handled by [BIRD](https://github.com/projectcalico/bird).
The BIRD configurations are templated out through [confd](https://github.com/kelseyhightower/confd).
In some cases, you may want to modify the BIRD configuration in order to achieve
features such as:

* Dual top of rack (ToR) peering
* Password-protected BGP peering

This document discusses how to overwrite the default BIRD BGP configuration
and use customized BGP configurations for particular use cases.

### confd templates

In order to overwrite the default BIRD confd templates, we first need replacement
confd templates which are customized as desired. At this time, we only support a
specific collection of BIRD templates. For more information on the functionality
provided by these templates, please contact your Tigera support representative.

### Overwriting confd templates

Once you have a set of confd templates, you will need to apply these templates to
your {{site.nodecontainer}} instances. We can mount in our templates using a
[Kubernetes ConfigMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/).
This will allow us to mount in all our template files to each node in our cluster without
needing to explicitly upload our templates to each host individually.

Create the `ConfigMap`:

```
kubectl create configmap bgp-templates -n kube-system --from-file=<path to directory of templates>
```

Once the configmap has been created, we need to create the appropriate volume mounts to make
the templates available in the {{site.prodname}} node containers.

Edit your `calico.yaml` file to add in the following for the `{{site.nodecontainer}}` container:

```
...

volumeMounts:

...

  - mountPath: /etc/calico/confd/templates
    name: bird-templates
...
volumes:

...

  - name: bird-templates
    configMap:
      name: bgp-templates
```

Your `calico.yaml` file should resemble the following:

```
kind: DaemonSet
apiVersion: networking.k8s.io/v1
metadata:
  name: calico-node
  namespace: kube-system
  labels:
    k8s-app: calico-node
spec:

...

          volumeMounts:
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
            - mountPath: /var/lib/calico
              name: var-lib-calico
              readOnly: false
            - mountPath: /etc/calico/confd/templates
              name: bird-templates

...

      volumes:
        # Used by calico/node.
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        - name: var-lib-calico
          hostPath:
            path: /var/lib/calico
        # Used to install CNI.
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
        - name: bird-templates
          configMap:
            name: bgp-templates

...
```

Now you will need to reapply the updated manifest with the following command.

```
kubectl apply -f calico.yaml
```

### Providing templated values

It is also important to note that you may need to provide specific values to be
rendered as a part of your BGP configuration by the BIRD confd templates. These
values can be set in the
[BGP Configuration]({{site.url}}/{{page.version}}/reference/resources/bgpconfig)
and [BGP Peer]({{site.url}}/{{page.version}}/reference/resources/bgppeer)
objects through [calicoctl]({{site.url}}/{{page.version}}/reference/calicoctl).
Follow the directions provided with the templates you have received in order to ensure
that all of the correct values are set for your BIRD configuration to render appropriately.

### Applying your changes

Once you have completed overwriting your BIRD templates and provided the template values,
you will need to reapply the updated manifest with the following command.

```
kubectl apply -f calico.yaml
```

Your BGP configuration changes should now be active!
