## Complete list of etcdv3 configuration options

| Configuration file option  | Environment variable | Description                                                                           | Schema
| ---------------------------| -------------------- | ------------------------------------------------------------------------------------- | ------
| `datastoreType`            | `DATASTORE_TYPE`     | Indicates the datastore to use. If unspecified, defaults to `etcdv3`. (optional)      | `kubernetes`, `etcdv3`
| `etcdEndpoints`            | `ETCD_ENDPOINTS`     | A comma separated list of etcd endpoints. Example: `http://127.0.0.1:2379` (required) | string
| `etcdUsername`             | `ETCD_USERNAME`      | User name for RBAC. Example: `user` (optional)                                        | string
| `etcdPassword`             | `ETCD_PASSWORD`      | Password for the given user name. Example: `password` (optional)                      | string
| `etcdKeyFile`              | `ETCD_KEY_FILE`      | Path to the etcd key file. Example: `/etc/calico/key.pem` (optional)                  | string
| `etcdCertFile`             | `ETCD_CERT_FILE`     | Path to the etcd client certificate, Example: `/etc/calico/cert.pem` (optional)       | string
| `etcdCACertFile`           | `ETCD_CA_CERT_FILE`  | Path to the etcd Certificate Authority file. Example: `/etc/calico/ca.pem` (optional) | string

> **Note**:
> - If you are running with TLS enabled, ensure your endpoint addresses use HTTPS.
> - When specifying through environment variables, the `DATASTORE_TYPE` environment
>   is not required for etcdv3.
> - All environment variables may also be prefixed with `CALICO_`, for example
>   `CALICO_DATASTORE_TYPE` and `CALICO_ETCD_ENDPOINTS` etc. may also be used.
>   This is useful if the non-prefixed names clash with existing environment
>   variables defined on your system
> - Previous versions of `{{include.cli}}` supported `ETCD_SCHEME` and `ETC_AUTHORITY` environment
>   variables as a mechanism for specifying the etcd endpoints. These variables are
>   no longer supported. Use `ETCD_ENDPOINTS` instead.
> - In kubeadm deployments, {{site.prodname}} is not configured to use the etcd run by kubeadm 
>   on the Kubernetes master. Instead, it launches its own instance of etcd as a pod, 
>   available at `http://10.96.232.136:6666`. Ensure you are connecting to the correct etcd 
>   or you will not see any of the expected data.
{: .alert .alert-info}