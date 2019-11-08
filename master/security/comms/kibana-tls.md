---
title: Provide TLS certificates for Kibana
---

### Big picture

Provide TLS certificates to secure access to {{site.prodname}} Kibana.

### Value

Providing TLS certificates for Calico Enterprise components is recommended as part of a zero trust network model for security.
Providing these certificates also ensures the user's browser will trust the Kibana UI.

### Before you begin...

By default, the {{site.prodname}} API server uses self-signed certificates on connections. To provide TLS certificates,
get the certificate and key pair for the {{site.prodname}} Kibana using any X.509-compatible tool or from your organization's 
Certificate Authority. The certificate must have a Subject Alternate Name of `calico-enterprise-kb-http.tigera-kibana.svc`.

### How to

#### Add TLS certificates for Kibana

To provide certificates for use during deployment, you must create a secret before applying the 'custom-resource.yaml' or 
before creating the LogStorage resource. To specify certificates for use in the {{site.prodname}} Kibana, create a secret 
using the following command:

```bash
kubectl create secret generic calico-enterprise-kibana-cert -n tigera-operator --from-file=tls.crt=</path/to/certificate-file> --from-file=tls.key=</path/to/key-file>
```

To update existing certificates, run the following command:

```bash
kubectl create secret generic calico-enterprise-kibana-cert -n tigera-operator --from-file=tls.crt=</path/to/certificate-file> --from-file=tls.key=</path/to/key-file> --dry-run -o yaml --save-config | kubectl replace -f -
```
