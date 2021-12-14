require "optparse"
require "yaml"
require_relative "../_plugins/lib"
require_relative "../_plugins/values"

usage = "ruby hack/gen_values_yaml.rb [arguments...]

It's recommended to run this from the root of the Calico repository,
as the default paths assume as much.

--config    Path to the jekyll config. [default: _config.yml]
--versions  Path to the versions.yml. [default: _data/versions.yml]
--registry  The registry prefix. [default: quay.io]
--chart     The chart to render. [default: calico]
"

# Extend the Hash class with deep_merge since the builtin 'merge' function does not merge duplicate keys in a Hash.
# Source: https://stackoverflow.com/a/30225093
class ::Hash
    def deep_merge(second)
        merger = proc { |key, v1, v2| Hash === v1 && Hash === v2 ? v1.merge(v2, &merger) : Array === v1 && Array === v2 ? v1 | v2 : [:undefined, nil, :nil].include?(v2) ? v1 : v2 }
        self.merge(second.to_h, &merger)
    end
end

OptionParser.new do |parser|
    parser.on("-c", "--config=CONFIG") do |config|
        @path_to_config = config
    end

    parser.on("-v", "--versions=VERSIONS") do |versions|
        @path_to_versions = versions
    end

    parser.on("-r", "--registry=REGISTRY") do |registry|
        @image_registry = registry
    end

    parser.on("-C", "--chart=CHART") do |chart|
        @chart = chart
    end
end.parse!

@path_to_config ||= "_config.yml"
@path_to_versions ||= "_data/versions.yml"
@image_registry ||= "quay.io/"
@chart ||= "calico"

# In order to preserve backwards compatibility with the existing template system,
# we process config.yml for imageNames and _versions.yml for tags,
# then write them in a more standard helm format.
config = YAML::load_file(@path_to_config)
imageNames = config["imageNames"]

versions_yml = YAML::load_file(@path_to_versions)
versions = parse_versions(versions_yml)

print gen_values(versions, imageNames, @image_registry, @chart, false)
