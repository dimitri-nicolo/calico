---
title: Custom BGP Configuration
canonical_url: https://docs.tigera.io/v2.3/usage/custom-bgp-config
---

### Big picture

Use customized BIRD configuration files to enable specialized use-cases.

### Concepts

In {{site.prodname}}, BGP is handled by [BIRD](https://github.com/projectcalico/bird).
The BIRD configurations are templated out through [confd](https://github.com/projectcalico/confd).
You can modify the BIRD configuration to use BIRD features which are not typically exposed using the
default configuration provided with {{site.prodname}}.

Customization of BGP templates should be done only with the help of your Tigera Support representative.

### How to

- [Update BGP configuration](#update-bgp-configuration)
- Appy BGP customizations based on how you've deployed {{site.prodname}}:
  - [Tigera operator](#tigera-operator)
  - [Manual deployment](#manual-deployment)

#### Update BGP configuration

Using the directions provided with the templates, set the correct values 
for the BGP configuration using these resources:

- [BGP Configuration]({{site.url}}/{{page.version}}/reference/resources/bgpconfig)
- [BGP Peer]({{site.url}}/{{page.version}}/reference/resources/bgppeer)
- [calicoctl]({{site.url}}/{{page.version}}/reference/calicoctl)

#### Tigera operator

1. Create your confd templates.
1. Create a ConfigMap from the templates.

  ```
  kubectl create configmap bird-templates -n tigera-operator --from-file=<path to directory of templates>
  ``` 

  The created config map will be used to populate the {{site.prodname}} BIRD configuration file templates. If a template with the same name already exists within the node container, it will be overwritten with the contents from the config map.

#### Manual deployment

1. Create your confd templates.
1. Apply these templates to your {{site.nodecontainer}} instances using a ConfigMap. 
   For help, see [Kubernetes ConfigMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/).

  ```
  kubectl create configmap bgp-templates -n kube-system --from-file=<path to directory of templates> 
  ```

1. To make the templates available to the {{site.prodname}} node containers, edit your `calico.yaml`
   file to add the following for the `{{site.nodecontainer}}` container:

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

1. Reapply the updated manifest with the following command.

   ```
   kubectl apply -f calico.yaml
   ```

   After the BIRD templates are overwritten, your BGP configuration changes are active!
