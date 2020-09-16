---
title: Troubleshooting Elasticsearch
description: Learn how to troubleshoot common issues with Elasticsearch.
canonical_url: '/maintenance/troubleshoot/elasticsearch'
---

### Elasticsearch resources and settings

The following user-configured resources are related to Elasticsearch:

- Log storage settings
  - Elasticsearch (for example, nodeCount amd replicas)
  - Kubernetes (for example, resourceRequirements, storage and nodeSelectors)
  - Tigera (for example, data retention)

- Storage classes and persistent volume provisioners
  - Storage classes define different types of storage
  - Persistent volumes for pod storage can be configured through storage classes or dynamic provisioners from cloud providers

### Diagnostic checklist

1. Rule out network problems, DNS problems, and network policy problems.
1. Check the following logs:

   | Logs                                | Sample command                                               |
   | ----------------------------------- | ------------------------------------------------------------ |
   | Elasticsearch pod                   | `kubectl logs -n tigera-elasticsearch -l common.k8s.elastic.co/type=elasticsearch` |
   | Kibana pod                          | `kubectl logs -n tigera-kibana -l common.k8s.elastic.co/type=kibana` |
   | Tigera operator                     | `kubectl logs -n tigera-operator -l k8s-app=tigera-operator` |
   | Elasticsearch (ECK) operator        | `kubectl logs -n tigera-eck-operator -l k8s-app=elastic-operator` |
   | Kube controllers (often overlooked) | `kubectl logs -n calico-system -l k8s-app=calico-kube-controllers` |
   | Kubernetes API server               | `kubectl logs -n kube-system -l component=kube-apiserver`<br />**Note**: See you platform documentation for specific command if above doesn't work. |
1. Check if there are multiple replicas or statefulsets of Kibana or Elasticsearch.  
   `kubectl get all -n tigera-kibana` and/or `kubectl get all -n tigera-elasticsearch`
1. Check if any of the pods in the `tigera-elasticsearch` namespace are pending.  
   `kubectl get pod -n tigera-elasticsearch`
1. Check the TigerStatus for problems.  
   `kubectl get tigerastatus -o yaml`
1. Check pod status and variables. 
1. Exec into Kibana and verify that the pod is running:
      ```
      $ df
      $ cat /usr/share/kibana/config/kibana.yml
      ```
      Exec into a fluentd pod and verify these variables:
      ```
      echo $ELASTIC_HOST
      echo $ELASTIC_PASSWORD
      echo $ELASTIC_PORT
      echo $ELASTIC_USERNAME
      exit
      ```
1. Exec into Elasticsearch and verify that the pod is running:
      ```
      $ df
      $ cat /mnt/elastic-internal/elasticsearch-config/elasticsearch.yml
      ```
      Then, port forward the Elasticsearch service and see if the variables work:

     `kubectl port-forward -n tigera-elasticsearch svc/tigera-secure-es-http 9200:9200`
     `curl https://localhost:9200 -k -u"$ELASTIC_USERNAME:$ELASTIC_PASSWORD"`  

### Common problems

#### Elasticsearch is pending

**Solution/workaround**: Most often, the reason is due to the absence of a persistent volume that matches the PVC. Check that there is a Kubernetes node with enough CPU and memory. Check if the user is using `dataNodeSelector` in LogStorage.

#### Pod cannot reach Elasticsearch

**Solution/workaround**: Are there any policy changes that may affect the installation? In many cases, removing and reapplying log storage solves the problem.

#### kube-apiserver logs showing many certificate errors

**Solution/workaround**: Sometimes a cluster ends up with multiple replicasets or statefulsets of Kibana or Elasticsearch if you modify the LogStorage resource. To see if this is the problem, run `kubectl get all -n tigera-(elasticsearch/kibana)`. If it is, you can ignore it; the issues will resolve over time.

If you are using a version prior to v2.8, the issue may be caused by the ValidatingWebhookConfiguration. Although we do not support modifying this admission webhook, consider deleting it as follows:

`kubectl delete validatingwebhookconfigurations validating-webhook-configuration`
`kubectl delete service -n tigera-eck-operator elastic-webhook-service`

As a last resort, you can temporarily remove your LogStorage and reapply it to the cluster. **Important!** Be aware that removing LogStorage temporarily removes Elasticsearch from your cluster. Features that depend on LogStorage are temporarily unavailable, including the dashboards in the Manager UI. Data ingestion is also temporarily paused, but will resume when the LogStorage is up and running again.

Follow these steps:

1. Export your current LogStorage CR to a file.  
    `kubectl get logstorage tigera-secure -o yaml --export=true > log-storage.yaml`
   
1. Apply the LogStorage CR.  
    `kubectl apply -f log-storage.yaml`

#### Elasticsearch is slow 

**Solution/workaround**: Start with diagnostics using the Kibana monitoring dashboard. Then, check the QoS of your LogStorage custom resource to see if it is causing throttling (or via the Kubernetes node itself). If the shard count is high, close old shards. Also, another option is to increase the Elasticsearch [CPU and memory]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.LogStorageSpec).

#### Elasticsearch crashes during booting

**Solution/workaround**: Disk provisioners can have issues where the disk does not allow write requests by the Elasticsearch user. Check the logs of the init containers.

#### Kibana dashboard is missing

**Solution/workaround**: Verify that the intrusion detection job is running, or try removing and reapplying: 

```
$ kubectl get intrusiondetections -o yaml > intrusiondetection.yaml

$ kubectl delete -f intrusiondetection.yaml 
intrusiondetection.operator.tigera.io "tigera-secure" deleted  
         
$ kubectl apply -f intrusiondetection.yaml 
intrusiondetection.operator.tigera.io/tigera-secure created
```
