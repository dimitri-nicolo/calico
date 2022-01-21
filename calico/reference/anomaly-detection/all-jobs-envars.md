---
title: Anomaly detection 
description: Anomaly detectors and descriptions.
canonical_url: /reference/anomaly-detection/all-jobs-envars
---

This topic lists {{site.prodname}} anomaly detectors and their environment variables.

### General anomaly detectors

The following environment variables can be used for both security and performance anomaly detectors.

- **AD_train_interval_minutes** 

     Default: 1440 hours (a full day). Determines the interval between retraining the existing models.

- **AD_search_interval_minutes**

     Default: 30 minutes. Interval between searching for the anomalies.

- **AD_max_docs** 

     Default: 500000 rows. Size of the dataset used for the training. The larger the number, the more precise are the trained models, but the more data read from the Elasticsearch storage, and the training takes more time.

- **Elasticsearch variables** 

     CLUSTER_NAME. Default: “cluster”. In a multi-cluster deployment, you must provide the name of the cluster where the anomaly detector will detects anomalies.

### Performance anomaly detectors 

The following detectors are available for [performance anomalies]({{site.baseurl}}/visibility/performance-hotspots). These detectors search primarily for performance anomalies like the slowness of processes or excessive resource consumption. Performance anomalies may be the result of malicious activity, or just increased activity of legitimate applications.

>**Note**: All sensitivity environment variables (noted by`_SENSITIVITY`), change the **sensitivity** of the detector. The higher the sensitivity, the more suspicious values are treated as anomalies. Note that you can have different default values for different detectors. **Valid range**: 0.0 to 100.0.
{: .alert .alert-info}

#### Numeric field anomaly in DNS log

ID: `generic_dns`. Looks for excessive values in several numeric fields in the `DNS` log. May indicate performance issues like the excessive resource consumption.

Environment variables:

- **AD_GENERIC_DNS_SENSITIVITY**
    
     Default: 4. Decrease this parameter if you want fewer alerts; increase it for more alerts.

- **AD_GENERIC_DNS_FIELDS**
    
     Default: "latency_count,latency_mean,latency_max". List of the `DNS` log numeric fields separated by `,`.  A separate model is trained for each field in this list. Remove a field from this list if you don't want to detect anomalies for this field.

#### Time series anomaly in DNS log

ID: `time_series_dns`. Looks at how numeric parameters in the `DNS` log changed over time. May indicate performance issues like the excessive resource consumption.

Environment variables:

- **AD_TIME_SERIES_DNS_ANOMALY_VALUE_THRESHOLD** 

     Default: 1. Increase this parameter if you want fewer alerts; decrease it for more alerts. 

#### Numeric field anomaly in flows log

ID: `generic_flows`. Looks for excessive values in several numeric fields in the flows log. May indicate performance issues like the excessive resource consumption.

Environment variables:

- **AD_GENERIC_FLOWS_SENSITIVITY**

     Default: 1. Decrease this parameter if you want fewer alerts; increase it for more alerts.

- **AD_GENERIC_FLOWS_FIELDS**

     Default: "bytes_in,bytes_out,num_flows,num_flows_started, num_flows_completed,packets_in,packets_out,num_process_names,num_process_ids,num_original_source_ips". List of the flows log numeric fields separated by ,. A separate model is trained for each field in this list. Remove a field from this list if you don't want to detect anomalies for this field.  

#### Time series anomaly in flows log

ID: `time_series_flows`. Looks for services that respond with unusual numbers of `4xx` and `5xx` HTTP response codes. These codes indicate an error on the server. May mean there is an underlying problem in the server that could interfere with day-to-day operations.

Environment variables: 

- **AD_TIME_SERIES_FLOWS_ANOMALY_VALUE_THRESHOLD** 

     Default: 6. Increase this parameter if you want fewer alerts; decrease it for more alerts.       

#### Numeric field anomaly in L7 log

ID: `generic_l7`. Looks for excessive values in several numeric fields in the `L7` log. May indicate performance issues like the excessive resource consumption.

Environment variables:

- **AD_GENERIC_L7_SENSITIVITY**

     Default: 10. Decrease this parameter if you want fewer alerts; increase it for more alerts.

- **AD_GENERIC_L7_FIELDS**

     Default: "duration_mean,duration_max,bytes_in,bytes_out,count". List of the `L7` log numeric fields separated by `,`. A separate model is trained for each field in this list. Remove a field from this list if you don't want to detect anomalies for this field.

#### Time series anomaly in L7 log

ID: `time_series_l7`. Looks at how numeric parameters in the `L7` log changed over time. May indicate performance issues like the excessive resource consumption.

Environment variables: 

