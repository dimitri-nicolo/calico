#!/bin/bash

KIBANA_CONFIG=$1

cat >${KIBANA_CONFIG} <<EOF
# Default Kibana configuration
server.name: kibana
server.host: "0"
elasticsearch.hosts: [ "http://elasticsearch:9200" ]
xpack.monitoring.ui.container.elasticsearch.enabled: true
EOF

cat >>${KIBANA_CONFIG} <<EOF
# Custom Tigera configuration
tigera.enabled: true
tigera.licenseEdition: ${LICENSE_EDITION:-enterpriseEdition}
EOF

if [[ "$GTM_INTEGRATION" == 'enabled' ]]; then
  cat >>${KIBANA_CONFIG} <<EOF
# Google Tag Manager configuration
csp:
  script_src:
    - https://www.googletagmanager.com
  img_src:
    - https://www.googletagmanager.com

google_tag_manager.enabled: true
google_tag_manager.container: "GTM-TCNXTCJ"
EOF
fi
