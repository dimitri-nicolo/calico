---
title: Configure L7 logs
description: Configure and aggregate L7 logs.
canonical_url: /visibility/elastic/l7/configure
---

>**Note**: This feature is tech preview. Tech preview features may be subject to significant changes before they become GA.
{: .alert .alert-info}

## Big picture

Deploy Envoy and use {{site.prodname}} L7 logs to monitor application activity.

## Value

Just like L3/4 {{site.prodname}} logs, platform operators and
development teams want visibility into L7 logs to see how applications are interacting with each
other. {{site.prodname}} flow logs only display which workloads are communicating
with each other, not the specific request details. {{site.prodname}} provides visibility into L7 traffic without the need of a service mesh.

L7 logs are also key for detecting anomalous behaviors like attempts to
access applications, restricted URLs, and scans for particular URLs.

## Concepts

### About L7 logs

L7 logs capture application interactions from HTTP header data in requests. Data shows what is actually sent in communications between specific pods, providing more specificity than flow logs. (Flow logs capture data only from connections for workload interactions).

Calico Enterprise collects L7 logs by sending the selected traffic through an Envoy proxy.

L7 logs are visible in the Manager UI, service graph, in the HTTP tab.

## Before you begin

**Not supported**

* Windows
* eBPF dataplane
* RKE clusters

**Limitations**

* Traffic selection for L7 metric collection is limited only to ClusterIPs.
* L7 log collection is not supported for Nodeport type services and host-network pod backend services (Ex. kubernetes service in the default namespace).
* Pods accessing itself through ClusterIPs are not supported. 
* When selecting and deselecting traffic for L7 log collection, active connections may be disrupted

**Required**

> **Important**: L7 logs require a minimum of 1 additional GB of log storage per node, per one day retention period. Adjust your [Log Storage](https://docs.tigera.io/maintenance/logstorage/adjust-log-storage-size) now. 
{: .alert .alert-danger}

- Configure Felix for log data collection

  Enable the Policy Sync API in Felix. To do this cluster-wide, modify the `default` FelixConfiguration to set the field `policySyncPathPrefix` to `/var/run/nodeagent`.

    ```bash
    kubectl patch felixconfiguration default --type='merge' -p '{"spec":{"policySyncPathPrefix":"/var/run/nodeagent"}}'
    ```

  Configure L7 log aggregation, retention, and reporting. See the
  [Felix Configuration documentation]({{site.baseurl}}/reference/felix/configuration#calico-enterprise-specific-configuration)
  for more details.

## How to

### Configure L7 logs

#### Step 1: Configure the L7 log collector

In this step, you configure the L7 log collector to gather the L7 logs.

1. Download the manifest file for L7 log collector daemonset.
   ```
   curl {{ "/manifests/l7/daemonset/l7-collector-daemonset.yaml" | absolute_url }} -O
   ```

1. In the “env” section of the `l7-collector` container in `l7-collector-daemonset.yaml`, set the
   following environment variables to meet your needs:

   | Environment Variable                | Default Value                         | Description |
   | ----------------------------------- | ------------------------------------- | ----------- |
   | `ENVOY_LOG_INTERVAL_SECONDS`        | 5 seconds                             | Interval in seconds for sending L7 log information for processing. |
   | `ENVOY_LOG_REQUESTS_PER_INTERVAL`   | Unlimited (-1)                        | Maximum number of unique L7 logs that are sent during each interval. All other requests beyond this limit are tracked in the count of requests. To ignore the maximum limit, set this to any negative number (for example, -1). |
   | `ENVOY_LOG_PATH`                    | `/tmp/envoy.log`                      | Path to envoy log files in the container. This should match `access_log` path in `envoy-config.yaml`|
   | `FELIX_DIAL_TARGET`                 |                                       | Path of the socket for communication with Felix. |
   | `LOG_LEVEL`                         | `Panic`                               | Logging level. There are seven levels: `Trace`, `Debug`, `Info`, `Warning`, `Error`, `Fatal` and `Panic`. |

1. Download the Envoy config.
   ```
   curl {{ "/manifests/l7/daemonset/envoy-config.yaml" | absolute_url }} -O
   ```

1. Create the Envoy config in `calico-system` namespace.
   ```
   kubectl create configmap envoy-config -n calico-system --from-file=envoy-config.yaml
   ```

#### Step 2: Enable L7 log collection

Apply the customized `l7-collector-daemonset.yaml` from Step 1 and ensure that `l7-collector` and `envoy-proxy` containers are in Running state. 

   ```
   kubectl apply -f l7-collector-daemonset.yaml
   ```

Enable L7 log collection daemonset mode in Felix by setting [Felix configuration]({{site.baseurl}}/reference/resources/felixconfig) variable `tproxyMode` to `Enabled` or by setting felix environment variable `FELIX_TPROXYMODE` to `Enabled`.

   ```
   kubectl patch felixconfiguration default --type='merge' -p '{"spec":{"tproxyMode":"Enabled"}}'
   ```

#### Step 3: Select traffic for L7 log collection

1. Annotate the services you wish to collect L7 logs as shown.
   ```
   kubectl annotate svc <service-name> -n <service-namespace> projectcalico.org/l7-logging=true
   ```

2. To disable L7 log collection remove the annotation.
   ```
   kubectl annotate svc <service-name> -n <service-namespace> projectcalico.org/l7-logging-
   ```

#### Step 4: Test your configuration

To test your installation, you must first know the appropriate path to access your cluster.
The path can be either of the following:
* The public address of your cluster/service
* The cluster IP of your application's service (if testing within the cluster)

After identifying the path, `curl` your service with a command similar to the following:
```
curl --head <path to access service>:<optional port>/<path>
```

Now view the L7 logs in Kibana by selecting the `tigera_secure_ee_l7` index pattern. You
should see the relevant L7 data from your request recorded.