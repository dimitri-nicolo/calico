require "jekyll"
require "tempfile"
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

      if extra_args.start_with?("tigera-secure-lma")
        @chart = "tigera-secure-lma"
        extra_args.slice! "tigera-secure-lma"
      else
        @chart = "calico"
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

      version = context.registers[:page]["version"]
      imageRegistry = context.registers[:page]["registry"]
      imageNames = context.registers[:site].config["imageNames"]
      versions = context.registers[:site].data["versions"]

      vs = parse_versions(versions, version)
      versionsYml = gen_values(vs, imageNames, imageRegistry)

      tv = Tempfile.new("temp_versions.yml")
      tv.write(versionsYml)
      tv.close

      # execute helm.
      cmd = """helm template _includes/#{version}/charts/#{@chart} \
        -f #{tv.path} \
        -f #{t.path}"""

      cmd += " " + @extra_args.to_s

      out = `#{cmd}`

      t.unlink
      tv.unlink
      return out
    end
  end
end

Liquid::Template.register_tag('helm', Jekyll::RenderHelmTagBlock)
