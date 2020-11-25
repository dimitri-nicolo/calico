---
title: API Docs
description: API Docs
canonical_url: "/reference/apidocs/index"
show_read_time: false
show_toc: false
---

<link rel="stylesheet" type="text/css" href="./reset.css">
<link rel="stylesheet" type="text/css" href="./swagger-ui.css">

<div id="swagger-ui"></div>

<script src="./swagger-ui-bundle.js" charset="UTF-8"></script>
<script src="./swagger-ui-standalone-preset.js" charset="UTF-8"></script>
<script>
    window.addEventListener('load', function() {
      const ui = SwaggerUIBundle({
        url: "https://petstore.swagger.io/v2/swagger.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "StandaloneLayout"
      });

      window.swaggerUI = ui;
    });
</script>
