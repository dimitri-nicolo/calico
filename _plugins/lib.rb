Component = Struct.new(:image, :version, :registry) do
end

# Takes versions_yml which is structured as follows:
#
#   ["components"=>
#      {"calico/node"=>{"version"=>"v3.6.0"},
#       "typha"=>{"version"=>"v3.6.0"}}]
#
# If the component also specifies an 'image', the component will be returned as a Component class, e.g.
#
#  {"calico/node" => Component(image: "calico/node", version: "v3.6.0"),
#  "typha" => Component(image: "typha", version: "v3.6.0")}
#
# Otherwise, it the value will be a string, e.g:
#
# {"calico/node"=>"v3.6.0",
#   "typha"=>"v3.6.0"}
def parse_versions(versions_yml)
  components = versions_yml[0]["components"].clone
  versionsYml = components.each do |key,val|
    if val.include? "image"
      # if the "image" key is present, then imageNames should be pulled from versions.yml
      components[key] = Component.new(val["image"], val["version"], val["registry"])
    else
      components[key] = val["version"]
    end
  end

  unless versions_yml[0]["tigera-operator"].nil?
          operator = versions_yml[0]["tigera-operator"]
          versionsYml["tigera-operator"] = Component.new(operator["image"], operator["version"], operator["registry"])
  end
  return versionsYml
end
