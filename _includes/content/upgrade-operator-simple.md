
1. Download the new operator manifest.
   ```bash
   curl -L -O {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. If you previously [installed using a private registry]({{site.baseurl}}/getting-started/private-registry), you will need to
   [push the new images]({{site.baseurl}}/getting-started/private-registry/private-registry-regular#push-calico-enterprise-images-to-your-private-registry)
   and then [update the manifest]({{site.baseurl}}/getting-started/private-registry/private-registry-regular#run-the-operator-using-images-from-your-private-registry)
   downloaded in the previous step.

1. Apply the Tigera operator.
   ```bash
   kubectl apply -f tigera-operator.yaml
   ```

1. Install the new network policies to secure {{site.prodname}} component communications.
   ```bash
   kubectl apply -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
   ```

1. You can monitor progress with the following command:
   ```bash
   watch kubectl get tigerastatus
   ```

   **Note**: If there are any problems you can use `kubectl get tigerastatus -o yaml` to get more details.
   {: .alert .alert-info}

