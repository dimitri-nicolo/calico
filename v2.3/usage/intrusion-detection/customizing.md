---
title:  Customizing
canonical_url: https://docs.tigera.io/v2.3/usage/intrusion-detection/customizing
---

{{site.tseeprodname}} ships with Tigera-designed intrusion detection jobs, but you may customize the jobs
or design new ones to fit your environment using the full power of Elasticsearch machine learning.

Tigera **highly** recommends that you customize copies of the Tigera-designed jobs with different Job IDs,
rather than modify them in place.  Modification in place may result in loss of those customizations if
you reinstall or upgrade {{site.tseeprodname}}.  To make a copy of a job:

1. Access Kibana by clicking the "Kibana" icon along the left side of {{site.tseeprodname}} Manager, or by visting
   the Kibana URL provided by your Elasticsearch admin.
1. If necessary, log into Kibana. Note that your Kibana credentials may not be the same as you use to access
   {{site.tseeprodname}}; if you don't know your Kibana credentials, contact your Elasticsearch admin.
1. Click "Machine Learning" in the left-hand menu.
1. Click the gear icon to the far right of the job you wish to copy and select "Clone job"

Refer to the [Elasticsearch machine learning documentation] for more information on the configuration
options for machine learning jobs.

[Elasticsearch machine learning documentation]: https://www.elastic.co/guide/en/elastic-stack-overview/6.4/xpack-ml.html
