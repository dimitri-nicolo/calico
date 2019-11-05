---
title: Configure encryption and authentication
canonical_url: https://docs.tigera.io/v2.3/usage/encrypt-comms
---

## Connections from {{site.prodname}} components to etcd

If you are using the etcd datastore, we recommend enabling mutual TLS authentication on
its connections as follows.

- [Configure etcd](https://coreos.com/etcd/docs/latest/op-guide/security.html) to encrypt its
  communications with TLS and require clients to present certificates signed by the etcd certificate
  authority.

- Configure each {{site.prodname}} component to verify the etcd server's identity and to present
  a certificate to the etcd server that is signed by the etcd certificate authority.
  - [{{site.nodecontainer}}](../../reference/node/configuration)
  - [`calicoctl`](../../getting-started/calicoctl/configure/etcd)
  - [CNI plugin](../../reference/cni-plugin/configuration#etcd-location) (Kubernetes and OpenShift only)
  - [Kubernetes controllers](../../reference/kube-controllers/configuration#etcdv3) (Kubernetes and OpenShift only)
  - [Felix](../../reference/felix/configuration#etcd-datastore-configuration) (on [bare metal hosts](../../getting-started/bare-metal/installation/))
  - [Typha](../../reference/typha/configuration#etcd-datastore-configuration) (often deployed in
    larger Kubernetes deployments)

### Connections from {{site.prodname}} components to kube-apiserver (Kubernetes and OpenShift)

We recommend enabling TLS on kube-apiserver, as well as the client certificate and JSON web token (JWT)
authentication modules. This ensures that all of its communications with {{site.prodname}} components occur
over TLS. The {{site.prodname}} components present either an X.509 certificate or a JWT to kube-apiserver
so that kube-apiserver can verify their identities.

### Connections from Node to Typha (Kubernetes)

We recommend enabling mutual TLS authentication on connections from Node to Typha.
To do so, you must provision Typha with a server certificate with extended key usage `ServerAuth` and Node with a client
certificate with extended key usage `ClientAuth`. Each service will need the private key associated with their certificate.
In addition, you must configure one of the following.

- **SPIFFE identifiers** (recommended): Generate a [SPIFFE](https://github.com/spiffe/spiffe) identifier for Node,
  and include Node's SPIFFE ID in the `URI SAN` field of its certificate.
  Similarly, generate a [SPIFFE](https://github.com/spiffe/spiffe) identifier for Typha,
  and include Typha's SPIFFE ID in the `URI SAN` field of its certificate.

- **Common Name identifiers**: Set a common name on the Typha certificate and a different
  common name on the Node certificate.

> **Tip**: If you are migrating from Common Name to SPIFFE identifiers, you can set both values.
> If either matches, the communication succeeds.
{: .alert .alert-success}

#### Configure Node to Typha TLS based on your deployment

##### Operator deployment

For clusters installed using operator, see how to [provide TLS certificates for Typha and Node](typha-node-tls).

##### Manual/Helm deployment

Here is an example of how you can secure the Node-Typha communications in your
cluster:

1.  Choose a certificate authority, or set up your own.

1.  Obtain or generate the following leaf certificates, signed by that
    authority, and corresponding keys:

    -  A certificate for each Node with Common Name `typha-client` and
       extended key usage `ClientAuth`.

    -  A certificate for each Typha with Common Name `typha-server` and
       extended key usage `ServerAuth`.

1.  Configure each Typha with:

    -  `CAFile` pointing to the certificate authority certificate

    -  `ServerCertFile` pointing to that Typha's certificate

    -  `ServerKeyFile` pointing to that Typha's key

    -  `ClientCN` set to `typha-client`

    -  `ClientURISAN` unset.

1.  Configure each Node with:

    -  `TyphaCAFile` pointing to the Certificate Authority certificate

    -  `TyphaCertFile` pointing to that Node's certificate

    -  `TyphaKeyFile` pointing to that Node's key

    -  `TyphaCN` set to `typha-server`

    -  `TyphaURISAN` unset.

For a [SPIFFE](https://github.com/spiffe/spiffe)-compliant deployment you can
follow the same procedure as above, except:

1.  Choose [SPIFFE
    Identities](https://github.com/spiffe/spiffe/blob/master/standards/SPIFFE-ID.md#2-spiffe-identity)
    to represent Node and Typha.

1.  When generating leaf certificates for Node and Typha, put the relevant
    SPIFFE Identity in the certificate as a URI SAN.

1.  Leave `ClientCN` and `TyphaCN` unset.

1.  Set Typha's `ClientURISAN` parameter to the SPIFFE Identity for Node that
    you use in each Node certificate.

1.  Set Node's `TyphaURISAN` parameter to the SPIFFE Identity for Typha.

For detailed reference information on these parameters, refer to:

- **Typha**: [Node-Typha TLS configuration](../../reference/typha/configuration#felix-typha-tls-configuration)

- **Node**: [Node-Typha TLS configuration](../../reference/felix/configuration#felix-typha-tls-configuration)

## {{site.prodname}} Manager connections

Tigera {{site.prodname}} Manager's web interface, run from your browser, uses HTTPS to securely communicate
with the {{site.prodname}} Manager, which in turn, communicates with the Kubernetes and {{site.prodname}} API
servers also using HTTPS. Through the installation steps, secure communication between
{{site.prodname}} components should already be configured, but secure communication through your web
browser of choice may not. To verify if this is properly configured, the web browser
you are using should display `Secure` in the address bar.

Before we set up TLS certificates, it is important to understand the traffic
that we are securing. By default, your web browser of choice communicates with
{{site.prodname}} Manager through a
[`NodePort` service](https://kubernetes.io/docs/tutorials/services/source-ip/#source-ip-for-services-with-typenodeport){:target="_blank"}
over port `30003`. The NodePort service passes through packets without modification.
TLS traffic is [terminated](https://en.wikipedia.org/wiki/TLS_termination_proxy){:target="_blank"}
at the {{site.prodname}} Manager. This means that the TLS certificates used to secure traffic
between your web browser and the {{site.prodname}} Manager do not need to be shared or related
to any other TLS certificates that may be used elsewhere in your cluster or when
configuring {{site.prodname}}. The flow of traffic should look like the following:

![{{site.prodname}} Manager traffic diagram]({{site.url}}/images/cnx-tls-mgr-comms.svg){: width="60%" }

> **Note** the `NodePort` service in the above diagram can be replaced with other
> [Kubernetes services](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services---service-types){:target="_blank"}.
> Configuration will vary if another service, such as a load balancer, is placed between the web
> browser and the {{site.prodname}} Manager.
{: .alert .alert-info}

In order to properly configure TLS in the {{site.prodname}} Manager, you will need
certificates and keys signed by an appropriate Certificate Authority (CA).
For more high level information on certificates, keys, and CAs, see
[this blogpost](https://blog.talpor.com/2015/07/ssltls-certificates-beginners-tutorial){:target="_blank"}.

> **Note** It is important when generating your certificates to make sure
> that the Common Name or Subject Alternative Name specified in your certificates
> matches the host name/DNS entry/IP address that is used to access the {{site.prodname}} Manager
> (i.e. what it says in the browser address bar).
{: .alert .alert-info}

### Configuring {{site.prodname}} Manager UI TLS

Configure manager TLS based on how you've deployed {{site.prodname}}
- [Operator deployment](#operator-deployment)
- [Manual/Helm deployment](#manualhelm-deployment)

#### Operator deployment

For clusters installed using the Tigera Operator, see how to [configure manager TLS](manager-tls).

#### Manual/Helm deployment

Once you have the proper server certificates, you will need to add them to the
{{site.prodname}} Manager. During installation of the {{site.prodname}} Manager, you should have run
the following command.

```
sudo kubectl create secret generic cnx-manager-tls --from-file=cert=/etc/kubernetes/pki/apiserver.crt \
--from-file=key=/etc/kubernetes/pki/apiserver.key -n kube-system
```
> **Note** If you are using certificates not from a third party CA,
> you will need to also add your certificates to your web browser.
> See the `Troubleshooting` section for details.

The `.crt` and `.key` files should be the TLS certificate and key respectively
that you are using for securing the traffic with TLS. If you need to replace the
certificates that you specified during installation, rerunning this command while
specifying the correct files will fix the issue. Once the certificates are updated,
you will need to kill the {{site.prodname}} Manager pod so that it is restarted to uptake the new
certificates.

```
kubectl delete pod <cnx-manager-pod-name> -n kube-system
```

### Issues with certificates

If your web browser still does not display `Secure` in the address bar, the most
common reasons and their fixes are listed below.

- **Untrusted Certificate Authority**: Your browser may not display `Secure` because
  it does not know (and therefore trust) the certificate authority (CA) that issued
  the certificates that the {{site.prodname}} Manager is using. This is generally caused by using
  self-signed certificates (either generated by Kubernetes or manually). If you have
  certificates signed by a recognized CA, we recommend that you use them with the {{site.prodname}}
  Manager since the browser will automatically recognize them.

  If you opt to use self-signed certificates you can still configure your browser to
  trust the CA on a per-browser basis by importing the CA certificates into the browser.
  In Google Chrome, this can be achieved by selecting Settings, Advanced, Privacy and security,
  Manage certificates, Authorities, Import. This is not recommended since it requires the CA
  to be imported into every browser you access {{site.prodname}} Manager from.

- **Mismatched Common Name or Subject Alternative Name**: If you are still having issues
  securely accessing {{site.prodname}} Manager with TLS, you may want to make sure that the Common Name
  or Subject Alternative Name specified in your certificates matches the host name/DNS
  entry/IP address that is used to access the {{site.prodname}} Manager (i.e. what it says in the browser
  address bar). In Google Chrome you can check the {{site.prodname}} Manager certificate with Developer Tools
  (Ctrl+Shift+I), Security. If you are issued certificates which do not match,
  you will need to reissue the certificates with the correct Common Name or
  Subject Alternative Name and reconfigure {{site.prodname}} Manager following the steps above.

### Ingress proxies and load balancers

You may wish to configure proxy elements, including hardware or software load balancers, Kubernetes Ingress
proxies etc., between user web browsers and the {{site.prodname}} Manager.  If you do so, configure your proxy
such that {{site.prodname}} Manager receives a HTTPS (TLS) connection, not unencrypted HTTP.

If you require TLS termination at any of these proxy elements, you will need to

  * use a proxy that supports transparent HTTP/2 proxying, for example, [Envoy](https://www.envoyproxy.io/)
  * re-originate a TLS connection from your proxy to {{site.prodname}} Manager, as it expects TLS

If you do not require TLS termination, configure your proxy to "pass thru" the TLS to {{site.prodname}} Manager.

## Prometheus connections

Configure TLS based on your deployment.

> **Note**: Operator deployment does not support configuring TLS for Prometheus.
{: .alert .alert-info}

### Manual/Helm deployment

#### Format your certificates

In order to secure connections between Prometheus and {{site.prodname}}, you will need to first
have the following:

  - A Certificate Authority (CA) certificate (Used to sign the {{site.prodname}}/Prometheus certificate and key)
  (`ca.pem` in this example)
  - A certificate for {{site.prodname}} (`calico.pem` in this example)
  - A private key for {{site.prodname}} (`calico-key.pem` in this example)
  - A certificate for Prometheus (`prom.pem` in this example)
  - A private key for Prometheus (`prom-key.pem` in this example)

For just the {{site.prodname}} certificate, you will need to concatenate your certificate
to the CA certificate.

```
cat calico.pem ca.pem >> concat-cert.pem
```

#### Mount your certificates into {{site.prodname}}

You now need to mount the {{site.prodname}} certificate (the concatenated certificate) and key
into the `{{site.nodecontainer}}` daemonset.

> **Note**: The `{{site.noderunning}}` daemonset is found in the `calico-cnx.yaml` file provided as an example.
{: .alert .alert-info}

Encode the concatenated certificate, the corresponding private key, and the CA
certificate used to sign the Prometheus certificate and key in base64 format. In the
following commands, we call these files `concat-cert.pem`, `calico-key.pem`, and
`ca.pem`, respectively.

```
cat concat-cert.pem | base64 -w 0
cat calico-key.pem | base64 -w 0
cat ca.pem | base64 -w 0
```

Create a secret for the files and place this in the `calico-cnx.yaml` file.

```
apiVersion: v1
kind: Secret
metadata:
  name: certs
  namespace: kube-system
data:
  concat-cert.pem: <Your base64 encoding of concat-cert.pem goes here>
  calico-key.pem: <Your base64 encoding of your calico-key.pem goes here>
  ca.pem: <Your base64 encoding of your ca.pem goes here>
```

Add the appropriate `volumeMounts` and `volumes` to their corresponding sections in
the `{{site.noderunning}}` daemonset.

```
        ...
            volumeMounts:
              - mountPath: /lib/modules
                name: lib-modules
                readOnly: true
              - mountPath: /var/run/calico
                name: var-run-calico
                readOnly: false
              - mountPath: /etc
                name: tls-certs-dir
        ...

        volumes:
          # Used by {{site.nodecontainer}}.
          - name: lib-modules
            hostPath:
              path: /lib/modules
          - name: var-run-calico
            hostPath:
              path: /var/run/calico
          # Used to install CNI.
          - name: cni-bin-dir
            hostPath:
              path: /opt/cni/bin
          - name: cni-net-dir
            hostPath:
              path: /etc/cni/net.d
          - name: tls-certs-dir
            secret:
              secretName: certs
        ...
```

> **Note**: Alternatively, you can mount the location of your certificates directly
into the container instead of using secrets.
{: .alert .alert-info}

```
        ...
        volumes:
          # Used by {{site.nodecontainer}}.
          - name: lib-modules
            hostPath:
              path: /lib/modules
          - name: var-run-calico
            hostPath:
              path: /var/run/calico
          # Used to install CNI.
          - name: cni-bin-dir
            hostPath:
              path: /opt/cni/bin
          - name: cni-net-dir
            hostPath:
              path: /etc/cni/net.d
          - name: tls-certs-dir
            hostPath:
              path: <path to your certs goes here>
        ...
```

Make sure that {{site.prodname}} knows where to read the certificates from by setting the
`FELIX_PROMETHEUSREPORTERCERTFILE` and `FELIX_PROMETHEUSREPORTERKEYFILE` environment
variables. You must also specify where to find the CA certificate used
to sign the client certificate in `FELIX_PROMETHEUSREPORTERCAFILE` (either the CA
certificate from above or the optional CA certificate). You can set
these in the `spec.template.spec.containers.env` section of the `{{site.nodecontainer}}`
daemonset as shown below.

```
              ...
              - name: FELIX_PROMETHEUSREPORTERPORT
                value: "9081"
              # The TLS certs and keys for testing denied packet metrics
              - name: FELIX_PROMETHEUSREPORTERCERTFILE
                value: "/etc/certs/concat-cert.pem"
              - name: FELIX_PROMETHEUSREPORTERKEYFILE
                value: "/etc/certs/calico-key.pem"
              - name: FELIX_PROMETHEUSREPORTERCAFILE
                value: "/etc/certs/ca.pem"
              ...
```

#### Mount your certificates into Prometheus

> **Note**: The following changes need to be made to the `monitor-calico.yaml` file or your equivalent manifest.
{: .alert .alert-info}

Encode your Prometheus certificate, your Prometheus private key, and your
CA certificate in base64 format. In the following commands, we refer to these
files as `prom.pem`, `prom-key.pem`, and `ca.pem` respectively.

```
cat ca.pem | base64 -w 0
cat prom.pem | base64 -w 0
cat prom-key.pem | base64 -w 0
```

Take the base64 output and add it to a secret in the same manifest file as
the service monitor `calico-node-monitor` (found in the `monitor-calico.yaml`
file provided as an example). Make sure that the secret is in the same
namespace as your `calico-node-monitor` (`calico-monitoring` in the example).

```
apiVersion: v1
kind: Secret
metadata:
  name: certs
  namespace: calico-monitoring
data:
  ca.pem: <Your base64 certificate output goes here>
  prom.pem: <Your base64 certificate output goes here>
  prom-key.pem: <Your base64 private key output goes here>
```

Add your secrets so that they can be mounted in the service monitor. In the
manifest for your Prometheus instance (`calico-node-prometheus` in the example),
add a `secrets` section to the `spec` listing the secrets you defined.

```
  ...
  alerting:
      alertmanagers:
        - namespace: calico-monitoring
          name: calico-node-alertmanager
          port: web
          scheme: http
    secrets:
      - certs
  ...
```

Prometheus will mount your secrets at `/etc/prometheus/secrets/` in the container.
Specify the location of your secrets in the `spec.endpoints.tlsConfig` section of your
service monitor (`calico-node-monitor` in the example).  Also make sure to change the
endpoint scheme to use TLS by specifying `scheme: https`.

```
  ...
  endpoints:
    - port: calico-metrics-port
      interval: 5s
      scheme: https
      tlsConfig:
        caFile: /etc/prometheus/secrets/certs/ca.pem
        certFile: /etc/prometheus/secrets/certs/prom.pem
        keyFile: /etc/prometheus/secrets/certs/prom-key.pem
        serverName: <the common name used in the calico certificate goes here>
        insecureSkipVerify: false
  ...
```

> **Note**: Make sure that `serverName` is set to the common name field
> of your {{site.prodname}} certificate. If this is misconfigured, then connections will fail verification.
> If you wish to skip certificate verification, then you can ignore the `serverName`
> field and instead set `insecureskipVerify` to `true`.
{: .alert .alert-info}

#### Reapply your changes

Apply your changes.

```
kubectl apply -f calico-cnx.yaml
kubectl apply -f monitor-calico.yaml
```

Congratulations! Your metrics are now secured with TLS.

> **Note**: Changes to the daemonset may require you to delete the existing pods
> in order to schedule new pods with your changes.
{: .alert .alert-info}
