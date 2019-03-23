def gen_values(versions, imageNames, version, imageRegistry)
    components = versions[version][0]["components"]
    versionsYml = <<~EOF
    node:
      image: #{imageRegistry}#{imageNames["node"]}
      tag: #{components["cnx-node"]["version"]}
    calicoctl:
      image: #{imageRegistry}#{imageNames["calicoctl"]}
      tag: #{components["calicoctl"]["version"]}
    typha:
      image: #{imageRegistry}#{imageNames["typha"]}
      tag: #{components["typha"]["version"]}
    cni:
      image: #{imageNames["cni"]}
      tag: #{components["calico/cni"]["version"]}
    kubeControllers:
      image: #{imageRegistry}#{imageNames["kubeControllers"]}
      tag: #{components["cnx-kube-controllers"]["version"]}
    flannel:
      image: #{imageNames["flannel"]}
      tag: #{components["flannel"]["version"]}
    dikastes:
      image: #{imageRegistry}#{imageNames["dikastes"]}
      tag: #{components["dikastes"]["version"]}
    flexvol:
      image: #{imageRegistry}#{imageNames["flexvol"]}
      tag: #{components["flexvol"]["version"]}
    EOF

    versionsYml += <<~EOF
    cnxApiserver:
      image: #{imageRegistry}#{imageNames["cnxApiserver"]}
      tag: #{components["cnx-apiserver"]["version"]}
    cnxManager:
      image: #{imageRegistry}#{imageNames["cnxManager"]}
      tag: #{components["cnx-manager"]["version"]}
    cnxManagerProxy:
      image: #{imageRegistry}#{imageNames["cnxManagerProxy"]}
      tag: #{components["cnx-manager-proxy"]["version"]}
    cnxQueryserver:
      image: #{imageRegistry}#{imageNames["cnxQueryserver"]}
      tag: #{components["cnx-queryserver"]["version"]}
    cloudControllers:
      image: #{imageRegistry}#{imageNames["cloudControllers"]}
      tag: #{components["cloud-controllers"]["version"]}
    intrusionDetectionController:
      image: #{imageRegistry}#{imageNames["intrusion-detection-controller"]}
      tag: #{components["intrusion-detection-controller"]["version"]}

    prometheusOperator:
      tag: #{components["prometheus-operator"]["version"]}
    prometheus:
      image: #{imageNames["prometheus"]}
      tag: #{components["prometheus"]["version"]}
    alertmanager:
      image: #{imageNames["alertmanager"]}
      tag: #{components["alertmanager"]["version"]}
    prometheusConfigReloader:
      tag: #{components["prometheus-config-reloader"]["version"]}
    configmapReload:
      tag: #{components["configmap-reload"]["version"]}
    elasticsearchOperator:
      tag: #{components["elasticsearch-operator"]["version"]}
    elasticsearch:
      tag: #{components["elasticsearch"]["version"]}
    kibana:
      tag: #{components["kibana"]["version"]}
    fluentd:
      image: #{imageRegistry}#{imageNames["fluentd"]}
      tag: #{components["fluentd"]["version"]}
    esCurator:
      image: #{imageRegistry}#{imageNames["es-curator"]}
      tag: #{components["es-curator"]["version"]}
    elasticTseeInstaller:
      image: #{imageRegistry}#{imageNames["elastic-tsee-installer"]}
      tag: #{components["elastic-tsee-installer"]["version"]}
    esProxy:
      image: #{imageRegistry}#{imageNames["es-proxy"]}
      tag: #{components["es-proxy"]["version"]}
    EOF
end
