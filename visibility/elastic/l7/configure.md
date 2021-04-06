---
title: Configure L7 logs
description: Configure and aggregate L7 logs.
canonical_url: /visibility/elastic/l7/configure
---

>**Note**: This feature is currently unavailable for RKE and OpenShift clusters.
{: .alert .alert-info}

## Big Picture

Use {{site.prodname}} L7 logs to monitor application activity.

## Value

Just like visibility into L3/4 {{site.prodname}} metrics, platform operators and
development teams want L7 metrics to see how applications are interacting with each
other. {{site.prodname}} flow logs only display which workloads are communicating
with each other, not the specific request details. {{site.prodname}} provides L7
metrics, regardless of if you are using a service mesh or not.

L7 metrics are also key for detecting anomalous behaviors like attempts to
access applications, restricted URLs, and scans for particular URLs.

## Concepts

### About L7 logs

{{site.prodname}} implements an Envoy log collector so L7 metrics can be collected with
or without a service mesh (like Istio).

#### Log data differences

L7 logs and flow logs capture different types of data for troubleshooting.

* L7 logs for application interactions.
  Captures header data from requests. Data shows what is actually sent in communications
  between specific pods, and is greater in volume and specificity than flow logs.

* Flow logs for workload interactions.
  Captures data from connections and shows communications between workloads.

## Before you begin...

In the namespace of the pod that you want to monitor, create a Kubernetes pull secret
for accessing {{site.prodname}} images. This should match the pull secret created
during [{{site.prodname}} installation]({{site.baseurl}}/getting-started/kubernetes/quickstart).

```bash
kubectl create secret generic tigera-pull-secret -n <application pod namespace> --from-file=.dockerconfigjson=<path/to/pull/secret> --type kubernetes.io/dockerconfigjson
```

> **Important**: Enabling L7 logs requires at least an additional 1 GB LogStorage per node per one day retention period. 
>Please adjust your [Log Storage](https://docs.tigera.io/maintenance/logstorage/adjust-log-storage-size) before proceeding. 
{: .alert .alert-danger}

## How to

#### Step 1: Configure the envoy log collector

In this step, you configure the Envoy log collector to gather the L7 metrics.

1. Download the patch file to patch-envoy.yaml.
   ```
   curl {{ "/manifests/l7/patch-envoy.yaml" | absolute_url }} -O
   ```

1. In the “env” section of the envoy-collector container in `patch-envoy.yaml`, set the
   following environment variables to meet your needs:

| Environment Variable                | Default Value                         | Description |
| ----------------------------------- | ------------------------------------- | ----------- |
| `ENVOY_LOG_INTERVAL_SECONDS`        | 5 seconds                             | Interval in seconds for sending L7 log information for processing. |
| `ENVOY_LOG_REQUESTS_PER_INTERVAL`   | Unlimited (-1)                        | Maximum number of unique L7 logs that are sent during each interval. All other requests beyond this limit are tracked in the count of requests. To ignore the maximum limit, set this to any negative number (for example, -1). |
| `ENVOY_LOG_PATH`                    | `/tmp/envoy.log`                      | Path to envoy log files in the container. |
| `FELIX_DIAL_TARGET`                 |                                       | Path of the socket for communication with Felix. |
| `LOG_LEVEL`                         | `Panic`                               | Logging level. There are seven levels: `Trace`, `Debug`, `Info`, `Warning`, `Error`, `Fatal` and `Panic`. |

1. Download the Envoy config.
   ```
   curl {{ "/manifests/l7/envoy-config.yaml" | absolute_url }} -O
   ```

1. Create the Envoy config.
   ```
   kubectl create configmap envoy-config -n <application pod namespace> --from-file=envoy-config.yaml
   ```

#### Step 2: Configure Felix for log data collection

In this step, you enable the Policy Sync API on Felix.

1. Enable the Policy Sync API in Felix. To do this cluster-wide, modify the `default`
FelixConfiguration to set the field `policySyncPathPrefix` to `/var/run/nodeagent`.

    ```bash
    kubectl patch felixconfiguration default --type='merge' -p '{"spec":{"policySyncPathPrefix":"/var/run/nodeagent"}}'
    ```

1. (Optional) Configure L7 log aggregation, retention, and reporting. See the
[Felix Configuration documentation]({{site.baseurl}}/reference/felix/configuration#calico-enterprise-specific-configuration)
for more details.

#### Step 3: Install the envoy log collector

Now that Felix has been configured, apply the customized `patch-envoy.yaml` from Step 1.

```
kubectl patch deployment <name of application deployment> -n <namespace> --patch "$(cat patch-envoy.yaml)"
```

#### Step 4: Test your installation

To test your installation, you must first know the appropriate path to access your cluster.
The path can be either of the following:
* The public address of your cluster/service
* The cluster IP of your application's service (if testing within the cluster)

After identifying the path, `curl` your service with a command similar to the following:
```
curl <path to access service>:<optional port>/<path>
```

Now view the L7 logs in Kibana by selecting the `tigera_secure_ee_l7` index pattern. You
should see the relevant L7 data from your request recorded.
