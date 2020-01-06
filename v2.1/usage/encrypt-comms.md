---
title: Configuring encryption and authentication
canonical_url: https://docs.tigera.io/v2.3/usage/encrypt-comms
---

## etcd connections

If you are using the etcd datastore, we recommend enabling mutual TLS authentication on
its connections as follows.

- [Configure etcd](https://coreos.com/etcd/docs/latest/op-guide/security.html) to encrypt its
  communications with TLS and require clients to present certificates signed by the etcd certificate
  authority.

- Configure each {{site.tseeprodname}} component to verify the etcd server's identity and to present
  a certificate to the etcd server that is signed by the etcd certificate authority.
  - [{{site.nodecontainer}}](../reference/node/configuration)
  - [`calicoctl`](./calicoctl/configure/etcd)
  - [CNI plugin](../reference/cni-plugin/configuration#etcd-location)
  - [Felix](../reference/felix/configuration#etcd-datastore-configuration) (on
    [bare metal hosts](../getting-started/bare-metal/installation/))
  - {{site.tseeprodname}} API Server
  - {{site.tseeprodname}} Query Server

## kube-apiserver communications (Kubernetes and OpenShift)

### Unidirectional communications

All communications with kube-apiserver occur over TLS 1.2 with client authentication by
default. The {{site.tseeprodname}} components authenticate to kube-apiserver with either an x.509 certificate
or a [JSON web token (JWT)](https://jwt.io/). You do not need to take action to secure these
communications.

### Bidirectional communications

The {{site.tseeprodname}} API Server requires a bidirectional connection to kube-apiserver. By default,
the {{site.tseeprodname}} API Server uses a self-signed TLS certificate and the kube-apiserver does not
verify the signature. We recommend replacing the self-signed TLS certificate with one signed by a
certificate authority and configuring kube-apiserver to verify the signature.

To do so, you must download the manifest that corresponds to your datastore:
**[cnx-etcd.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml){:target="_blank"}**
or **[cnx-kdd.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-kdd.yaml){:target="_blank"}**.
Make the following changes and then reapply the manifest.

1. Remove the line `insecureSkipTLSVerify: true` from the `APIService` section.
1. Uncomment the line `caBundle:` in the `APIService` and append the base64-encoded CA file contents.
1. Uncomment the line `apiserver.key:` in the `cnx-apiserver-certs` `Secret` and append the
   base64-encoded key file contents.
1. Uncomment the line `apiserver.crt:` in the `cnx-apiserver-certs` `Secret` and append the
   base64-encoded certificate file contents.
1. Uncomment the lines associated with `volumeMounts` and `volumes` named `apiserver-certs`.

## Typha connections (Kubernetes)

{% include {{page.version}}/felix-typha-tls-intro.md %}

To use TLS, each Typha instance must have a certificate and key pair signed by
a trusted CA. Typha then only accepts TLS connections, and requires each
connecting client to present a certificate that is signed by a trusted CA and
has an expected identity in its Common Name or URI SAN field.  Either
`ClientCN` or `ClientURISAN` must be configured, and Typha will check the
presented certificate accordingly.

-  For a [SPIFFE](https://github.com/spiffe/spiffe)-compliant deployment you
   should configure `ClientURISAN` with a [SPIFFE
   Identity](https://github.com/spiffe/spiffe/blob/master/standards/SPIFFE-ID.md#2-spiffe-identity)
   and provision client certificates with the same identity in their URI SAN
   field.

-  Alternatively you can configure `ClientCN` and provision client certificates
   with that `ClientCN` value in their Common Name field.

If both those parameters are set, a client certificate only has to match one of
them; this is intended for when a deployment is migrating from using Common
Name to using a URI SAN, to express identity.

| Configuration parameter | Environment variable   | Description | Schema |
| ----------------------- | ---------------------- | ----------- | ------ |
| `CAFile`                | `TYPHA_CAFILE`         | The full path to the certificate file for the Certificate Authorities that Typha trusts for Felix-Typha communications. | string |
| `ClientCN`              | `TYPHA_CLIENTCN`       | If set, the Common Name that each connecting client certificate must have. [Default: not set] | string |
| `ClientURISAN`          | `TYPHA_CLIENTURISAN`   | If set, a URI SAN that each connecting client certificate must have. [Default: not set] | string |
| `ServerCertFile`        | `TYPHA_SERVERCERTFILE` | The full path to the certificate file for this Typha instance. | string |
| `ServerKeyFile`         | `TYPHA_SERVERKEYFILE`  | The full path to the private key file for this Typha instance. | string |

{% include {{page.version}}/felix-typha-tls-howto.md %}

## {{site.tseeprodname}} Manager connections

Tigera {{site.tseeprodname}} Manager's web interface, run from your browser, uses HTTPS to securely communicate
with the {{site.tseeprodname}} Manager, which in turn, communicates with the Kubernetes and {{site.tseeprodname}} API
servers also using HTTPS. Through the installation steps, secure communication between
{{site.tseeprodname}} components should already be configured, but secure communication through your web
browser of choice may not. To verify if this is properly configured, the web browser
you are using should display `Secure` in the address bar.

Before we set up TLS certificates, it is important to understand the traffic
that we are securing. By default, your web browser of choice communicates with
{{site.tseeprodname}} Manager through a
[`NodePort` service](https://kubernetes.io/docs/tutorials/services/source-ip/#source-ip-for-services-with-typenodeport){:target="_blank"}
over port `30003`. The NodePort service passes through packets without modification.
TLS traffic is [terminated](https://en.wikipedia.org/wiki/TLS_termination_proxy){:target="_blank"}
at the {{site.tseeprodname}} Manager. This means that the TLS certificates used to secure traffic
between your web browser and the {{site.tseeprodname}} Manager do not need to be shared or related
to any other TLS certificates that may be used elsewhere in your cluster or when
configuring {{site.tseeprodname}}. The flow of traffic should look like the following:

![{{site.tseeprodname}} Manager traffic diagram]({{site.baseurl}}/images/cnx-tls-mgr-comms.svg){: width="60%" }

> **Note** the `NodePort` service in the above diagram can be replaced with other
> [Kubernetes services](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services---service-types){:target="_blank"}.
> Configuration will vary if another service, such as a load balancer, is placed between the web
> browser and the {{site.tseeprodname}} Manager.
{: .alert .alert-info}

In order to properly configure TLS in the {{site.tseeprodname}} Manager, you will need
certificates and keys signed by an appropriate Certificate Authority (CA).
For more high level information on certificates, keys, and CAs, see
[this blogpost](https://blog.talpor.com/2015/07/ssltls-certificates-beginners-tutorial){:target="_blank"}.

> **Note** It is important when generating your certificates to make sure
> that the Common Name or Subject Alternative Name specified in your certificates
> matches the host name/DNS entry/IP address that is used to access the {{site.tseeprodname}} Manager
> (i.e. what it says in the browser address bar).
{: .alert .alert-info}

Once you have the proper server certificates, you will need to add them to the
{{site.tseeprodname}} Manager. During installation of the {{site.tseeprodname}} Manager, you should have run
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
you will need to kill the {{site.tseeprodname}} Manager pod so that it is restarted to uptake the new
certificates.

```
kubectl delete pod <cnx-manager-pod-name> -n kube-system
```

If your web browser still does not display `Secure` in the address bar, the most
common reasons and their fixes are listed below.

- **Untrusted Certificate Authority**: Your browser may not display `Secure` because
  it does not know (and therefore trust) the certificate authority (CA) that issued
  the certificates that the {{site.tseeprodname}} Manager is using. This is generally caused by using
  self-signed certificates (either generated by Kubernetes or manually). If you have
  certificates signed by a recognized CA, we recommend that you use them with the {{site.tseeprodname}}
  Manager since the browser will automatically recognize them.

  If you opt to use self-signed certificates you can still configure your browser to
  trust the CA on a per-browser basis by importing the CA certificates into the browser.
  In Google Chrome, this can be achieved by selecting Settings, Advanced, Privacy and security,
  Manage certificates, Authorities, Import. This is not recommended since it requires the CA
  to be imported into every browser you access {{site.tseeprodname}} Manager from.

- **Mismatched Common Name or Subject Alternative Name**: If you are still having issues
  securely accessing {{site.tseeprodname}} Manager with TLS, you may want to make sure that the Common Name
  or Subject Alternative Name specified in your certificates matches the host name/DNS
  entry/IP address that is used to access the {{site.tseeprodname}} Manager (i.e. what it says in the browser
  address bar). In Google Chrome you can check the {{site.tseeprodname}} Manager certificate with Developer Tools
  (Ctrl+Shift+I), Security. If you are issued certificates which do not match,
  you will need to reissue the certificates with the correct Common Name or
  Subject Alternative Name and reconfigure {{site.tseeprodname}} Manager following the steps above.
