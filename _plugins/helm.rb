require "jekyll"
require "tempfile"
require "yaml"

# This plugin enables jekyll to render helm charts.
# Traditionally, Jekyll will render files which make use of the Liquid templating language.
# This plugin adds a new 'tag' that when specified will pass the input to the Helm binary.
# example use:
#
# {% helm %}
# datastore: kubernetes
# networking: calico
# {% endhelm %}
module Jekyll
  class RenderHelmTagBlock < Liquid::Block
    def initialize(tag_name, extra_args, liquid_options)
      super
      if not extra_args.empty?
        @extra_args = extra_args
      end
    end
    def render(context)
      text = super

      # Because helm hasn't merged stdin support, write the passed-in values.yaml
      # to a tempfile on disk.
      t = Tempfile.new("jhelm")
      t.write(text)
      t.close

      version = context.registers[:page]["version"]
      imageRegistry = context.registers[:page]["registry"]

      # Load the versions.yml file so it can be rewritten in a standard helm format.
      versionFile = YAML::load_file('_data/versions.yml')
      if not versionFile.key?(version)
        puts "skipping because #{version} not present in _versions.yml"
        t.unlink
        return
      end

      components = versionFile[version][0]["components"]

      # In order to preserve backwards compatibility with the existing template system,
      # we process config.yml for imageNames and _versions.yml for tags,
      # then write them in a more standard helm format.
      configYml = YAML::load_file('_config.yml')
      imageNames = configYml["imageNames"]
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
        dikastes:
          image: #{imageNames["dikastes"]}
          tag: #{components["dikastes"]["version"]}
        flexvol:
          image: #{imageNames["flexvol"]}
          tag: #{components["flexvol"]["version"]}
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
      tv = Tempfile.new("temp_versions.yml")
      tv.write(versionsYml)
      tv.close

      # execute helm.
      cmd = """helm template _includes/#{version}/charts/calico \
        --set imageRegistry=#{imageRegistry} \
        --set prodname=#{configYml["prodname"]} \
        --set nodecontainer=#{configYml["nodecontainer"]} \
        -f #{tv.path} \
        -f #{t.path}"""

      if @extra_args
        cmd += " " + @extra_args
      end

      out = `#{cmd}`

      t.unlink
      tv.unlink
      return out
    end
  end
end

Liquid::Template.register_tag('helm', Jekyll::RenderHelmTagBlock)
