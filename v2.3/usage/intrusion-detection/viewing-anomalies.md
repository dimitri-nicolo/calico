---
title: Viewing anomalies
canonical_url: https://docs.tigera.io/v2.3/usage/intrusion-detection/viewing-anomalies
---

Once intrusion detection jobs are running their analysis you will be able to see the results, which are called 
"anomalies."  You can get anomaly information either using the Kibana UI, or using the [Elasticsearch REST API].  
This section will briefly walk thru the process of viewing anomalies using Kibana. Refer to the 
[Elasticsearch machine learning documentation] for more information.

1. Access Kibana by clicking the "Kibana" icon along the left side of {{site.tseeprodname}} Manager, or by visting
   the Kibana URL provided by your Elasticsearch admin.
1. If necessary, log into Kibana. Note that your Kibana credentials may not be the same as you use to access
   {{site.tseeprodname}}; if you don't know your Kibana credentials, contact your Elasticsearch admin.
1. Click "Machine Learning" in the left-hand menu.
1. Click "Anomaly Explorer" in the top menu.

From this view, you can see anomalies identified by the intrusion detection jobs.  Anomalies with higher
severity scores deviate more extremely from the typical behavior of your services, and thus are more suspicious
as possible security events.

Use the Job dropdown in the top left to select which intrusion detection jobs to view, and use the Kibana time
selector in the top right to select the time range to view.

The values in the "influenced by" column identify which pods or namespaces are responsible for the anomalous
behavior, and thus where you should concentrate your subsequent investigations.

[Elasticsearch REST API]: https://www.elastic.co/guide/en/elasticsearch/reference/6.4/ml-apis.html
[Elasticsearch machine learning documentation]: https://www.elastic.co/guide/en/elastic-stack-overview/6.4/xpack-ml.html
