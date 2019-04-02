def gen_values(versions, imageNames, imageRegistry)
    versionsYml = <<~EOF
    node:
      image: #{imageRegistry}#{imageNames["node"]}
      tag: #{versions["cnx-node"]}
    calicoctl:
      image: #{imageRegistry}#{imageNames["calicoctl"]}
      tag: #{versions["calicoctl"]}
    typha:
      image: #{imageRegistry}#{imageNames["typha"]}
      tag: #{versions["typha"]}
    cni:
      # cni does not use imageRegistry as it is an external OS image
      image: #{imageNames["cni"]}
      tag: #{versions["calico/cni"]}
    kubeControllers:
      image: #{imageRegistry}#{imageNames["kubeControllers"]}
      tag: #{versions["cnx-kube-controllers"]}
    flannel:
      image: #{imageNames["flannel"]}
      tag: #{versions["flannel"]}
    dikastes:
      image: #{imageRegistry}#{imageNames["dikastes"]}
      tag: #{versions["dikastes"]}
    flexvol:
      # flexvol does not use imageRegistry as it is an external OS image
      image: #{imageNames["flexvol"]}
      tag: #{versions["flexvol"]}
    EOF

    versionsYml += <<~EOF
    cnxApiserver:
      image: #{imageRegistry}#{imageNames["cnxApiserver"]}
      tag: #{versions["cnx-apiserver"]}
    cnxManager:
      image: #{imageRegistry}#{imageNames["cnxManager"]}
      tag: #{versions["cnx-manager"]}
    cnxManagerProxy:
      image: #{imageRegistry}#{imageNames["cnxManagerProxy"]}
      tag: #{versions["cnx-manager-proxy"]}
    cnxQueryserver:
      image: #{imageRegistry}#{imageNames["cnxQueryserver"]}
      tag: #{versions["cnx-queryserver"]}
    cloudControllers:
      image: #{imageRegistry}#{imageNames["cloudControllers"]}
      tag: #{versions["cloud-controllers"]}
    intrusionDetectionController:
      image: #{imageRegistry}#{imageNames["intrusion-detection-controller"]}
      tag: #{versions["intrusion-detection-controller"]}

    prometheusOperator:
      tag: #{versions["prometheus-operator"]}
    prometheus:
      image: #{imageNames["prometheus"]}
      tag: #{versions["prometheus"]}
    alertmanager:
      image: #{imageNames["alertmanager"]}
      tag: #{versions["alertmanager"]}
    prometheusConfigReloader:
      tag: #{versions["prometheus-config-reloader"]}
    configmapReload:
      tag: #{versions["configmap-reload"]}
    elasticsearchOperator:
      tag: #{versions["elasticsearch-operator"]}
    elasticsearch:
      tag: #{versions["elasticsearch"]}
    kibana:
      tag: #{versions["kibana"]}
    fluentd:
      image: #{imageRegistry}#{imageNames["fluentd"]}
      tag: #{versions["fluentd"]}
    esCurator:
      image: #{imageRegistry}#{imageNames["es-curator"]}
      tag: #{versions["es-curator"]}
    elasticTseeInstaller:
      image: #{imageRegistry}#{imageNames["elastic-tsee-installer"]}
      tag: #{versions["elastic-tsee-installer"]}
    esProxy:
      image: #{imageRegistry}#{imageNames["es-proxy"]}
      tag: #{versions["es-proxy"]}
      
    complianceController:
      image: #{imageRegistry}#{imageNames["compliance-controller"]}
      tag: #{versions["compliance-controller"]}
    complianceServer:
      image: #{imageRegistry}#{imageNames["compliance-server"]}
      tag: #{versions["compliance-server"]}
    complianceSnapshotter:
      image: #{imageRegistry}#{imageNames["compliance-snapshotter"]}
      tag: #{versions["compliance-snapshotter"]}

    EOF
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
def parse_versions(versions_yml, version)
  if not versions_yml.key?(version)
    raise IndexError.new "requested version '#{version}' not present in versions.yml"
  end

  components = versions_yml[version][0]["components"].clone
  return components.each { |key,val| components[key] = val["version"] }
end
