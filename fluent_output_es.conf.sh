#!/bin/sh

OUTPUT_FLOW_ES=$(cat <<EOM
    @type elasticsearch
    host "#{ENV['ELASTIC_HOST']}"
    port "#{ENV['ELASTIC_PORT']}"
    scheme "https"
    user "#{ENV['ELASTIC_USER']}"
    password "#{ENV['ELASTIC_PASSWORD']}"
    ca_file /etc/fluentd/elastic/ca.pem
    ssl_verify "#{ENV['ELASTIC_SSL_VERIFY']}"
    index_name "tigera_secure_ee_flows.#{ENV['ELASTIC_INDEX_SUFFIX']}.%Y%m%d"
    template_file /fluentd/etc/elastic_mapping_flows.template
    template_name tigera_secure_ee_flows
    <buffer tag, time>
      timekey 1d
      flush_mode interval
      flush_interval "#{ENV['ELASTIC_FLUSH_INTERVAL']}"
    </buffer>
EOM
)

OUTPUT_AUDIT_TSEE_ES=$(cat <<EOM
    @type elasticsearch
    host "#{ENV['ELASTIC_HOST']}"
    port "#{ENV['ELASTIC_PORT']}"
    scheme "https"
    user "#{ENV['ELASTIC_USER']}"
    password "#{ENV['ELASTIC_PASSWORD']}"
    ca_file /etc/fluentd/elastic/ca.pem
    ssl_verify "#{ENV['ELASTIC_SSL_VERIFY']}"
    index_name "tigera_secure_ee_audit_ee.#{ENV['ELASTIC_INDEX_SUFFIX']}.%Y%m%d"
    <buffer tag, time>
      timekey 1d
      flush_mode interval
      flush_interval "#{ENV['ELASTIC_FLUSH_INTERVAL']}"
    </buffer>
EOM
)

OUTPUT_AUDIT_KUBE_ES=$(cat <<EOM
    @type elasticsearch
    host "#{ENV['ELASTIC_HOST']}"
    port "#{ENV['ELASTIC_PORT']}"
    scheme "https"
    user "#{ENV['ELASTIC_USER']}"
    password "#{ENV['ELASTIC_PASSWORD']}"
    ca_file /etc/fluentd/elastic/ca.pem
    ssl_verify "#{ENV['ELASTIC_SSL_VERIFY']}"
    index_name "tigera_secure_ee_audit_kube.#{ENV['ELASTIC_INDEX_SUFFIX']}.%Y%m%d"
    <buffer tag, time>
      timekey 1d
      flush_mode interval
      flush_interval "#{ENV['ELASTIC_FLUSH_INTERVAL']}"
    </buffer>
EOM
)
