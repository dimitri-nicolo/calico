---
title:  Customize anomaly detection jobs
canonical_url: https://docs.tigera.io/v2.4/security/threat-detection-and-prevention/anomaly-detection/customizing

---

{{site.prodname}} ships with Tigera-designed anomaly detection jobs. But you can customize the jobs
or design new ones using the full power of Elasticsearch machine learning.

**Important!** Always clone the predefined anomaly protection jobs before modifying them. When you reinstall or upgrade, predefined anomaly detections jobs can change; if you modify the originals, your modifications are overwritten.

To clone a predefined anomaly protection job:

1. Access Kibana by clicking the "Kibana" icon along the left side of {{site.prodname}} Manager.
1. Note that your Kibana credentials may not be the same as you use to access {{site.prodname}}.
   A default user `elastic` is created and stored in the `tigera-secure-es-elastic-user` secret during installation. You can obtain the password using the following command:

   ```
   kubectl -n tigera-elasticsearch get secret tigera-secure-es-elastic-user -o yaml |  awk '/elastic:/{print $2}' | base64 --decode
   ```

1. Click "Machine Learning" in the left-hand menu.
1. Click the gear icon to the far right of the job you wish to copy and select "Clone job"

See the [Elasticsearch machine learning documentation] for more information on the configuration
options for machine learning jobs.

[Elasticsearch machine learning documentation]: https://www.elastic.co/guide/en/elastic-stack-overview/6.4/xpack-ml.html
