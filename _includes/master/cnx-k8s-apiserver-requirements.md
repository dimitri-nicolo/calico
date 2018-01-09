Ensure that the kube-apiserver has been started with the appropriate flags.
- Refer to the Kubernetes documentation to
  [Configure the aggregation layer](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/)
  with the proper flags.
- Refer to the [authentication guide]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication) to choose a supported authentication
  mechanism and configure the Kubernetes API server accordingly.
- The Kubernetes API server must be configured to provide HTTPS access.
