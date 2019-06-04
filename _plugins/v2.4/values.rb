def gen_values_v2_4(versions, imageNames, imageRegistry, chart)
    versionsYml = gen_chart_specific_values_v2_4(versions, imageNames, imageRegistry, chart)
    versionsYml += <<~EOF
    calicoctl:
      image: #{imageRegistry}#{imageNames["calicoctl"]}
      tag: #{versions["calicoctl"]}
    dikastes:
      image: #{imageRegistry}#{imageNames["dikastes"]}
      tag: #{versions["dikastes"]}
    flexvol:
      # flexvol does not use imageRegistry as it is an external OS image
      image: #{imageNames["flexvol"]}
      tag: #{versions["flexvol"]}
    EOF

    versionsYml += <<~EOF
    intrusionDetectionController:
      image: #{imageRegistry}#{imageNames["intrusion-detection-controller"]}
      tag: #{versions["intrusion-detection-controller"]}

    prometheusOperator:
      tag: #{versions["prometheus-operator"]}
    prometheusConfigReloader:
      tag: #{versions["prometheus-config-reloader"]}
    configmapReload:
      tag: #{versions["configmap-reload"]}
    elasticsearchOperator:
      tag: #{versions["elasticsearch-operator"]}
    EOF
end


def gen_chart_specific_values_v2_4(versions, imageNames, imageRegistry, chart)
  if chart == "tigera-secure-ee"
    versionsYml = <<~EOF
    runElasticsearchOperatorClusterAdmin: false
    createCustomResources: true
    
    # Configuration for setting up the manager UI.
    manager:
      image: #{imageRegistry}#{imageNames["cnxManager"]}
      tag: #{versions["cnx-manager"]}
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
    
    # Configuration for the manager UI proxy.
    managerProxy:
      image: #{imageRegistry}#{imageNames["cnxManagerProxy"]}
      tag: #{versions["cnx-manager-proxy"]}
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
      image: #{imageRegistry}#{imageNames["es-proxy"]}
      tag: #{versions["es-proxy"]}
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
      image: #{imageRegistry}#{imageNames["fluentd"]}
      tag: #{versions["fluentd"]}
      # Set to true to create a security context constraint for fluentd enabling
      # it to ingest logs volume-mounted from the host in environments where doing
      # so is restricted.
      runAsPrivileged: false
      # Optional configurating for changing the Fluentd mount path for log files to ingest.
      logFileMountPath: "/var/log/calico"
    
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
      image: #{imageRegistry}#{imageNames["es-curator"]}
      tag: #{versions["es-curator"]}
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
      image: #{imageRegistry}#{imageNames["elastic-tsee-installer"]}
      tag: #{versions["elastic-tsee-installer"]}
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
      image: #{imageRegistry}#{imageNames["compliance-controller"]}
      tag: #{versions["compliance-controller"]}
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
      image: #{imageRegistry}#{imageNames["compliance-reporter"]}
      tag: #{versions["compliance-reporter"]}
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
      image: #{imageRegistry}#{imageNames["compliance-snapshotter"]}
      tag: #{versions["compliance-snapshotter"]}
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
      image: #{imageRegistry}#{imageNames["compliance-server"]}
      tag: #{versions["compliance-server"]}
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
    
    alertmanager:
      image: #{imageNames["alertmanager"]}
      tag: #{versions["alertmanager"]}
      # Configuration for the service which exposes the Prometheus alertmanager.
      service:
        type: NodePort
        nodePort:
    
    prometheus:
      image: #{imageNames["prometheus"]}
      tag: #{versions["prometheus"]}
      scrapeTargets:
        # Node settings grant control over the Prometheus instance tasked with
        # scraping Tigera Secure EE node.
        node:
          # Configuration for the service which fronts the Prometheus instance scraping Tigera Secure EE node.
          service:
            type: NodePort
            nodePort:
    
      # Create RBAC roles to enable PrometheusOperator to create finalizers.
      createFinalizers: false
    
    kibana:
      tag: #{versions["kibana"]}
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
      tag: #{versions["elasticsearch"]}
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
    
    # Optionally specify docker configuration to be used for imagePullSecrets. 
    # Default to an empty list. 
    #
    # E.g. 
    # imagePullSecrets:
    #   <secret_name>: <.docker/config.json contents>
    imagePullSecrets: {}
    
    # TODO: move to helpers
    platform: ""
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
    
    # initialPool configures the pool used by Calico when using calico-ipam.
    # Note that these settings are only applied during initial install.
    # Changing these settings post-install will have no effect.
    initialPool:
      # The default IPv4 pool to create on startup if none exists. Pod IPs will be
      # chosen from this range. Changing this value after installation will have
      # no effect. This should fall within `--cluster-cidr`.
      cidr: "192.168.0.0/16"
    
      # Can be "None", "CrossSubnet", or "Always"
      ipIpMode: Always
    
    # Configuration for Canal config job 
    configureCanal: 
      # Optional configuration for setting resource limits on the Canal config job container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
    
    # Sets the ipam. Can be 'calico-ipam' or 'host-local'
    ipam: calico-ipam
    
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
        crt:
        ca:
        key:
    
    # Configuration for setting up Calico node
    node:
      image: #{imageRegistry}#{imageNames["node"]}
      tag: #{versions["cnx-node"]}
      env:
        # Optional environment variables for configuring Calico node.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FELIX_LOGSEVERITYSCREEN
        #   value: "debug"
      # Optional configuration for setting resource limits on the Calico node container.
      resources:
        requests:
          cpu: #"250m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
    
    # Configuration for setting up Calico CNI.
    cni:
      # cni does not use imageRegistry as it is an external OS image
      image: #{imageNames["cni"]}
      tag: #{versions["calico/cni"]}
      env:
        # Optional environment variables for configuring Calico CNI.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
    
    # Configuration for setting up Flannel.
    flannel:
      image: #{imageNames["flannel"]}
      tag: #{versions["flannel"]}
      env:
        # Optional environment variables for configuring Flannel.
        # These should match the EnvVar spec of the corev1 Kubernetes API. For example:
        # - name: FOO
        #   value: bar
      # Optional configuration for setting resource limits on the Flannel container.
      resources:
        requests:
          cpu: #"100m"
          memory: #"128Mi"
        limits:
          cpu: #"2000m"
          memory: #"1024Mi"
    
    # Configuration for setting up Calico kube controllers
    kubeControllers:
      image: #{imageRegistry}#{imageNames["kubeControllers"]}
      tag: #{versions["cnx-kube-controllers"]}
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
    
    # Configuration for setting up Typha
    typha:
      image: #{imageRegistry}#{imageNames["typha"]}
      tag: #{versions["typha"]}
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
    
    # Configuration for the Calico aggregated API server.
    apiserver:
      image: #{imageRegistry}#{imageNames["cnxApiserver"]}
      tag: #{versions["cnx-apiserver"]}
      # Authentication information for securing communications between TSEE manager and TSEE apiserver.
      # Leave blank to use self-signed certs.
      tls:
        crt:
        key:
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
    
    # Configuration for the Calico query server.
    queryserver:
      image: #{imageRegistry}#{imageNames["cnxQueryserver"]}
      tag: #{versions["cnx-queryserver"]}
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
      image: #{imageRegistry}#{imageNames["cloudControllers"]}
      tag: #{versions["cloud-controllers"]}
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
