---
title: Provide TLS certificates for log storage
redirect_from: latest/security/comms/log-storage-tls
---

### Big picture

Provide TLS certificates to secure access to {{site.prodname}} to the log storage.

### Value

Providing TLS certificates for {{site.prodname}} components is recommended as part of a zero trust network model for security. 

### Features

This how-to guide uses the following features: 

- **LogStorage**

### Before you begin...

By default, the {{site.prodname}} log storage uses self-signed certificates on connections. To provide TLS certificates,
get the certificate and key pair for the {{site.prodname}} log storage using any X.509-compatible tool or from your organization's 
Certificate Authority. The certificate must have Common Name or a Subject Alternate Name of `calico-enterprise-elasticsearch-cert.tigera-elasticsearch.svc`.

### How to

#### Add TLS certificates for log storage

To provide certificates for use during deployment you must create a secret before applying the 'custom-resource.yaml' or 
before creating the LogStorage resource. To specify certificates for use by the {{site.prodname}} components, create a secret 
using the following command:

```bash
kubectl create secret generic calico-enterprise-elasticsearch-cert -n tigera-operator --from-file=tls.crt=</path/to/certificate-file> --from-file=tls.key=</path/to/key-file>
```

To update existing certificates, run the following command:

```bash
kubectl create secret generic calico-enterprise-elasticsearch-cert -n tigera-operator --from-file=tls.crt=</path/to/certificate-file> --from-file=tls.key=</path/to/key-file> --dry-run -o yaml --save-config | kubectl replace -f -
```

> **Note**: If the {{site.prodname}} log storage already exists, you must manually delete the log storage pods one by one
>after updating the secret. These pods will be in the `tigera-elasticsearch` namespace and be prefixed with `calico-enterprise`.
>The other {{site.prodname}} components will be unable to communicate with the log storage until the pods are restarted.
