---
title: Enabling TLS verification
---

## About enabling TLS verification

The {{site.prodname}} API Server deployed by the provided
**[cnx-etcd.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml){:target="_blank"}** 
and **[cnx-kdd.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-kdd.yaml){:target="_blank"}**
manifests will communicate with the Kubernetes
API Server.  The manifest, by default, requires no updates to work but 
uses a set of hard coded TLS certs and CA. We recommend that you use your
own set of keys when deploying to production.

## Before you begin

You will need to obtain or generate the following in PEM format:
- Certificate Authority (CA) certificate
- certificate signed by the CA
- private key for the generated certificate

## Generating certificate files

1. Create a root key (This is only needed if you are generating your CA)
   ```
   openssl genrsa -out rootCA.key 2048
   ```

1. Create a Certificate Authority (CA) certificate (This is only needed if you are generating your CA)
   ```
   openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.pem
   ```
   At each of the prompts press enter.

1. Generate a private key
   ```
   openssl genrsa -out calico.key 2048
   ```

1. Generate a signing request
   ```
   openssl req -new -key calico.key -out calico.csr
   ```
   At each of the prompts press enter except at the Common Name prompt enter
   `cnx-api.kube-system.svc`


1. Generate the signed certificate
   ```
   openssl x509 -req -in calico.csr -CA rootCA.pem -CAkey rootCA.key -CAcreateserial -out calico.crt -days 500 -sha256
   ```

When including the contents of the CA certificate, generated signed
certificate, and generated private key files the contents must be base64
encoded before being added to the manifest file.
Here is an example command to do the base64 encoding:
`cat rootCA.pem | base64 -w 0`.

## Adding certificate files to the manifest

The **[cnx-etcd.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml){:target="_blank"}** 
and **[cnx-kdd.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-kdd.yaml){:target="_blank"}** manifests must be updated
with the following changes

1. Remove the line `insecureSkipTLSVerify: true` from the `APIService` section.
1. Uncomment the line `caBundle:` in the `APIService` and append the base64 encoded CA file contents.
1. Uncomment the line `apiserver.key:` in the `cnx-apiserver-certs` `Secret` and append the base64 encoded key file contents.
1. Uncomment the line `apiserver.crt:` in the `cnx-apiserver-certs` `Secret` and append the base64 encoded certificate file contents.
1. Uncomment the lines associated with `volumeMounts` and `volumes` named `apiserver-certs`.