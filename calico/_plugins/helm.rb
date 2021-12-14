require "jekyll"
require "tempfile"
require "open3"

require_relative "./lib"

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

      @chart = "calico"
      if extra_args.start_with?("tigera-secure-ee")
        @chart = "tigera-secure-ee"
        extra_args.slice! "tigera-secure-ee"
      end
      if extra_args.start_with?("tigera-operator")
        @chart = "tigera-operator"
        extra_args.slice! "tigera-operator"
      end
      if extra_args.start_with?("tigera-prometheus-operator")
        @chart = "tigera-prometheus-operator"
        extra_args.slice! "tigera-prometheus-operator"
      end

      @mock_es = false
      if extra_args.start_with?(" secure-es")
        @mock_es = true
        extra_args.slice! " secure-es"
      end

      # helm doesn't natively have an --execute-dir flag but it sure would be useful if it did.
      # here we replace instances of "--execute-dir $dir" with individual calls to "--execute $file" by
      # iterating over files in that directory.
      extra_args.gsub!(/--execute-dir (\S*)/) do |_|
        e = []
        Dir.foreach "_includes/charts/#{@chart}/#{$1}" do |file|
            next if File.directory?("_includes/charts/#{@chart}/#{$1}/#{file}")
            e << "--execute #{$1}/#{file}"
        end
        e.join(" ")
      end

      @extra_args = extra_args
    end

    def render(context)
      text = super

      # Because helm hasn't merged stdin support, write the passed-in values.yaml
      # to a tempfile on disk.
      t = Tempfile.new("jhelm")
      t.write(text)
      t.close

      imageRegistry = context.registers[:page]["registry"]
      imageNames = context.registers[:site].config["imageNames"]
      versions = context.registers[:site].data["versions"]


      if context.registers[:site].data["versions"][0].has_key?("dockerRepo")
          imageRegistry = context.registers[:site].data["versions"][0]["dockerRepo"]
        unless imageRegistry.end_with? "/"
          imageRegistry = imageRegistry << "/"
        end
      end

      vs = parse_versions(versions)

      versionsYml = gen_values(vs, imageNames, imageRegistry, @chart, true)

      tv = Tempfile.new("temp_versions.yml")
      tv.write(versionsYml)
      tv.close

      # Execute helm.
      # Set the default etcd endpoint placeholder for rendering in the docs.
      cmd = """helm template _includes/charts/#{@chart} \
        -f #{tv.path} \
        -f #{t.path} \
        --set node.resources.requests.cpu='250m' \
        --set manager.service.type=NodePort \
        --set manager.service.nodePort=30003 \
        --set alertmanager.service.type=NodePort \
        --set alertmanager.service.nodePort=30903 \
        --set prometheus.scrapeTargets.node.service.type=NodePort \
        --set prometheus.scrapeTargets.node.service.nodePort=30909 \
        --set kibana.service.type=NodePort \
        --set kibana.service.nodePort=30601 \
        --set etcd.endpoints='http://<ETCD_IP>:<ETCD_PORT>'"""

      if @chart == "calico" or @chart == "tigera-secure-ee"
        # static rendered manifests for the 'tigera-secure-ee-core' (calico) and
        # 'tigera-secure-ee' chart should configure components to use an imagePullSecret
        # named 'cnx-pull-secret'.
        cmd +=  " --set imagePullSecrets.cnx-pull-secret=''"
      end

      # Add mock elasticsearch settings if required for rendering in the docs.
      if @mock_es
        cmd = mock_elastic_settings(cmd)
      end

      cmd += " " + @extra_args.to_s

      out, stderr, status = Open3.capture3(cmd)
      if status != 0
        raise "failed to execute helm for '#{context.registers[:page]["path"]}': #{stderr}"
      end

      t.unlink
      tv.unlink
      return out
    end
    def mock_elastic_settings(cmd)
      cmd += " " + """--set elasticsearch.host='__ELASTICSEARCH_HOST__' \
        --set elasticsearch.tls.ca=fake \
        --set elasticsearch.fluentd.password=fake \
        --set elasticsearch.manager.password=fake \
        --set elasticsearch.curator.password=fake \
        --set elasticsearch.compliance.benchmarker.password=fake \
        --set elasticsearch.compliance.controller.password=fake \
        --set elasticsearch.compliance.reporter.password=fake \
        --set elasticsearch.compliance.snapshotter.password=fake \
        --set elasticsearch.compliance.server.password=fake \
        --set elasticsearch.intrusionDetection.password=fake \
        --set elasticsearch.elasticInstaller.password=fake"""
      return cmd
    end
  end
end

Liquid::Template.register_tag('helm', Jekyll::RenderHelmTagBlock)