- **AD_TIME_SERIES_L7_ANOMALY_VALUE_THRESHOLD** 

     Default: 6. Increase this parameter if you want fewer alerts; decrease it for more alerts.

{% include /content/anomaly-detection/job-descriptions-common.md %}  

### Security anomaly detectors 

The following detectors are available [security anomalies]({{site.baseurl}}/threat/security-anomalies). These detectors search primarily for security anomalies related to malicious activities. 

>**Note**: All sensitivity environment variables (noted by`_SENSITIVITY`), change the **sensitivity** of the detector. The higher the sensitivity, the more suspicious values are treated as anomalies. Note that you can have different default values for different detectors. **Valid range**: 0.0 to 100.0.
{: .alert .alert-info}

#### Inbound service bytes

ID: `bytes_in`. Looks for services that receive an anomalously high amount of data. Specifically, the detector looks for services that are unusual with respect to their replica set, and replica sets that are unusual with respect to the rest of the cluster. May indicate a denial of service attack, data exfiltration, or other attacks.

Environment variables: n/a

#### Outbound service bytes

ID: `bytes_out`. Looks for pods that send an anomalously high amount of data. Specifically, the detector looks for pods that are unusual with respect to their replica set, and replica sets that are unusual with respect to the rest of the cluster. May indicate a denial of service attack, data exfiltration, or other attacks.

Environment variables: n/a

#### Domain Generation Algorithms (DGA)

ID: `dga`. Looks for the domain names that could be created by the {% include open-new-window.html text='Domain Generation Algorithms (DGA)' url='https://en.wikipedia.org/wiki/Domain_generation_algorithm' %}, frequently used by malware. Generated domain names (URLs) are used to communicate between the malware code and the malware servers. Presence of the DGA may indicate presence of the malware code.

Environment variables: 

- **AD_DGA_SCORE_THRESHOLD** 

     Default: 0.5.  Separates the DGA domain names from "good" domain names. Increase this parameter if you want fewer alerts; decrease it for more alerts.

#### HTTP connection spike

ID:` http_connection_spike`. Looks for the services that get too many HTTP inbound connections. May indicate a denial of service attack.

Environment variables: 

- **AD_HTTP_CONNECTION_SPIKE_SENSITIVITY**

     Default: 20. Decrease this parameter if you want fewer alerts; increase it for more alerts.

#### HTTP response code

ID: `http_response_codes`. Looks for services that respond with unusual numbers of `4xx` and `5xx` HTTP response codes. These codes indicate an error on the server. May mean there is an underlying problem in the server that could interfere with day-to-day operations.

Environment variables:    

- **AD_HTTP_RESPONSE_CODES_THRESHOLD** 

     Default: 0.5. Increase this parameter if you want fewer alerts; decrease it for more alerts. 

- **AD_HTTP_RESPONSE_CODES_EVENTS_PER_DAY_THRESHOLD**

     Default: 0.3. Increase this parameter if you want fewer alerts; drecrease it for more alerts. 

#### Rare HTTP request verbs

ID: `http_verbs`. Looks for the services that sent HTTP requests with rare verbs, like `HEAD`, `CONNECT`, `OPTIONS`. May indicate an attempt to exploit or enumerate web service behaviour.

Environment variables:  

- **AD_HTTP_VERBS_THRESHOLD**

     Default: 0.99. Increase this parameter if you want fewer alerts; decrease it for more alerts.

- **AD_HTTP_VERBS_EVENTS_PER_DAY_THRESHOLD**

     Default: 1. Increase this parameter if you want fewer alerts; decrease it for more alerts.  

#### IP sweep

ID: `ip_sweep`. Looks for pods in your cluster that are sending packets to many destinations. The detector compares pods both with other pods in their replica set, and with other pods in the cluster generally. May indicate an attacker has gained control of a pod and is gathering reconnaissance on what else they can reach.

Environment variables:  

- **AD_ip_sweep_threshold**

     Default: 32. Threshold for triggering an anomaly for the **ip_sweep** detector. This is a number of unique destination IPs called from the specific source_name_aggr in the same source_namespace, and the same time bucket.

#### Port scan

ID: `port_scan`. Looks for pods in your cluster that are sending packets to one destination on multiple ports. The detector compares pods both with other pods in their replica set, and with other pods in the cluster generally. May indicate an attacker has gained control of a pod and is gathering reconnaissance on what else they can reach.

 Environment variables: 

 - **AD_port_scan_threshold** 

      Default: 500. Threshold for triggering an anomaly for the **port_scan** detector. This is a number of unique destination ports called from the specific source_name_aggr in the same source_namespace, and the same time bucket.  

{% include /content/anomaly-detection/job-descriptions-common.md %}          

