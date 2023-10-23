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
    docker pull quay.io/tigera/operator:v1.30.2
    docker pull quay.io/tigera/cnx-manager:v3.17.0
    docker pull quay.io/tigera/voltron:v3.17.0
    docker pull quay.io/tigera/guardian:v3.17.0
    docker pull quay.io/tigera/cnx-apiserver:v3.17.0
    docker pull quay.io/tigera/cnx-queryserver:v3.17.0
    docker pull quay.io/tigera/kube-controllers:v3.17.0
    docker pull quay.io/tigera/calicoq:v3.17.0
    docker pull quay.io/tigera/typha:v3.17.0
    docker pull quay.io/tigera/calicoctl:v3.17.0
    docker pull quay.io/tigera/cnx-node:v3.17.0
    docker pull quay.io/tigera/dikastes:v3.17.0
    docker pull quay.io/tigera/dex:v3.17.0
    docker pull quay.io/tigera/fluentd:v3.17.0
    docker pull quay.io/tigera/es-proxy:v3.17.0
    docker pull quay.io/tigera/kibana:v3.17.0
    docker pull quay.io/tigera/elasticsearch:v3.17.0
    docker pull quay.io/tigera/elasticsearch:v3.17.0-fips
    docker pull quay.io/tigera/cloud-controllers:v3.17.0
    docker pull quay.io/tigera/intrusion-detection-job-installer:v3.17.0
    docker pull quay.io/tigera/es-curator:v3.17.0
    docker pull quay.io/tigera/intrusion-detection-controller:v3.17.0
    docker pull quay.io/tigera/compliance-controller:v3.17.0
    docker pull quay.io/tigera/compliance-reporter:v3.17.0
    docker pull quay.io/tigera/compliance-snapshotter:v3.17.0
    docker pull quay.io/tigera/compliance-server:v3.17.0
    docker pull quay.io/tigera/compliance-benchmarker:v3.17.0
    docker pull quay.io/tigera/ingress-collector:v3.17.0
    docker pull quay.io/tigera/l7-collector:v3.17.0
    docker pull quay.io/tigera/license-agent:v3.17.0
    docker pull quay.io/tigera/cni:v3.17.0
    docker pull quay.io/tigera/cni:v3.17.0-fips
    docker pull quay.io/tigera/firewall-integration:v3.17.0
    docker pull quay.io/tigera/egress-gateway:v3.17.0
    docker pull quay.io/tigera/honeypod:v3.17.0
    docker pull quay.io/tigera/honeypod-exp-service:v3.17.0
    docker pull quay.io/tigera/honeypod-controller:v3.17.0
    docker pull quay.io/tigera/key-cert-provisioner:v1.1.7
    docker pull quay.io/tigera/anomaly_detection_jobs:v3.17.0
    docker pull quay.io/tigera/anomaly-detection-api:v3.17.0
    docker pull quay.io/tigera/elasticsearch-metrics:v3.17.0
    docker pull quay.io/tigera/packetcapture:v3.17.0
    docker pull quay.io/tigera/prometheus:v3.17.0
    docker pull quay.io/tigera/prometheus-operator:v3.17.0
    docker pull quay.io/tigera/prometheus-config-reloader:v3.17.0
    docker pull quay.io/tigera/prometheus-service:v3.17.0
    docker pull quay.io/tigera/es-gateway:v3.17.0
    docker pull quay.io/tigera/deep-packet-inspection:v3.17.0
    docker pull quay.io/tigera/eck-operator:v3.17.0
    docker pull quay.io/tigera/alertmanager:v3.17.0
    docker pull quay.io/tigera/envoy:v3.17.0
    docker pull quay.io/tigera/envoy-init:v3.17.0
    docker pull quay.io/tigera/pod2daemon-flexvol:v3.17.0
    docker pull quay.io/tigera/csi:v3.17.0
    docker pull quay.io/tigera/node-driver-registrar:v3.17.0
    ```

    For hybrid Linux + Windows clusters, pull the following Windows
    images.

    ``` highlight
    docker pull quay.io/tigera/fluentd-windows:v3.17.0
    docker pull quay.io/tigera/calico-windows:v3.17.0
    docker pull quay.io/tigera/calico-windows-upgrade:v3.17.0
    ```

2.  Retag the images with the name of your private registry
    `$PRIVATE_REGISTRY`.

    ``` highlight
    docker tag quay.io/tigera/operator:v1.30.2 $PRIVATE_REGISTRY/tigera/operator:v1.30.2
    docker tag quay.io/tigera/cnx-manager:v3.17.0 $PRIVATE_REGISTRY/tigera/cnx-manager:v3.17.0
    docker tag quay.io/tigera/voltron:v3.17.0 $PRIVATE_REGISTRY/tigera/voltron:v3.17.0
    docker tag quay.io/tigera/guardian:v3.17.0 $PRIVATE_REGISTRY/tigera/guardian:v3.17.0
    docker tag quay.io/tigera/cnx-apiserver:v3.17.0 $PRIVATE_REGISTRY/tigera/cnx-apiserver:v3.17.0
    docker tag quay.io/tigera/cnx-queryserver:v3.17.0 $PRIVATE_REGISTRY/tigera/cnx-queryserver:v3.17.0
    docker tag quay.io/tigera/kube-controllers:v3.17.0 $PRIVATE_REGISTRY/tigera/kube-controllers:v3.17.0
    docker tag quay.io/tigera/calicoq:v3.17.0 $PRIVATE_REGISTRY/tigera/calicoq:v3.17.0
    docker tag quay.io/tigera/typha:v3.17.0 $PRIVATE_REGISTRY/tigera/typha:v3.17.0
    docker tag quay.io/tigera/calicoctl:v3.17.0 $PRIVATE_REGISTRY/tigera/calicoctl:v3.17.0
    docker tag quay.io/tigera/cnx-node:v3.17.0 $PRIVATE_REGISTRY/tigera/cnx-node:v3.17.0
    docker tag quay.io/tigera/dikastes:v3.17.0 $PRIVATE_REGISTRY/tigera/dikastes:v3.17.0
    docker tag quay.io/tigera/dex:v3.17.0 $PRIVATE_REGISTRY/tigera/dex:v3.17.0
    docker tag quay.io/tigera/fluentd:v3.17.0 $PRIVATE_REGISTRY/tigera/fluentd:v3.17.0
    docker tag quay.io/tigera/es-proxy:v3.17.0 $PRIVATE_REGISTRY/tigera/es-proxy:v3.17.0
    docker tag quay.io/tigera/kibana:v3.17.0 $PRIVATE_REGISTRY/tigera/kibana:v3.17.0
    docker tag quay.io/tigera/elasticsearch:v3.17.0 $PRIVATE_REGISTRY/tigera/elasticsearch:v3.17.0
    docker tag quay.io/tigera/elasticsearch:v3.17.0-fips $PRIVATE_REGISTRY/tigera/elasticsearch:v3.17.0-fips
    docker tag quay.io/tigera/cloud-controllers:v3.17.0 $PRIVATE_REGISTRY/tigera/cloud-controllers:v3.17.0
    docker tag quay.io/tigera/intrusion-detection-job-installer:v3.17.0 $PRIVATE_REGISTRY/tigera/intrusion-detection-job-installer:v3.17.0
    docker tag quay.io/tigera/es-curator:v3.17.0 $PRIVATE_REGISTRY/tigera/es-curator:v3.17.0
    docker tag quay.io/tigera/intrusion-detection-controller:v3.17.0 $PRIVATE_REGISTRY/tigera/intrusion-detection-controller:v3.17.0
    docker tag quay.io/tigera/compliance-controller:v3.17.0 $PRIVATE_REGISTRY/tigera/compliance-controller:v3.17.0
    docker tag quay.io/tigera/compliance-reporter:v3.17.0 $PRIVATE_REGISTRY/tigera/compliance-reporter:v3.17.0
    docker tag quay.io/tigera/compliance-snapshotter:v3.17.0 $PRIVATE_REGISTRY/tigera/compliance-snapshotter:v3.17.0
    docker tag quay.io/tigera/compliance-server:v3.17.0 $PRIVATE_REGISTRY/tigera/compliance-server:v3.17.0
    docker tag quay.io/tigera/compliance-benchmarker:v3.17.0 $PRIVATE_REGISTRY/tigera/compliance-benchmarker:v3.17.0
    docker tag quay.io/tigera/ingress-collector:v3.17.0 $PRIVATE_REGISTRY/tigera/ingress-collector:v3.17.0
    docker tag quay.io/tigera/l7-collector:v3.17.0 $PRIVATE_REGISTRY/tigera/l7-collector:v3.17.0
    docker tag quay.io/tigera/license-agent:v3.17.0 $PRIVATE_REGISTRY/tigera/license-agent:v3.17.0
    docker tag quay.io/tigera/cni:v3.17.0 $PRIVATE_REGISTRY/tigera/cni:v3.17.0
    docker tag quay.io/tigera/cni:v3.17.0-fips $PRIVATE_REGISTRY/tigera/cni:v3.17.0-fips
    docker tag quay.io/tigera/firewall-integration:v3.17.0 $PRIVATE_REGISTRY/tigera/firewall-integration:v3.17.0
    docker tag quay.io/tigera/egress-gateway:v3.17.0 $PRIVATE_REGISTRY/tigera/egress-gateway:v3.17.0
    docker tag quay.io/tigera/honeypod:v3.17.0 $PRIVATE_REGISTRY/tigera/honeypod:v3.17.0
    docker tag quay.io/tigera/honeypod-exp-service:v3.17.0 $PRIVATE_REGISTRY/tigera/honeypod-exp-service:v3.17.0
    docker tag quay.io/tigera/honeypod-controller:v3.17.0 $PRIVATE_REGISTRY/tigera/honeypod-controller:v3.17.0
    docker tag quay.io/tigera/key-cert-provisioner:v1.1.7 $PRIVATE_REGISTRY/tigera/key-cert-provisioner:v1.1.7
    docker tag quay.io/tigera/anomaly_detection_jobs:v3.17.0 $PRIVATE_REGISTRY/tigera/anomaly_detection_jobs:v3.17.0
    docker tag quay.io/tigera/anomaly-detection-api:v3.17.0 $PRIVATE_REGISTRY/tigera/anomaly-detection-api:v3.17.0
    docker tag quay.io/tigera/elasticsearch-metrics:v3.17.0 $PRIVATE_REGISTRY/tigera/elasticsearch-metrics:v3.17.0
    docker tag quay.io/tigera/packetcapture:v3.17.0 $PRIVATE_REGISTRY/tigera/packetcapture:v3.17.0
    docker tag quay.io/tigera/prometheus:v3.17.0 $PRIVATE_REGISTRY/tigera/prometheus:v3.17.0
    docker tag quay.io/tigera/prometheus-operator:v3.17.0 $PRIVATE_REGISTRY/tigera/prometheus-operator:v3.17.0
    docker tag quay.io/tigera/prometheus-config-reloader:v3.17.0 $PRIVATE_REGISTRY/tigera/prometheus-config-reloader:v3.17.0
    docker tag quay.io/tigera/prometheus-service:v3.17.0 $PRIVATE_REGISTRY/tigera/prometheus-service:v3.17.0
    docker tag quay.io/tigera/es-gateway:v3.17.0 $PRIVATE_REGISTRY/tigera/es-gateway:v3.17.0
    docker tag quay.io/tigera/deep-packet-inspection:v3.17.0 $PRIVATE_REGISTRY/tigera/deep-packet-inspection:v3.17.0
    docker tag quay.io/tigera/eck-operator:v3.17.0 $PRIVATE_REGISTRY/tigera/eck-operator:v3.17.0
    docker tag quay.io/tigera/alertmanager:v3.17.0 $PRIVATE_REGISTRY/tigera/alertmanager:v3.17.0
    docker tag quay.io/tigera/envoy:v3.17.0 $PRIVATE_REGISTRY/tigera/envoy:v3.17.0
    docker tag quay.io/tigera/envoy-init:v3.17.0 $PRIVATE_REGISTRY/tigera/envoy-init:v3.17.0
    docker tag quay.io/tigera/pod2daemon-flexvol:v3.17.0 $PRIVATE_REGISTRY/tigera/pod2daemon-flexvol:v3.17.0
    docker tag quay.io/tigera/csi:v3.17.0 $PRIVATE_REGISTRY/tigera/csi:v3.17.0
    docker tag quay.io/tigera/node-driver-registrar:v3.17.0 $PRIVATE_REGISTRY/tigera/node-driver-registrar:v3.17.0
    ```

    For hybrid Linux + Windows clusters, retag the following Windows
    images with the name of your private registry.

    ``` highlight
    docker tag quay.io/tigera/fluentd-windows:v3.17.0 $PRIVATE_REGISTRY/$IMAGE_PATH/fluentd-windows:v3.17.0
    docker tag quay.io/tigera/calico-windows:v3.17.0 $PRIVATE_REGISTRY/$IMAGE_PATH/calico-windows:v3.17.0
    docker tag quay.io/tigera/calico-windows-upgrade:v3.17.0 $PRIVATE_REGISTRY/$IMAGE_PATH/calico-windows-upgrade:v3.17.0
    ```

3.  Push the images to your private registry.

    ``` highlight
    docker push $PRIVATE_REGISTRY/tigera/operator:v1.30.2
    docker push $PRIVATE_REGISTRY/tigera/cnx-manager:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/voltron:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/guardian:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/cnx-apiserver:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/cnx-queryserver:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/kube-controllers:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/calicoq:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/typha:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/calicoctl:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/cnx-node:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/dikastes:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/dex:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/fluentd:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/es-proxy:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/kibana:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/elasticsearch:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/elasticsearch:v3.17.0-fips
    docker push $PRIVATE_REGISTRY/tigera/cloud-controllers:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/intrusion-detection-job-installer:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/es-curator:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/intrusion-detection-controller:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/compliance-controller:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/compliance-reporter:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/compliance-snapshotter:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/compliance-server:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/compliance-benchmarker:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/ingress-collector:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/l7-collector:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/license-agent:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/cni:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/cni:v3.17.0-fips
    docker push $PRIVATE_REGISTRY/tigera/firewall-integration:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/egress-gateway:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/honeypod:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/honeypod-exp-service:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/honeypod-controller:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/key-cert-provisioner:v1.1.7
    docker push $PRIVATE_REGISTRY/tigera/anomaly_detection_jobs:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/anomaly-detection-api:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/elasticsearch-metrics:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/packetcapture:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/prometheus:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/prometheus-operator:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/prometheus-config-reloader:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/prometheus-service:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/es-gateway:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/deep-packet-inspection:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/eck-operator:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/alertmanager:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/envoy:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/envoy-init:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/pod2daemon-flexvol:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/csi:v3.17.0
    docker push $PRIVATE_REGISTRY/tigera/node-driver-registrar:v3.17.0
    ```

    For hybrid Linux + Windows clusters, push the following Windows
    images to your private registry.

    ``` highlight
    docker push $PRIVATE_REGISTRY/$IMAGE_PATH/fluentd-windows:v3.17.0
    docker push $PRIVATE_REGISTRY/$IMAGE_PATH/calico-windows:v3.17.0
    docker push $PRIVATE_REGISTRY/$IMAGE_PATH/calico-windows-upgrade:v3.17.0
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
