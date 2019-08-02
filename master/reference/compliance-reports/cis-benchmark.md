---
title: CIS benchmark report
---

To create a CIS benchmark report, create a `GlobalReport` with the `reportType` set to `cis-benchmark`.

The following sample command uses a GlobalReport to create a daily CIS benchmark report that run on all the nodes.

```
kubectl apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata: 
  name: daily-cis-benchmark-report
spec:
  reportType: cis-benchmark
  schedule: 0 0 * * *
```

## Openshift
While there is no extra setup configuration required by the user to generate a benchmark report for Openshift, the result sets will be different than a report generated for regular Kubernetes clusters. Use the [Openshift Container Platform security guide](https://docs.openshift.com/container-platform/3.11/security/securing_container_platform.html) to cross-reference the benchmark results.

## Security Note
Executing the CIS benchmarks requires running a pod with some elevated privileges. This includes access to the host’s process space, and volume mounting certain directories (/var/lib, /etc/systemd, /etc/kubernetes, /usr/bin) as read-only. If this is not considered an acceptable risk to your security organization, you can disable this feature by running the following command:

```
kubectl delete daemonset -n calico-monitoring compliance-benchmarker
```

### Downloadable reports

### total-summary.csv

A textual representation of the dashboard.

| Heading | Description | Format |
|----|----|----|
| startTime | The report interval start time. | RFC3339 string |
| endTime   | The report interval start time. | RFC3339 string |
| type      | The type of benchmark report | string |
| hiPercentageThreshold  | The percentage of passing tests required to rate a node as high | int |
| medPercentageThreshold | The percentage of passing tests required to rate a node as medium | int |
| hiNodeCount  | The number of nodes rated as high | int |
| medNodeCount | The number of nodes rated as medium | int |
| lowNodeCount | The number of nodes rated as low | int |

### node-summary.csv

A .csv file of test result summaries per node.

| Heading | Description | Format |
|----|----|----|
| node | The name of the node. | string |
| version | The version of the platform. | string |
| status | The rating of the node based on percentage of tests passing. | string |
| testsPassing | The number of tests passing. | int |
| testsFailing | The number of tests failing. | int |
| testsUnknown | The number of tests whose results are undetermined due to automation restrictions. | int |
| testsTotal | The total number of tests executed. | int |

#### failed-tests.csv

A .csv file of tests that have failed. 

| Heading | Description | Format |
|----|----|
| nodeName  | Node where the test is executed. | string | 
| testIndex | Index of the test on the Kubernetes CIS benchmark. | string |
| status    | Test results: PASS, FAIL, INFO. | string |
| scored    | Indicates whether the Kubernetes CIS benchmark counts this test towards their scoring. | string |

#### all-tests.csv

A .csv file with tests that were executed on all nodes. Format remains the same as above.
