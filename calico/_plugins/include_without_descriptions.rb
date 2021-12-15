require 'liquid'
require 'yaml'

module Jekyll
  class IncludeWithoutDescriptions < Liquid::Tag

    # Create a tag in the docs with as input a valid yaml file. The result will be the yaml,
    # with all the 'description' keys removed from the structure recursively.
    # Example "{% include_without_descriptions _includes/charts/file.yaml %}"
    def initialize(tag_name, file, tokens)
      super
      @file = file
    end

 	def render(context)
      hash = YAML.load_file(@file.strip!)
      replace_description(hash)
      YAML.dump(hash)
    end

    # Iteratively remove description fields.
    def replace_description(hash)
      hash.delete_if{|k, v| k == "description" and v.class == String }
      hash.each do |k, v|
        if v.class == Hash
          hash[k] = replace_description(v)
        elsif v.class == Array
          v.each_with_index do |x, i|
            if x.class == Hash
              v[i] = replace_description(x)
            end
          end
        end
      end
    end
  end
end

Liquid::Template.register_tag('include_without_descriptions', Jekyll::IncludeWithoutDescriptions)
