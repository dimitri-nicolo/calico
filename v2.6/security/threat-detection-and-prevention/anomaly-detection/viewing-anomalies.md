---
title: Viewing anomalies
canonical_url: https://docs.tigera.io/v2.4/security/threat-detection-and-prevention/anomaly-detection/viewing-anomalies
---

### Running Anomaly Detection Jobs

Anomaly detection jobs are run by Elastic Stack. Jobs can be [managed using Kibana] or the [Elasticsearch REST API]. Elastic requires a sufficient number of machine learning nodes with adequate memory to run each job.  Elastic offers guidelines for [sizing for machine learning with Elasticsearch].

You can select and run whichever anomaly detection jobs best suit your environment.

### Viewing anomaly detection results

The intrusion detection controller periodically polls Elastic for results from anomaly detection jobs. You can see the results (called "anomalies") in the {{ site.prodname }} Manager. It is normal to not see any events listed on this page.

Open {{ site.prodname }} Manager, and navigate to the “Alerts” page. If any of your pods or replica sets are identified as engaging in anomalous activity, you will see the results listed on this page. Note that anomaly detection results are curated to remove uncertain anomaly detection events, and expected anomalies in Kubernetes (such as pod creation or deletion).

### Viewing unfiltered anomaly detection results

You can view unfiltered anomaly information either using the Kibana UI, or using the [Elasticsearch REST API].  
This section briefly walks through the process of viewing anomalies using Kibana. Refer to the 
[Elasticsearch machine learning documentation] for more information.

1. Access Kibana by clicking the "Kibana" icon along the left side of {{site.prodname}} Manager, or by visting
   the Kibana URL provided by your Elasticsearch admin.
1. If necessary, log into Kibana. Note that your Kibana credentials may not be the same as you use to access
   {{site.prodname}}; if you don't know your Kibana credentials, contact your Elasticsearch admin.
1. Click "Machine Learning" in the left-hand menu.
1. Click "Anomaly Explorer" in the top menu.

From this view, you can see anomalies identified by the anomaly detection jobs.  Anomalies with higher
severity scores deviate more extremely from the typical behavior of your services, and thus are more suspicious
as potential security events.

Use the Job dropdown in the top left to select which anomaly detection jobs to view, and use the Kibana time
selector in the top right to select the time range to view.

The values in the "influenced by" column identify which pods or namespaces are responsible for the anomalous
behavior, and thus where you should concentrate your subsequent investigations.

[managed using Kibana]: https://www.elastic.co/guide/en/kibana/6.4/xpack-ml.html
[sizing for machine learning with Elasticsearch]: https://www.elastic.co/blog/sizing-machine-learning-with-elasticsearch
[Elasticsearch REST API]: https://www.elastic.co/guide/en/elasticsearch/reference/6.4/ml-apis.html
[Elasticsearch machine learning documentation]: https://www.elastic.co/guide/en/elastic-stack-overview/6.4/xpack-ml.html
