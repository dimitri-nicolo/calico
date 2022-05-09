# Support matrix hook does two things:
# 1. checks that the support matrix is only using values from support_matrix_list defined in _config.yml
# 2. Expands special value "all" to include all the platforms defined in support_matrix_list in _config.yml
Jekyll::Hooks.register :site, :pre_render do |site|
  support_matrix_list = site.config["support_matrix_list"]
  site.data["support_matrix"].each do |feature_name, platforms|
    platforms["platforms"].each do |platform|
      # If we find special string "all", then replace with all platform names specified in the support_matrix_list
      if platform.downcase == "all"
        site.data["support_matrix"][feature_name]["platforms"] = support_matrix_list
        next
      end
      if not support_matrix_list.include? platform
            raise "\n Unexpected platform '#{platform}' specified for feature '#{feature_name}' in the support matrix"
      end
    end
  end
end
