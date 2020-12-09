---
title: Calico Enterprise API reference
description: Calico Enterprise API reference
canonical_url: "/reference/apidocs/index"
show_read_time: false
show_toc: false
---

<link rel="stylesheet" type="text/css" href="./style.css">
<link rel="stylesheet" type="text/css" href="/css/swagger-ui/swagger-ui.css">

<div id="swagger-ui"></div>

<script src="/js/swagger-ui/swagger-ui-bundle.js" charset="UTF-8"></script>
<script src="/js/swagger-ui/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
<script>
    window.addEventListener('load', function() {
      const ui = SwaggerUIBundle({
        url: "./swagger.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
        //   SwaggerUIStandalonePreset
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl
        ],
        // layout: "StandaloneLayout"
      });

      window.swaggerUI = ui;
    });
</script>
