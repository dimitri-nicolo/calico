def gen_values(versions, _, imageRegistry, chart, forDocs)
    return gen_chart_specific_values(versions, imageRegistry, chart, forDocs)
end


def gen_chart_specific_values(versions, imageRegistry, chart, forDocs)
  docsOverrides = Hash.new("")
  if forDocs
    docsOverrides["core.apiserver.tls.crt"] = "<replace with base64 encoded certificate>"
    docsOverrides["core.apiserver.tls.key"] = "<replace with base64 encoded private key>"
    docsOverrides["core.apiserver.tls.cabundle"] = "<replace with base64 encoded Certificate Authority bundle>"
    docsOverrides["core.typha.tls.caBundle"] = "<replace with PEM-encoded (not base64) Certificate Authority bundle>"
    docsOverrides["core.typha.tls.typhaCrt"] = "<replace with base64-encoded Typha certificate>" 
    docsOverrides["core.typha.tls.typhaKey"] = "<replace with base64-encoded Typha private key>"
    docsOverrides["core.typha.tls.felixCrt"] = "<replace with base64-encoded Felix certificate>"
    docsOverrides["core.typha.tls.felixKey"] = "<replace with base64-encoded Felix private key>"
  end
  if chart == "tigera-secure-ee"
    versionsYml = <<~EOF
    runElasticsearchOperatorClusterAdmin: false
    createCustomResources: true
    
    # Configuration for setting up the manager UI.
    manager:
      image: #{imageRegistry}#{versions["cnx-manager"].image}
      tag: #{versions["cnx-manager"].version}
      # Authentication information for securing communications between TSEE manager and the web browser.
      # Leave blank to use self-signed certs.
      tls:
        crt:
        key:
      auth:
        # Auth type. TSEE supports Basic, OIDC, Token, and OAuth
        # type: (OIDC | Basic | Token | OAuth)
        type: Basic
        # The authority.  Required if authentication type is OIDC or OAuth.
        # OIDC  authority: "https://accounts.google.com"
        # OAuth authority: "https://<oauth-authority>/oauth/authorize"
        authority: ""
        # The client ID. Required if authentication type is OIDC or OAuth.
        # OIDC  clientID: "<oidc-client-id>"
        # OAuth clientID: "cnx-manager"
        clientID: ""
      env:
        # Optional environment variables for configuring the manager UI.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Configuration for the service which exposes the manager UI.
      service:
        type: ClusterIP
        nodePort:
        loadBalancerIP:
        clusterIP:
      # Optional configuration for setting resource limits on the manager container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
      # kibanaURL is used to populate a link to Kibana from the Manager web UI.
      kibanaURL: 'https://127.0.0.1:30601'
 
    # Configuration for the manager UI proxy.
    voltron:
      image: #{imageRegistry}#{versions["voltron"].image}
      tag: #{versions["voltron"].version}
      env:
        # Optional environment variables for configuring the manager proxy.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Optional configuration for setting resource limits on the manager proxy container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"

    # Configuration for the esProxy container.
    esProxy:
      image: #{imageRegistry}#{versions["es-proxy"].image}
      tag: #{versions["es-proxy"].version}
      env:
        # Optional environment variables for configuring the elasticsearch proxy container.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Optional configuration for setting resource limits on the esProxy container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"
    
    # Configuration for the Tigera custom fluentd.
    fluentd:
      image: #{imageRegistry}#{versions["fluentd"].image}
      tag: #{versions["fluentd"].version}
      # Set to true to create a security context constraint for fluentd enabling
      # it to ingest logs volume-mounted from the host in environments where doing
      # so is restricted.
      runAsPrivileged: false
      # Optional configuration for changing the Fluentd mount path for kube audit log files to ingest.
      kubeAuditMountPath: "/var/log/calico"
    
      env:
        # Optional environment variables for configuring the Tigera fluentd.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Optional configuration for setting resource limits on the Tigera fluentd container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"
    
    # Configuration for the Tigera elasticsearch curator.
    esCurator:
      image: #{imageRegistry}#{versions["es-curator"].image}
      tag: #{versions["es-curator"].version}
      env:
        # Optional environment variables for configuring the elasticsearch curator.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Optional configuration for setting resource limits on the elasticsearch curator container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"
    
    # Configuration for the Tigera elasticsearch dashboard installer job.
    elasticTseeInstaller:
      enable: true
      image: #{imageRegistry}#{versions["elastic-tsee-installer"].image}
      tag: #{versions["elastic-tsee-installer"].version}
      env:
        # Optional environment variables for configuring the elasticsearch dashboard installer job.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Optional configuration for setting resource limits on the elasticsearch dashboard installer container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"
    
    # Configuration for the compliance controller.
    complianceController:
      image: #{imageRegistry}#{versions["compliance-controller"].image}
      tag: #{versions["compliance-controller"].version}
      env:
        # Optional environment variables for configuring the compliance controller.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: LOG_LEVEL
        #   value: "warning"
      # Optional configuration for setting resource limits on the compliance controller container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi" 
    
    # Configuration for the compliance reporter.
    complianceReporter:
      image: #{imageRegistry}#{versions["compliance-reporter"].image}
      tag: #{versions["compliance-reporter"].version}
      # Set to true to create a security context constraint for compliance reporter enabling
      # it to write report logs volume-mounted from the host in environments where doing
      # so is restricted.
      runAsPrivileged: false
      env:
        # Optional environment variables for configuring the compliance reporter.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: LOG_LEVEL
        #   value: "warning"
      # Optional configuration for setting resource limits on the compliance reporter container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"
    
    # Configuration for the compliance snapshotter.
    complianceSnapshotter:
      image: #{imageRegistry}#{versions["compliance-snapshotter"].image}
      tag: #{versions["compliance-snapshotter"].version}
      env:
        # Optional environment variables for configuring the compliance snapshotter.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: LOG_LEVEL
        #   value: "warning"
      # Optional configuration for setting resource limits on the compliance snapshotter container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"

    # Configuration for the compliance server.
    complianceServer:
      image: #{imageRegistry}#{versions["compliance-server"].image}
      tag: #{versions["compliance-server"].version}
      env:
        # Optional environment variables for configuring the compliance server.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: LOG_LEVEL
        #   value: "warning"
      # Optional configuration for setting resource limits on the compliance server container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"
    
    # Configuration for the compliance benchmarker.
    complianceBenchmarker:
      image: #{imageRegistry}#{versions["compliance-benchmarker"].image}
      tag: #{versions["compliance-benchmarker"].version}
      runAsPrivileged: false
      env:
        # Optional environment variables for configuring the compliance server.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: LOG_LEVEL
        #   value: "warning"
      # Optional configuration for setting resource limits on the compliance server container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"500m"
          memory: #"1024Mi"
    
    kibana:
      image: #{imageRegistry}#{versions["kibana"].image}
      tag: #{versions["kibana"].version}
      version: #{versions["eck-kibana"]}
      # The address of your kibana instance.
      host:
      # The port your kibana instance is listening on
      port: 5601
      # Configuration for the service which exposes your Kibana instance.
      service:
        type: ClusterIP
        nodePort:
        loadBalancerIP:
        clusterIP:

    elasticsearch:
      image: #{imageRegistry}#{versions["elasticsearch"].image}
      tag: #{versions["elasticsearch"].version}
      version: #{versions["eck-elasticsearch"]}
      # Information for configuring connections to a BYO elasticsearch cluster.
      # Leave all fields blank to deploy a self-hosted elasticsearch instance.
    
      # The address of your elasticsearch cluster.
      host:
      # The port your elasticsearch is listening on.
      port: 9200
      tls:
        # Authentication information for communications between es-proxy and BYO elasticsearch.
        # The CA used for authenticate es-proxy with elasticsearch.
        ca:
        # Leave blank to use self-signed certs.
        crt:
        key:
        selfSignedCertificate:
          dns: cluster.local
      fluentd:
        # The username and password fluentd should use when connecting to elasticsearch.
        username: tigera-ee-fluentd
        password: ""
      manager:
        # The username and password the manager should use when connecting to elasticsearch.
        username: tigera-ee-manager
        password: ""
      curator:
        # Username and password that the curator should use when connecting to elasticsearch.
        username: tigera-ee-curator
        password: ""
      compliance:
        benchmarker:
          # Username and password for the compliance benchmarker to authenticate with elasticsearch.
          username: tigera-ee-compliance-benchmarker
          password: ""
        controller:
          # Username and password for the compliance controller to authenticate with elasticsearch.
          username: tigera-ee-compliance-controller
          password: ""
        reporter:
          # Username and password for the compliance reporter to authenticate with elasticsearch.
          username: tigera-ee-compliance-reporter
          password: ""
        snapshotter:
          # Username and password for the compliance snpashotter to authenticate with elasticsearch.
          username: tigera-ee-compliance-snapshotter
          password: ""
        server:
          # Username and password for the compliance server to authenticate with elasticsearch.
          username: tigera-ee-compliance-server
          password: ""
      intrusionDetection:
        # Username and password for the intrusion detection controller to authenticate with elasticsearch.
        username: tigera-ee-intrusion-detection
        password: ""
      elasticInstaller:
        # Username and password for the job installer to authenticate with elasticsearch.
        username: tigera-ee-installer
        password: ""
      persistentVolume:
        capacity: 10Gi
      nodeCount: 1
      # Change this value to override the storage class used by Elasticsearch nodes. We recommend choosing a storage
      # class dedicated to Calico Enterprise only to ensure data retention after upgrades from versions before v2.8.0.
      storageClassName: tigera-elasticsearch
      # NodeSelector gives you more control over the nodes that Elasticsearch will run on. The contents of NodeSelector will
      # be added to the PodSpec of the Elasticsearch nodes. For the pod to be eligible to run on a node, the node must have
      # each of the indicated key-value pairs as labels.
      #
      # E.g.
      # nodeSelector:
      #   my-node: ssd-data-node
      nodeSelector: {}
    
    intrusionDetectionController:
      image: #{imageRegistry}#{versions["intrusion-detection-controller"].image}
      tag: #{versions["intrusion-detection-controller"].version}

    prometheusOperator:
      image: #{imageRegistry}#{versions["prometheus-operator"].image}
      tag: #{versions["prometheus-operator"].version}

    prometheusConfigReloader:
      image: #{imageRegistry}#{versions["prometheus-config-reloader"].image}
      tag: #{versions["prometheus-config-reloader"].version}

    elasticsearchOperator:
      image: #{versions["elasticsearch-operator"].registry}/#{versions["elasticsearch-operator"].image}
      tag: #{versions["elasticsearch-operator"].version}
    
    licenseAgent:
      image: #{imageRegistry}#{versions["license-agent"].image}
      tag: #{versions["license-agent"].version}

    firewallIntegration:
      image: #{imageRegistry}#{versions["firewall-integration"].image}
      tag: #{versions["firewall-integration"].version}

    # Optionally specify docker configuration to be used for imagePullSecrets. 
    # Default to an empty list. 
    #
    # E.g. 
    # imagePullSecrets:
    #   <secret_name>: <.docker/config.json contents>
    imagePullSecrets: {}
    EOF
  elsif chart == "tigera-operator"
    versionsYml = <<~EOF
    imagePullSecrets: {}

    installation:
      enabled: true
      variant: TigeraSecureEnterprise
      kubernetesProvider: ""

    apiServer:
      enabled: true

    intrusionDetection:
      enabled: true

    logCollector:
      enabled: true

    logStorage:
      enabled: true
      nodes:
        count: 1

    manager:
      enabled: true

    monitor:
      enabled: true

    compliance:
      enabled: true

    # Optional configuration for setting custom BGP templates where
    # key is the filename of the template and value is the contents of the template.
    bgp: {}

    certs:
      node:
        key:
        cert:
        commonName:
      typha:
        key:
        cert:
        commonName:
        caBundle:
      manager:
        key:
        cert:
      elasticsearch:
        key:
        cert:
      kibana:
        key:
        cert:
      apiServer:
        key:
        cert:
      compliance:
        key:
        cert:

    # Configuration for the tigera operator
    tigeraOperator:
      image: #{versions.fetch("tigera-operator").image}
      version: #{versions.fetch("tigera-operator").version}
      registry: #{versions.fetch("tigera-operator").registry}
      #{if forDocs then 'namespace: "tigera-operator"' end}

    calicoctl:
      enabled: false
      image: #{imageRegistry}#{versions["calicoctl"].image}
      tag: #{versions["calicoctl"].version}
      binPath: /bin

    techPreviewOptions:
      # set to name of desired apparmor policy for the calico-node container and
      # pod will be annotated with 'container.apparmor.security.beta.kubernetes.io/calico-node'
      nodeApparmorPolicyName: ""
    EOF
  elsif chart == "tigera-prometheus-operator"
    versionsYml = <<~EOF
    imagePullSecrets: {}

    installation:
      kubernetesProvider: ""

    prometheusOperator:
      image: #{imageRegistry}#{versions["prometheus-operator"].image}
      tag: #{versions["prometheus-operator"].version}

    prometheusConfigReloader:
      image: #{imageRegistry}#{versions["prometheus-config-reloader"].image}
      tag: #{versions["prometheus-config-reloader"].version}

    EOF
  else 
    versionsYml = <<~EOF
    # Configuration for federation controller
    federation:
      enabled: false
      # Optional configuration for setting resource limits on the federation controller container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
    
    network: calico

    # controlPlaneNodeSelector is a dictionary of node selectors to apply to
    # all 'control-plane' components.
    controlPlaneNodeSelector: {}
    
    # initialPool configures the pool used by Calico when using calico-ipam.
    # Note that these settings are only applied during initial install.
    # Changing these settings post-install will have no effect.
    initialPool:
      # The default IPv4 pool to create on startup if none exists. Pod IPs will be
      # chosen from this range. Changing this value after installation will have
      # no effect. This should fall within `--cluster-cidr`.
      cidr: "192.168.0.0/16"
    
      # Can be "Never", "CrossSubnet", or "Always"
      ipIpMode: Always
    
    # Sets the ipam. Can be 'calico-ipam' or 'host-local'
    ipam: calico-ipam

    # Sets the mtu.
    mtu: 1440

    datastore: kubernetes
    app_layer_policy:
      enabled: false
      configured: false
    
    # Configuration for etcd
    etcd:
      # Endpoints for the etcd instances. This can be a comma separated list of endpoints.
      endpoints:
      # Authentication information for accessing secure etcd instances.
      tls:
        crt: null
        ca: null
        key: null
    # Sets the networking mode. Can be 'calico', 'flannel', or 'none'
    network: calico
    # Sets the ipam. Can be 'calico-ipam' or 'host-local'
    ipam: calico-ipam

    # Sets the mtu.
    mtu: "1440"

    node:
      image: #{imageRegistry}#{versions["cnx-node"].image}
      tag: #{versions["cnx-node"].version}

      # configure which port prometheus metrics are served on.
      # setting to a positive number will also annotate calico-node with the prometheus scrape autodiscovery annotations.
      # set to 0 to disable prometheus metrics altogether.
      prometheusMetricsPort: 9081
      logLevel: info
      env:
        # Optional environment variables for configuring Calico node.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FELIX_REPORTINGINTERVALSECS
        #   value: "500"
      # Optional configuration for setting resource limits on the Calico node container.
      resources:
        requests:
          cpu: #"250m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
      seccompProfile: ""
      appArmorProfile: ""
    
    # Configuration for setting up Calico CNI.
    cni:
      # cni does not use imageRegistry as it is an external OS image
      image: #{versions["tigera-cni"].registry}/#{versions["tigera-cni"].image}
      tag: #{versions["tigera-cni"].version}
      env:
        # Optional environment variables for configuring Calico CNI.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      seccompProfile: ""
      appArmorProfile: ""
    
    # Configuration for setting up Calico kube controllers
    kubeControllers:
      image: #{imageRegistry}#{versions["cnx-kube-controllers"].image}
      tag: #{versions["cnx-kube-controllers"].version}
      env:
        # Optional environment variables for configuring Calico kube controllers.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: LOG_LEVEL
        #   value: debug
      # Optional configuration for setting resource limits on the Calico kube controllers container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
      seccompProfile: ""
      appArmorProfile: ""
    
    # Configuration for setting up Typha
    typha:
      image: #{imageRegistry}#{versions["typha"].image}
      tag: #{versions["typha"].version}
      enabled: false
      env:
        # Optional environment variables for configuring Typha.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: TYPHA_LOGSEVERITYSYS
        #   value: debug
      # Optional configuration for setting resource limits on the Typha container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
      # Authentication information for securing communications between Typha and Felix.
      tls:
        # Leave these blank to use self-signed certs.
        caBundle: #{docsOverrides["core.typha.tls.caBundle"]}
        typhaCrt: #{docsOverrides["core.typha.tls.typhaCrt"]}
        typhaKey: #{docsOverrides["core.typha.tls.typhaKey"]}
        felixCrt: #{docsOverrides["core.typha.tls.felixCrt"]}
        felixKey: #{docsOverrides["core.typha.tls.felixKey"]}
        # Change these if you generated certs with different common names on them
        typhaCommonName: calico-typha
        felixCommonName: calico-felix
      seccompProfile: ""
      appArmorProfile: ""

    # Configuration for the Calico aggregated API server.
    apiserver:
      image: #{imageRegistry}#{versions["cnx-apiserver"].image}
      tag: #{versions["cnx-apiserver"].version}
      # Authentication information for securing communications between TSEE manager and TSEE apiserver.
      # Leave blank to use self-signed certs.
      tls:
        crt: #{docsOverrides["core.apiserver.tls.crt"]}
        key: #{docsOverrides["core.apiserver.tls.key"]}
        cabundle: #{docsOverrides["core.apiserver.tls.cabundle"]}
      runAsPrivileged: false
      env:
        # Optional environment variables for configuring the Calico API Server.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Optional configuration for setting resource limits on the API server container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
      seccompProfile: ""
      appArmorProfile: ""
    
    # Configuration for the Calico query server.
    queryserver:
      image: #{imageRegistry}#{versions["cnx-queryserver"].image}
      tag: #{versions["cnx-queryserver"].version}
      env:
        # Optional environment variables for configuring the Calico query server.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Optional configuration for setting resource limits on the Calico query server container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
      seccompProfile: ""
      appArmorProfile: ""

    calicoctl:
      enabled: false
      image: #{imageRegistry}#{versions["calicoctl"].image}
      tag: #{versions["calicoctl"].version}
      seccompProfile: ""
      appArmorProfile: ""
      binPath: /bin

    dikastes:
      image: #{imageRegistry}#{versions["dikastes"].image}
      tag: #{versions["dikastes"].version}

    flexvol:
      # flexvol does not use imageRegistry as it is an external OS image
      image: #{versions["flexvol"].registry}/#{versions["flexvol"].image}
      tag: #{versions["flexvol"].version}

    # Optional configuration for setting custom BGP templates where
    # key is the filename of the template and value is the contents of the template.
    bgp: {}
    
    # TODO: move to helpers
    rbac: ""
    platform: ""
    
    # Optionally specify docker configuration to be used for imagePullSecrets. 
    # Default to an empty list. 
    #
    # E.g. 
    # imagePullSecrets:
    #   <secret_name>: <.docker/config.json contents>
    imagePullSecrets: {}
    
    # Configuration for the Tigera Cloud Controllers.
    cloudControllers:
      image: #{imageRegistry}#{versions["cloud-controllers"].image}
      tag: #{versions["cloud-controllers"].version}
      enabled: false
      # Optional configuration for setting resource limits on the Cloud Controllers container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
    EOF
  end
end


# Takes versions_yml which is structured as follows:
#
# {"v3.6"=>
#     ["components"=>
#        {"calico/node"=>{"version"=>"v3.6.0"},
#         "typha"=>{"version"=>"v3.6.0"}}]
#
# And for a given version, return a Hash of each components' version by component name e.g:
#
# {"calico/node"=>"v3.6.0",
#   "typha"=>"v3.6.0"}
#
#def parse_versions(versions_yml, version)
#  if not versions_yml.key?(version)
#    raise IndexError.new "requested version '#{version}' not present in versions.yml"
#  end
#
#  components = versions_yml[version][0]["components"].clone
#  return components.each { |key,val| components[key] = val["version"] }
#end
