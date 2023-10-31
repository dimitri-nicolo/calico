### Before you begin…

-   Configure pull access to your private registry
-   [Configure pull access to Tigera’s private container
    registry](https://docs.tigera.io/getting-started/calico-enterprise#get-private-registry-credentials-and-license-key).

#### Push Calico Enterprise images to your private registry

In order to install images from your private registry, you must first
pull the images from Tigera’s registry, re-tag them with your own
registry, and then push the newly tagged images to your own registry.

1.  Use the following commands to pull the required Calico Enterprise
    images.

    ``` highlight
    docker pull quay.io/tigera/operator:__OP_VERSION__
    docker pull quay.io/tigera/cnx-manager:__CE_VERSION__
    docker pull quay.io/tigera/voltron:__CE_VERSION__
    docker pull quay.io/tigera/guardian:__CE_VERSION__
    docker pull quay.io/tigera/cnx-apiserver:__CE_VERSION__
    docker pull quay.io/tigera/cnx-queryserver:__CE_VERSION__
    docker pull quay.io/tigera/kube-controllers:__CE_VERSION__
    docker pull quay.io/tigera/calicoq:__CE_VERSION__
    docker pull quay.io/tigera/typha:__CE_VERSION__
    docker pull quay.io/tigera/calicoctl:__CE_VERSION__
    docker pull quay.io/tigera/cnx-node:__CE_VERSION__
    docker pull quay.io/tigera/dikastes:__CE_VERSION__
    docker pull quay.io/tigera/dex:__CE_VERSION__
    docker pull quay.io/tigera/fluentd:__CE_VERSION__
    docker pull quay.io/tigera/es-proxy:__CE_VERSION__
    docker pull quay.io/tigera/kibana:__CE_VERSION__
    docker pull quay.io/tigera/elasticsearch:__CE_VERSION__
    docker pull quay.io/tigera/elasticsearch:__CE_VERSION__-fips
    docker pull quay.io/tigera/cloud-controllers:__CE_VERSION__
    docker pull quay.io/tigera/intrusion-detection-job-installer:__CE_VERSION__
    docker pull quay.io/tigera/es-curator:__CE_VERSION__
    docker pull quay.io/tigera/intrusion-detection-controller:__CE_VERSION__
    docker pull quay.io/tigera/compliance-controller:__CE_VERSION__
    docker pull quay.io/tigera/compliance-reporter:__CE_VERSION__
    docker pull quay.io/tigera/compliance-snapshotter:__CE_VERSION__
    docker pull quay.io/tigera/compliance-server:__CE_VERSION__
    docker pull quay.io/tigera/compliance-benchmarker:__CE_VERSION__
    docker pull quay.io/tigera/ingress-collector:__CE_VERSION__
    docker pull quay.io/tigera/l7-collector:__CE_VERSION__
    docker pull quay.io/tigera/license-agent:__CE_VERSION__
    docker pull quay.io/tigera/cni:__CE_VERSION__
    docker pull quay.io/tigera/cni:__CE_VERSION__-fips
    docker pull quay.io/tigera/firewall-integration:__CE_VERSION__
    docker pull quay.io/tigera/egress-gateway:__CE_VERSION__
    docker pull quay.io/tigera/honeypod:__CE_VERSION__
    docker pull quay.io/tigera/honeypod-exp-service:__CE_VERSION__
    docker pull quay.io/tigera/honeypod-controller:__CE_VERSION__
    docker pull quay.io/tigera/key-cert-provisioner:__KSP_VERSION__
    docker pull quay.io/tigera/anomaly_detection_jobs:__CE_VERSION__
    docker pull quay.io/tigera/anomaly-detection-api:__CE_VERSION__
    docker pull quay.io/tigera/elasticsearch-metrics:__CE_VERSION__
    docker pull quay.io/tigera/packetcapture:__CE_VERSION__
    docker pull quay.io/tigera/prometheus:__CE_VERSION__
    docker pull quay.io/tigera/prometheus-operator:__CE_VERSION__
    docker pull quay.io/tigera/prometheus-config-reloader:__CE_VERSION__
    docker pull quay.io/tigera/prometheus-service:__CE_VERSION__
    docker pull quay.io/tigera/es-gateway:__CE_VERSION__
    docker pull quay.io/tigera/deep-packet-inspection:__CE_VERSION__
    docker pull quay.io/tigera/eck-operator:__CE_VERSION__
    docker pull quay.io/tigera/alertmanager:__CE_VERSION__
    docker pull quay.io/tigera/envoy:__CE_VERSION__
    docker pull quay.io/tigera/envoy-init:__CE_VERSION__
    docker pull quay.io/tigera/pod2daemon-flexvol:__CE_VERSION__
    docker pull quay.io/tigera/csi:__CE_VERSION__
    docker pull quay.io/tigera/node-driver-registrar:__CE_VERSION__
    ```

    For hybrid Linux + Windows clusters, pull the following Windows
    images.

    ``` highlight
    docker pull quay.io/tigera/fluentd-windows:__CE_VERSION__
    docker pull quay.io/tigera/calico-windows:__CE_VERSION__
    docker pull quay.io/tigera/calico-windows-upgrade:__CE_VERSION__
    ```

2.  Retag the images with the name of your private registry
    `$PRIVATE_REGISTRY`.

    ``` highlight
    ```

    For hybrid Linux + Windows clusters, retag the following Windows
    images with the name of your private registry.

    ``` highlight
    docker tag quay.io/tigera/fluentd-windows:__CE_VERSION__ $PRIVATE_REGISTRY/$IMAGE_PATH/fluentd-windows:__CE_VERSION__
    docker tag quay.io/tigera/calico-windows:__CE_VERSION__ $PRIVATE_REGISTRY/$IMAGE_PATH/calico-windows:__CE_VERSION__
    docker tag quay.io/tigera/calico-windows-upgrade:__CE_VERSION__ $PRIVATE_REGISTRY/$IMAGE_PATH/calico-windows-upgrade:__CE_VERSION__
    ```

3.  Push the images to your private registry.

    ``` highlight
    docker push $PRIVATE_REGISTRY/tigera/operator:__OP_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/cnx-manager:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/voltron:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/guardian:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/cnx-apiserver:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/cnx-queryserver:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/kube-controllers:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/calicoq:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/typha:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/calicoctl:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/cnx-node:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/dikastes:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/dex:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/fluentd:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/es-proxy:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/kibana:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/elasticsearch:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/elasticsearch:__CE_VERSION__-fips
    docker push $PRIVATE_REGISTRY/tigera/cloud-controllers:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/intrusion-detection-job-installer:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/es-curator:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/intrusion-detection-controller:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/compliance-controller:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/compliance-reporter:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/compliance-snapshotter:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/compliance-server:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/compliance-benchmarker:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/ingress-collector:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/l7-collector:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/license-agent:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/cni:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/cni:__CE_VERSION__-fips
    docker push $PRIVATE_REGISTRY/tigera/firewall-integration:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/egress-gateway:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/honeypod:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/honeypod-exp-service:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/honeypod-controller:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/key-cert-provisioner:__KSP_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/anomaly_detection_jobs:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/anomaly-detection-api:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/elasticsearch-metrics:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/packetcapture:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/prometheus:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/prometheus-operator:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/prometheus-config-reloader:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/prometheus-service:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/es-gateway:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/deep-packet-inspection:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/eck-operator:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/alertmanager:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/envoy:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/envoy-init:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/pod2daemon-flexvol:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/csi:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/tigera/node-driver-registrar:__CE_VERSION__
    ```

    For hybrid Linux + Windows clusters, push the following Windows
    images to your private registry.

    ``` highlight
    docker push $PRIVATE_REGISTRY/$IMAGE_PATH/fluentd-windows:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/$IMAGE_PATH/calico-windows:__CE_VERSION__
    docker push $PRIVATE_REGISTRY/$IMAGE_PATH/calico-windows-upgrade:__CE_VERSION__
    ```

    > **Important**: Do not push the private Calico Enterprise images to
    > a public registry.

#### Run the operator using images from your private registry

Before applying `tigera-operator.yaml`, modify registry references to
use your custom registry:

``` highlight
sed -ie "s?quay.io/?$PRIVATE_REGISTRY?g" tigera-operator.yaml
sed -ie "s?quay.io?$PRIVATE_REGISTRY?g" tigera-operator.yaml
```

Next, ensure that an image pull secret has been configured for your
custom registry. Set the enviroment variable
`PRIVATE_REGISTRY_PULL_SECRET` to the secret name. Then add the image
pull secret to the operator deployment spec:

``` highlight
sed -ie "/serviceAccountName: tigera-operator/a \      imagePullSecrets:\n\      - name: $PRIVATE_REGISTRY_PULL_SECRET"  tigera-operator.yaml
```

If you are installing Prometheus operator as part of Calico Enterprise,
then before applying `tigera-prometheus-operator.yaml`, modify registry
references to use your custom registry:

``` highlight
sed -ie "s?quay.io/?$PRIVATE_REGISTRY?g" tigera-prometheus-operator.yaml
sed -ie "s?quay.io?$PRIVATE_REGISTRY?g" tigera-prometheus-operator.yaml
sed -ie "/serviceAccountName: calico-prometheus-operator/a \      imagePullSecrets:\n\      - name: $PRIVATE_REGISTRY_PULL_SECRET"  tigera-prometheus-operator.yaml
```

Before applying `custom-resources.yaml`, modify registry references to
use your custom registry:

``` highlight
sed -ie "s?quay.io?$PRIVATE_REGISTRY?g" custom-resources.yaml
```

#### Configure the operator to use images from your private registry.

Set the `spec.registry` field of your Installation resource to the name
of your custom registry. For example:

    apiVersion: operator.tigera.io/v1
    kind: Installation
    metadata:
      name: default
    spec:
      variant: TigeraSecureEnterprise
      imagePullSecrets:
        - name: tigera-pull-secret
      registry: myregistry.com

> **Note:** See [the Installation resource reference
> page](https://docs.tigera.io/reference/installation/api) for more
> information on the `imagePullSecrets` and `registry` fields.
