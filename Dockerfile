FROM fluent/fluentd:v1.9.3-1.0
MAINTAINER spike@tigera.io

USER root

RUN apk add --update --virtual .build-deps \
        build-base=0.5-r1 ruby-dev=2.5.8-r0 \
 && gem install \
        fluent-plugin-elasticsearch:4.0.5 fluent-plugin-s3:1.3.0 \
        fluent-plugin-splunk-hec:1.1.2 fluent-plugin-sumologic_output:1.6.1 \
        fluent-plugin-cloudwatch-logs:0.8.0 \
 && fluent-gem install fluent-plugin-remote_syslog:1.0.0 \
 && gem sources --clear-all \
 && apk del .build-deps \
 && rm -rf /var/cache/apk/* \
           /home/fluent/.gem/ruby/2.3.0/cache/*.gem
RUN apk add --no-cache curl=7.64.0-r3 jq=1.6-r0
RUN apk add --no-cache ca-certificates && update-ca-certificates

ADD elastic_mapping_flows.template /fluentd/etc/elastic_mapping_flows.template
ADD elastic_mapping_dns.template /fluentd/etc/elastic_mapping_dns.template
ADD elastic_mapping_audits.template /fluentd/etc/elastic_mapping_audits.template
ADD elastic_mapping_bgp.template /fluentd/etc/elastic_mapping_bgp.template
COPY fluent_sources.conf /fluentd/etc/fluent_sources.conf
COPY fluent_transforms.conf /fluentd/etc/fluent_transforms.conf
COPY fluent_output.conf /fluentd/etc/fluent_output.conf
COPY fluent_output_optional.conf /fluentd/etc/fluent_output_optional.conf
COPY outputs /fluentd/etc/outputs
COPY inputs /fluentd/etc/inputs
COPY filters /fluentd/etc/filters

# Compliance reports logs needs a regex pattern because there will be 
# multiple logs (one per report type), e.g. compliance.network-access.reports.log
ENV COMPLIANCE_LOG_FILE=/var/log/calico/compliance/compliance.*.reports.log
ENV FLOW_LOG_FILE=/var/log/calico/flowlogs/flows.log
ENV DNS_LOG_FILE=/var/log/calico/dnslogs/dns.log
ENV BIRD_LOG_FILE=/var/log/calico/bird/current
ENV BIRD6_LOG_FILE=/var/log/calico/bird6/current
ENV POS_DIR=/var/log/calico
ENV ELASTIC_HOST=elasticsearch
ENV ELASTIC_PORT=9200
ENV ELASTIC_FLUSH_INTERVAL=5s
ENV KUBE_AUDIT_LOG=/var/log/calico/audit/kube-audit.log
ENV KUBE_AUDIT_POS=${POS_DIR}/kube-audit.log.pos
ENV ELASTIC_INDEX_SUFFIX=cluster
ENV S3_FLUSH_INTERVAL=5s
ENV S3_STORAGE=false
ENV ELASTIC_FLOWS_INDEX_SHARDS=1
ENV ELASTIC_FLOWS_INDEX_REPLICAS=0
ENV ELASTIC_DNS_INDEX_SHARDS=1
ENV ELASTIC_DNS_INDEX_REPLICAS=0
ENV ELASTIC_AUDIT_INDEX_REPLICAS=0
ENV ELASTIC_TEMPLATE_OVERWRITE=true
ENV SYSLOG_PACKET_SIZE=1024

COPY readiness.sh /bin/
RUN chmod +x /bin/readiness.sh

COPY liveness.sh /bin/
RUN chmod +x /bin/liveness.sh

COPY syslog-environment.sh /bin/
COPY syslog-config.sh /bin/
RUN chmod +x /bin/syslog-config.sh /bin/syslog-environment.sh

COPY splunk-environment.sh /bin/
RUN chmod +x /bin/splunk-environment.sh

COPY splunk-config.sh /bin/
RUN chmod +x /bin/splunk-config.sh

COPY sumo-environment.sh /bin/
RUN chmod +x /bin/sumo-environment.sh

COPY sumo-config.sh /bin/
RUN chmod +x /bin/sumo-config.sh

COPY ee_entrypoint.sh /bin/
RUN chmod +x /bin/ee_entrypoint.sh

COPY eks/bin/eks-log-forwarder-startup /bin/

RUN mkdir /fluentd/etc/output_flows
RUN mkdir /fluentd/etc/output_dns
RUN mkdir /fluentd/etc/output_tsee_audit
RUN mkdir /fluentd/etc/output_kube_audit
RUN mkdir /fluentd/etc/output_compliance_reports
RUN mkdir /fluentd/etc/output_bgp

EXPOSE 24284

ENTRYPOINT []
CMD /bin/ee_entrypoint.sh fluentd -c /fluentd/etc/${FLUENTD_CONF} -p /fluentd/plugins $FLUENTD_OPT
