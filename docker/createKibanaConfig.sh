#!/bin/bash

KIBANA_CONFIG=$1

cat >${KIBANA_CONFIG} <<EOF
# Default Kibana configuration
server.name: kibana
server.host: "0"
elasticsearch.hosts: [ "http://elasticsearch:9200" ]
xpack.monitoring.ui.container.elasticsearch.enabled: true

# Custom Tigera configuration
csp.rules:
    - "script-src 'unsafe-eval' 'self' 'unsafe-inline' https://www.googletagmanager.com"
    - "img-src www.googletagmanager.com 'self' data:"

tigera.enabled: true
tigera.pluginEnabled: ${GTM_INTEGRATION}
tigera.container: "GTM-TCNXTCJ"
tigera.licenseEdition: ${LICENSE_EDITION: -enterpriseEdition}

EOF

# remove this script from fs
rm "$0"
