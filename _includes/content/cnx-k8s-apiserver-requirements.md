#### Aggregation layer enabled

The aggregation layer of kube-apiserver must be enabled. Refer to the 
[Kubernetes documentation](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/){:target="_blank"}
for details. 

#### Authentication method configured

[Select a supported authentication method]({{site.baseurl}}/getting-started/cnx/create-user-login)
and [configure kube-apiserver](https://kubernetes.io/docs/admin/authentication/){:target="_blank"} accordingly.
  
#### TLS communications enabled

Ensure that kube-apiserver allows TLS communications, which it usually
does by default. Refer to the [Kubernetes documentation](https://kubernetes.io/docs/admin/accessing-the-api/){:target="_blank"}
for more information.
