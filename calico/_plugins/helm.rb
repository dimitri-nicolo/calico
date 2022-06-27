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
        all_files = Dir.entries "_includes/charts/#{@chart}/#{$1}"
        all_files.sort.each do |file|
          fpath = File.join($1, file)
          next if File.directory?("_includes/charts/#{@chart}/#{fpath}")

          # for helm v3, when templating crd files, you must specify them relative
          # to the crd directory. so trim the 'crds' from the name.
          # we don't need to worry about the helm v2 case because crds are stored in templates/
          # and can't be --executed from the crds/ directory.
          # if fpath.start_with? "template/crds" then
          #   fpath = Pathname.new(fpath).relative_path_from(Pathname.new("template/crds"))
          # end
          if fpath.start_with? "crds" then
            fpath = Pathname.new(fpath).relative_path_from(Pathname.new("crds"))
          end

            e << "--execute #{fpath}"
        end
        e.join(" ")
      end

      # substitute --execute with --show-only for helm v3 compatibility.
      if @chart == "tigera-operator" or @chart == "tigera-prometheus-operator" then
        extra_args.gsub!(/--execute (\S*)/) do |f|
          # operator CRDs have moved to root
          if $1.start_with? "crds/" then f.sub('--execute crds/', '--show-only ')
          # all other requests need to use --show-only instead of --execute for helm v3
          else f.sub('--execute', '--show-only')
          end
        end
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
      # with additional arguments for EE componenets
      cmd =  """bin/helm3 template --include-crds _includes/charts/#{@chart} \
        -f #{tv.path} \
        -f #{t.path} \
        -n tigera-operator \
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

      if @chart == "calico" 
        # static rendered manifests for the 'calico and 'tigera-secure-ee' 
        # chart should configure components to use an imagePullSecret
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
