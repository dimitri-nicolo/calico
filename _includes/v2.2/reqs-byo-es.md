#### Elasticsearch cluster requirements

- Elasticsearch version {{site.data.versions[page.version].first.components["elasticsearch"].version}}.
- Kibana version {{site.data.versions[page.version].first.components["kibana"].version}}.
- Kubernetes pods must be able to reach the Elasticsearch cluster.
- The cluster must support username and password authentication, and accounts must be created as follows.

#### User accounts

- Create a `tigera-ee-fluentd` user with permission to send documents to Elasticsearch.
- Create a `tigera-ee-manager` user with permission to issue queries to Elasticsearch.
- The username and password for each user and the CA certificate for the cluster will be needed
  during the installation process.
- Any users who are going to use the Kibana dashboards will need to be given appropriate
  credentials.

#### Security considerations

- {{site.prodname}} Manager users will be authenticated against Kubernetes by logging in through
  a [supported method]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication).
  Kubernetes RBAC is then used to authorize queries from {{site.prodname}} Manager to Elasticsearch.
  From Elasticsearch's perspective all queries will come from the `tigera-ee-manager` user.
- Because there's an authenticating proxy inside {{site.prodname}}, any Kubernetes user
  given permission to access `services/proxy/calico-monitoring/elasticsearch-tigera-elasticsearch`
  will be able to send queries to Elasticsearch as the `tigera-ee-manager` user.