---
title: Enable anomaly detection
canonical_url: https://docs.tigera.io/v2.4/security/threat-detection-and-prevention/anomaly-detection/enabling
---

## Flow Log Aggregation

By default, {{site.prodname}} collects and aggregates flow logs for all traffic that is allowed to pass to and
from pods, while denied flows are collected but not aggregated by default. Aggregating flow data simplifies 
some pieces of information, as individual flows are generalized according to their common components. This 
allows users to search and visualize traffic patterns as opposed to individual connections.

However, anomaly detection requires that additional flow data be stored. To do so, set the value of `flowLogsFileAggregationKindForAllowed` to 1 using a [FelixConfiguration][felixconfig].

## Enabling anomaly detection jobs

Anomaly detection jobs are included as part of standard {{site.prodname}} installation. You can control
their operation either using the Kibana UI, or using the [Elasticsearch REST API].  This section will briefly
walk through the process of enabling jobs using Kibana.  Refer to the
[Elasticsearch machine learning documentation] for more information.

1. Access Kibana by clicking the "Kibana" icon along the left side of {{site.prodname}} Manager.
1. Note that your Kibana credentials may not be the same as you use to access {{site.prodname}}.
   A default user `elastic` is created and stored in `tigera-secure-es-elastic-user` secret to be used during installation. You can obtain the password using the following command:

      ```
   kubectl -n tigera-elasticsearch get secret tigera-secure-es-elastic-user -o go-template='{{.data.elastic | base64decode}}' && echo
      ```

1. Click "Machine Learning" in the left-hand menu.

From this view, you can see the overview of all the anomaly detection jobs installed. See
[Job Descriptions][jobs] for a full explanation of what each job does.

To enable an anomaly detection job:

1. Click the ellipsis icon at the far right of the job name and select "Start datafeed."
1. Choose the start time, which is how far into the past the job should look for flow logs to analyze.
1. Choose the end time.

The jobs will ignore any flow logs they have already analyzed. If you want to analyze all new records, you
do not need to enter the exact time the job analyzed up to, just select something much prior to the previous
analysis. If you want the job to continue processing flow logs as they arrive, choose "No end time (Real-time
search)" as the end time.

For example, if you want to analyze all past flows and enable continutous real-time analysis, choose "Continue from 1969-12-31" as the start time, and "No end time (Real-time search)" as the end time.

Keep in mind that each job you enable will use CPU and memory on your Elasticsearch cluster. The default memory allocated for each job is 1024 MB and may need to be adjusted based on your cluster's flow log throughput. See [customizing]({{site.baseurl}}/{{page.version}}/security/threat-detection-and-prevention/anomaly-detection/customizing) to change the predefined anomaly detection jobs.

[Elasticsearch REST API]: https://www.elastic.co/guide/en/elasticsearch/reference/6.4/ml-apis.html
[Elasticsearch machine learning documentation]: https://www.elastic.co/guide/en/elastic-stack-overview/6.4/xpack-ml.html
[felixconfig]: ../../../reference/resources/felixconfig
[jobs]: ./job-descriptions
