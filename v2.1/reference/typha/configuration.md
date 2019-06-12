---
title: Configuring Typha
canonical_url: https://docs.tigera.io/v2.3/reference/typha/configuration
---

Configuration for Typha is read from one of two possible locations, in
order, as follows.

1.  Environment variables, prefixed with `TYPHA_`.

2.  The Typha configuration file.  The path to this file defaults to
    `/etc/calico/typha.cfg` but can be overridden using the `-c` or
    `--config-file` options on the command line.

The value of any configuration parameter is the value read from the
*first* location containing a value. For example, if an environment variable
contains a value, it takes precedence.

If not set in any of these locations, most configuration parameters have
defaults, and it should be rare to have to explicitly set them.

The full list of parameters which can be set is as follows.

#### General configuration

| Configuration parameter           | Environment variable                    | Description  | Schema |
| --------------------------------- | --------------------------------------- | -------------| ------ |
| `DatastoreType`                   | `TYPHA_DATASTORETYPE`                   | The datastore that Typha should read endpoints and policy information from. [Default: `etcdv3`] | `etcdv3`, `kubernetes`|
| `HealthEnabled`                   | `TYPHA_HEALTHENABLED`                   | When enabled, exposes Typha health information via an http endpoint. | boolean |
| `LogFilePath`                     | `TYPHA_LOGFILEPATH`                     | The full path to the Typha log. Set to `none` to disable file logging. [Default: `/var/log/calico/typha.log`] | string |
| `LogSeverityFile`                 | `TYPHA_LOGSEVERITYFILE`                 | The log severity above which logs are sent to the log file. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `LogSeverityScreen`               | `TYPHA_LOGSEVERITYSCREEN`               | The log severity above which logs are sent to the stdout. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `LogSeveritySys`                  | `TYPHA_LOGSEVERITYSYS`                  | The log severity above which logs are sent to the syslog. Set to `""` for no logging to syslog. [Default: `Info`] | `Debug`, `Info`, `Warning`, `Error`, `Fatal` |
| `PrometheusGoMetricsEnabled`      | `TYPHA_PROMETHEUSGOMETRICSENABLED`      | Set to `false` to disable Go runtime metrics collection, which the Prometheus client does by default. This reduces the number of metrics reported, reducing Prometheus load. [Default: `true`]  | boolean |
| `PrometheusMetricsEnabled`        | `TYPHA_PROMETHEUSMETRICSENABLED`        | Set to `true` to enable the Prometheus metrics server in Typha. [Default: `false`] | boolean |
| `PrometheusMetricsPort`           | `TYPHA_PROMETHEUSMETRICSPORT`           | Experimental: TCP port that the Prometheus metrics server should bind to. [Default: `9091`] | int |
| `PrometheusProcessMetricsEnabled` | `TYPHA_PROMETHEUSPROCESSMETRICSENABLED` | Set to `false` to disable process metrics collection, which the Prometheus client does by default. This reduces the number of metrics reported, reducing Prometheus load. [Default: `true`] | boolean |

#### etcd datastore configuration

| Configuration parameter | Environment variable  | Description | Schema |
| ----------------------- | --------------------- | ----------- | ------ |
| `EtcdCaFile`            | `TYPHA_ETCDCAFILE`    | Unnecessary if the CA that issued the etcd server certificate is in the list of trusted root CAs on the Typha host. Otherwise, use this parameter to supply Typha with the path to the file containing the root certificate of the CA that issued the etcd server certificate. Configures Typha to trust the signature on the certificates provided by the etcd server. To disable authentication of the server by Typha, set the value to `none`. [Default: `/etc/ssl/certs/ca-certificates.crt`] | string |
| `EtcdCertFile`          | `TYPHA_ETCDCERTFILE`  | Path to the file containing the client certificate issued to Typha. Enables Typha to participate in mutual TLS authentication and identify itself to the etcd server. Example: `/etc/typha/cert.pem` (optional) | string |
| `EtcdEndpoints`         | `TYPHA_ETCDENDPOINTS` | Comma-delimited list of etcd endpoints to connect to. Example: `http://etcd1:2379,http://etcd2:2379`. | `<scheme>://<ip-or-fqdn>:<port>` |
| `EtcdKeyFile`           | `TYPHA_ETCDKEYFILE`   | Path to the file containing the private key of the Typha client certificate. Enables Typha to participate in mutual TLS authentication and identify itself to the etcd server. Example: `/etc/felix/key.pem` (optional) | string |

#### Kubernetes API datastore configuration

The Kubernetes API datastore driver reads its configuration from Kubernetes-provided environment variables.
