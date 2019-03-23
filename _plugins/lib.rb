def gen_values(versions, imageNames, version, nodecontainer, imageRegistry)
    components = versions[version][0]["components"]
    versionsYml = <<~EOF
    node:
      image: #{imageNames["node"]}
      tag: #{components["cnx-node"]["version"]}
    calicoctl:
      image: #{imageNames["calicoctl"]}
      tag: #{components["calicoctl"]["version"]}
    typha:
      image: #{imageNames["typha"]}
      tag: #{components["typha"]["version"]}
    cni:
      image: #{imageNames["cni"]}
      tag: #{components["calico/cni"]["version"]}
    kubeControllers:
      image: #{imageNames["kubeControllers"]}
      tag: #{components["cnx-kube-controllers"]["version"]}
    flannel:
      image: #{imageNames["flannel"]}
    dikastes:
      image: #{imageNames["dikastes"]}
      tag: #{components["dikastes"]["version"]}
    flexvol:
      image: #{imageNames["flexvol"]}
      tag: #{components["flexvol"]["version"]}
    nodecontainer: #{nodecontainer}
    imageRegistry: #{imageRegistry}
    EOF

    versionsYml += <<~EOF
    cnxApiserver:
      image: #{imageNames["cnxApiserver"]}
      tag: #{components["cnx-apiserver"]["version"]}
    cnxManager:
      image: #{imageNames["cnxManager"]}
      tag: #{components["cnx-manager"]["version"]}
    cnxManagerProxy:
      image: #{imageNames["cnxManagerProxy"]}
      tag: #{components["cnx-manager-proxy"]["version"]}
    cnxQueryserver:
      image: #{imageNames["cnxQueryserver"]}
      tag: #{components["cnx-queryserver"]["version"]}
    cloudControllers:
      image: #{imageNames["cloudControllers"]}
      tag: #{components["cloud-controllers"]["version"]}
    intrusionDetectionController:
      image: #{imageNames["intrusion-detection-controller"]}
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
      image: #{imageNames["fluentd"]}
      tag: #{components["fluentd"]["version"]}
    esCurator:
      image: #{imageNames["es-curator"]}
      tag: #{components["es-curator"]["version"]}
    elasticTseeInstaller:
      image: #{imageNames["elastic-tsee-installer"]}
      tag: #{components["elastic-tsee-installer"]["version"]}
    esProxy:
      image: #{imageNames["es-proxy"]}
      tag: #{components["es-proxy"]["version"]}
    EOF
end
