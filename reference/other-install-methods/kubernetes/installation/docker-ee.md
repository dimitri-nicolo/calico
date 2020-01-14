---
title: Installing Calico Enterprise on Docker Enterprise
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/other
---

## Overview

This guide covers installing {{site.prodname}} to secure application connectivity across multi-cloud and legacy environments, with
policy and compliance capabilities for a Docker UCP Kubernetes deployment. The basic steps required are

- Install Docker Enterprise Engine and Docker Universal Control Plane (UCP) in your cluster
- Gather Docker UCP artifacts to modify/update {{site.prodname}} manifests
- Install {{site.prodname}}

## Before you begin

- Ensure that you have a compatible Docker Enterprise installation by following the [Deploy Docker Enterprise on Linux](https://docs.docker.com/v17.09/datacenter/install/linux/) instructions.
For a test environment, a minimum of 3 nodes is required. For a production environment, additional nodes should be deployed.

- Refer to [Docker Reference Architecture: Docker EE Best Practices and Design Considerations](https://success.docker.com/article/docker-ee-best-practices) for details.

- Ensure that your Docker Enterprise cluster also meets the {{site.prodname}} [system requirements](/{{page.version}}/getting-started/kubernetes/requirements).

- Ensure that you have the [credentials for the Tigera private registry](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials)
  and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).

## Install the Kubectl CLI Tool
Install the Kubernetes command-line tool, kubectl, to deploy and manage applications on Kubernetes.
As an example, you can run the following:

```bash
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s \
https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl
```

For more information please refer to [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" platform="docker-ee" %}

{% include {{page.version}}/pull-secret.md platform="docker-ee" %}

## <a name="install-docker-ucp"></a>Docker Enterprise/UCP Installation
During the installation of UCP, the installation will require the following flag `--unmanaged-cni`. This tells UCP to
not install the default Calico networking plugin.

For installing Docker UCP, follow the best practice steps outlined [Install UCP for Production](https://docs.docker.com/ee/ucp/admin/install/).

## Download the UCP Client Certificate Bundle
In order to use the Docker CLI client and kubectl, you just need to download and use a UCP client bundle.

A client bundle contains a private and public key pair that authorizes your requests in UCP. It also contains
utility scripts you can use to configure your Docker and kubectl client tools to talk to your UCP deployment.

Instructions for installing the client bundle can be found at [Docker Universal Control Plane CLI-Based Access](https://docs.docker.com/ee/ucp/user-access/cli/).

Once installed, you can verify kubectl functionality with a command such as `kubectl get nodes`.

## Installing {{site.prodname}}
In this step, you will obtain the etcd information from the UCP client bundle to modify the {{site.prodname}} networking manifest for etcd. The UCP client bundle files
you'll need to update the {{site.prodname}} etcd manifest are:
- ca.pem
- cert.pem
- key.pem

1. Download the {{site.prodname}} networking manifest for etcd.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/calico.yaml -O
   ```

There are a few sections within the `calico-config` section of the `ConfigMap` that need to be updated:
- `etcd_endpoints` set the value to the IP address and port of your etcd server. The port should be 12378 for Docker Enterprise default installations.
- `nodename_file_optional` option needs to be added and set to `true` within the `cni_network_config` section
- `etcd_ca` uncomment and populate this field for TLS enabled etcd with the secret reference`"/calico-secrets/etcd-ca"`
- `etcd_cert` uncomment and populate this field for TLS enabled etcd with the secret reference`"/calico-secrets/etcd-cert"`
- `etcd_key` uncomment and populate this field for TLS enabled etcd with the secret reference`"/calico-secrets/etcd-key"`

   > **Tip**: You can specify more than one `etcd_endpoint` IP using commas as delimiters.
   {: .alert .alert-success}

The `calico-etcd-secrets` secret section also needs to be updated with the base64 encoded strings from the UCP client bundle files.
- `etcd-ca` set the value to the base64 encoded string of the UCP client bundle `ca.pem` file
- `etcd-cert` set the value to the base64 encoded string of the UCP client bundle `cert.pem` file
- `etcd-key` set the value to the base64 encoded string of the UCP client bundle `key.pem` file

Here is an example script to update the above values. The script expects the UCP client bundle etcd files are in `/var/lib/docker/volumes/ucp-node-certs/_data/`
and the `etcd_endpoint` is discoverable inspecting the IP address of the `ucp-kv` container running on the master node.

```bash
#!/bin/bash
BASE64_ETCD_CA=$(sudo cat /var/lib/docker/volumes/ucp-node-certs/_data/ca.pem | base64 -w 0)
BASE64_ETCD_CERT=$(sudo cat /var/lib/docker/volumes/ucp-node-certs/_data/cert.pem | base64 -w 0)
BASE64_ETCD_KEY=$(sudo cat /var/lib/docker/volumes/ucp-node-certs/_data/key.pem | base64 -w 0)
{% raw %}
ETCD_ENDPOINT_IP=$(sudo docker inspect -f '{{.Config.Hostname}}' ucp-kv)
{% endraw %}

sed -i "s?etcd_endpoints: \"http://[0-9.:]*\"?etcd_endpoints: \"https://${ETCD_ENDPOINT_IP}:12378\"?g" calico.yaml
sed -i "s?etcd_endpoints: \"http://<ETCD_IP>:<ETCD_PORT>\"?etcd_endpoints: \"https://${ETCD_ENDPOINT_IP}:12378\"?g" calico.yaml

sed -i "s?etcd_ca: \"\"?etcd_ca: \"\/calico-secrets\/etcd-ca\"?" calico.yaml
sed -i "s?etcd_cert: \"\"?etcd_cert: \"\/calico-secrets\/etcd-cert\"?" calico.yaml
sed -i "s?etcd_key: \"\"?etcd_key: \"\/calico-secrets\/etcd-key\"?" calico.yaml

sed -i "s?# etcd-key: null?etcd-key: ${BASE64_ETCD_KEY}?" calico.yaml
sed -i "s?# etcd-cert: null?etcd-cert: ${BASE64_ETCD_CERT}?" calico.yaml
sed -i "s?# etcd-ca: null?etcd-ca: ${BASE64_ETCD_CA}?" calico.yaml
sed -i "s?\"mtu\": __CNI_MTU__,?\"mtu\": __CNI_MTU__,\n          \"nodename_file_optional\": true,?" calico.yaml
```

Once you have updated the {{site.prodname}} etcd manifest file, apply the manifest with the following cmd:

```bash
kubectl apply -f calico.yaml
```

The output should look similar to the following:
```
kubectl apply -f calico.yaml
configmap/calico-config created
secret/calico-etcd-secrets created
daemonset.extensions/calico-node created
deployment.extensions/calico-kube-controllers created
serviceaccount/calico-kube-controllers created
serviceaccount/calico-node created
```

## Validate the Cluster Services
After applying the {{site.prodname}} etcd manifest, the pods on the UCP Kubernetes cluster should all be healthy and
running. Verify the containers are running with the following cmd

```bash
kubectl get pods --all-namespaces
```

{% include {{page.version}}/cnx-api-install.md init="docker" net="calico" upgrade=include.upgrade %}

{% include {{page.version}}/apply-license.md cli="kubectl" %}

{% include {{page.version}}/cnx-monitor-install.md elasticsearch="operator" platform="docker-ee" %}

1. Continue to [Installing the {{site.prodname}} Manager](#install-cnx-mgr)

## <a name="install-cnx-mgr"></a>Installing the {{site.prodname}} Manager

1. Download the {{site.prodname}} etcd manifest and save the file as cnx.yaml. That is how we will refer to it in later steps.

    ```bash
    curl --compressed -O \
    {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx.yaml
    ```

    > **Note**: If you are upgrading from {{site.prodname}} v2.2 or earlier you will need to make some modifications prior
    > to upgrade to ensure RBAC behavior for tiered policy is unchanged. Please refer to the instructions in the comments for
    > `ClusterRole "ee-calico-tiered-policy-passthru"` in the `cnx-api.yaml` manifest, or the
    > [Configuring {{site.prodname}} RBAC]({{site.url}}/{{page.version}}/reference/cnx/rbac-tiered-policies) documentation
    > for more details.
    {: .alert .alert-info}

{% include {{page.version}}/cnx-cred-sed.md yaml="cnx" %}

1. Update the `cnx-manager` service section and replace `nodePort: 30003` with `nodePort: 33333`

1. Update the `tigera.cnx-manager.kibana-url` value to the URL and port of your Kibana instance. If you are using the
   {{site.prodname}} bundled Kibana, you must change the default port from `30601` to `33601` This port needs to match
   the `tigera-kibana` service `nodePort` in the monitor-calico.yaml manifest as well.

   > **Note**: Docker Enterprise requires non-reserved port ranges to be above 32000.
   {: .alert .alert-info}

1. Update the authentication method in the cnx.yaml manifest to use `Token`

    - Edit the cnx.yaml file and modify the `ConfigMap` named `tigera-cnx-manager-config`
      by setting the value of `tigera.cnx-manager.authentication-type` to `Token`.
      Refer to [Bearer tokens]({{site.url}}/{{page.version}}/reference/cnx/authentication#bearer-tokens)
      for more information. Also refer to Kubernetes' [Putting a bearer token in a request](https://kubernetes.io/docs/admin/authentication/#putting-a-bearer-token-in-a-request){:target="_blank"}
      for further details.<br>

1. Create a secret containing a TLS certificate and the private key used to
   sign it. For Docker EE, use `server.crt` and `server.key` found in the following path `/var/lib/kubelet/pods`
    > **Note**: The file path will be different in each environment: ie
    >
    > /var/lib/kubelet/pods/edb42fb2-c738-11e8-b509-0242ac11000b/volumes/kubernetes.io~secret/tls-secret/server.crt
    >
    > /var/lib/kubelet/pods/edb42fb2-c738-11e8-b509-0242ac11000b/volumes/kubernetes.io~secret/tls-secret/server.key
    {: .alert .alert-info}

    Here is an example script to create the secret using the TLS certificate
    ```bash
    #!/bin/bash
    KUBELET_PODS_PATH=$(sudo find /var/lib/kubelet/pods -name tls-secret)
    echo $KUBELET_PODS_PATH
    sudo cp ${KUBELET_PODS_PATH}/server.crt server.crt
    sudo cp ${KUBELET_PODS_PATH}/server.key server.key

    kubectl -n calico-monitoring create secret generic cnx-manager-tls \
    --from-file=cert=server.crt \
    --from-file=key=server.key
    sudo rm server.crt
    sudo rm server.key
    ```

     > **Note**: Web browsers will warn end users about self-signed certificates.
     > To stop the warnings by using valid certificates
     > instead, refer to [{{site.prodname}} Manager connections]({{site.url}}/{{page.version}}/security/comms/crypto-auth#{{site.prodnamedash}}-manager-connections).
     {: .alert .alert-info}

1. Apply the manifest to install the {{site.prodname}} Manager.

   ```bash
   kubectl apply -f cnx.yaml
   ```

1. Confirm that all of the pods are running with the following command.

   ```bash
   watch kubectl get pods --all-namespaces
   ```

   Wait until each pod has the `STATUS` of `Running`.

1. Continue to [Accessing the {{site.prodname}} UI](#accessing-cnx-mgr)

## <a name="accessing-cnx-mgr"></a>Accessing the {{site.prodname}} UI
Authentication to {{site.prodname}} UI is performed via tokens for Docker Enterprise. The authentication method was specified
in the cnx.yaml file in the previous [Installing {{site.prodname}} Manager](#install-cnx-mgr) section to use a `Token`.
In this section, we will create the `ServiceAccount` which will create a token to use.

In order to access the {{site.prodname}} UI an account and a role needs to be setup in Kubernetes.
Create a `ServiceAccount` account named `cnx-user` and provision it with a `ClusterRoleBinding` with `cluster-admin`
permissions.

```bash
#!/bin/bash
cat << EOF > manager-credentials.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: calico-monitoring
  name: cnx-user
automountServiceAccountToken: false

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: calico-monitoring:cnx-user:cluster-admin
subjects:
- kind: ServiceAccount
  namespace: calico-monitoring
  name: cnx-user
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
EOF
```

Apply the manifest with the following cmd:

```bash
kubectl apply -f manager-credentials.yaml
```

Now retrieve the token from the cluster.
- Find the `cnx-user` service account name.
- Get the service account's secret.
- Retrieve the serets `.data.token` value
- base64 decode the token

An example kubectl command using jsonpath syntax is provided to complete the above steps
```bash
kubectl -n calico-monitoring get secret \
$(kubectl -n calico-monitoring get serviceaccount cnx-user -o jsonpath='{.secrets[*].name}') \
-o jsonpath='{.data.token}' |base64 --decode
```

1. Continue to [Log into the {{site.prodname}} UI](#login-cnx-mgr)

## <a name="login-cnx-mgr"></a>Log into the {{site.prodname}} UI
Open a browser to `https://<docker node>:33333` and use the token retreived above for the {{site.prodname}} Token value. If the node
is not accessible a ssh tunnel may need to be created. For example:

   ```bash
   ssh <jumpbox> -L 127.0.0.1:33333:<docker node>:33333
   ```

> **Note**: The {{site.prodname}} UI port was modified in the cnx.yaml manifest by changing the specified `nodePort: 33333`
value.
{: .alert .alert-info}
