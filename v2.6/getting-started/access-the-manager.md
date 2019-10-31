---
title: Access the manager UI
---

### Big picture

Configure access to the {{site.prodname}} manager user interface.

### Value

By default, the manager UI is not exposed outside of the cluster. This article explains approaches for allowing access to the UI.

### Before you begin...

Make sure you have installed {{site.prodname}} using one of the [installation guides](/{{page.version}}/getting-started/).

### How to

Choose one of the following methods for accessing the manager UI:

- [Access using Kubernetes Ingress](#access-using-kubernetes-ingress)
- [Access using a LoadBalancer service](#access-using-a-loadbalancer-service)
- [Access using port-forwarding](#access-using-port-forwarding)

#### Access using Kubernetes Ingress

Kubernetes services can be exposed outside of the cluster using [the Kubernetes Ingress API](https://kubernetes.io/docs/concepts/services-networking/ingress/). This approach requires that your cluster be configured with an ingress controller to implement the `Ingress` resource.

To expose the manager using a Kubernetes ingress, you can create an `Ingress` resource like the one below.

```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: tigera-manager
  namespace: tigera-manager
spec:
  backend:
    serviceName: tigera-manager
    servicePort: 9443
```

> Note: You must ensure the {{site.prodname}} manager receives a HTTPS (TLS) connection, not unencrypted HTTP. If you require TLS termination at your ingress, you will need to either use a proxy that supports transparent HTTP/2 proxying, for example, Envoy, or re-originate a TLS connection from your proxy to the {{site.prodname}} manager. If you do not require TLS termination, configure your proxy to “pass thru” the TLS to {{site.prodname}} manager.

#### Access using a LoadBalancer service

Kubernetes services can be exposed outside of the cluster [by configuring type `LoadBalancer`](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/) in the service specification. This requires that your cluster be configured with a service load balancer controller to implement the external load balancer. Most managed Kubernetes platforms support this.

To expose the manager using a load balancer, create the following service.

```yaml
kind: Service
apiVersion: v1
metadata:
  name: tigera-manager-external
  namespace: tigera-manager
spec:
  type: LoadBalancer
  selector:
    k8s-app: tigera-manager
  externalTrafficPolicy: Local
  ports:
  - port: 9443
    targetPort: 9443
    protocol: TCP
```

> Note: You must ensure the {{site.prodname}} manager receives a HTTPS (TLS) connection, not unencrypted HTTP. If you require TLS termination at your load balancer, you will need to either use a load balancer that supports transparent HTTP/2 proxying, or re-originate a TLS connection from your load balancer to the {{site.prodname}} manager. If you do not require TLS termination, configure your proxy to “pass thru” the TLS to {{site.prodname}} manager.

After creating the service, it may take a few minutes for the load balancer to be created. Once complete, the load balancer IP address will appear as an `ExternalIP` in `kubectl get services -n tigera-manager tigera-manager-external`.

#### Access using port-forwarding

You can use `kubectl` to forward a local port to the Kubernetes API server, where it will be proxied to the manager UI. This approach is not recommended for production, but is useful for scenarios where you do not have a load balancer or ingress infrastructure configured, or if you are looking to get started quickly.

To forward traffic locally, use the following command:

```
kubectl port-forward -n tigera-manager service/tigera-manager 9443:9443
```

You can now access the manager UI in your browser at `https://localhost:9443`.

### Above and beyond

- [Create a user and login to the manager UI]({{site.url}}/{{page.version}}/getting-started/create-user-login)
