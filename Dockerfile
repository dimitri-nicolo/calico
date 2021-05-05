# Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM fluent/fluentd:v1.11.1-1.0
MAINTAINER spike@tigera.io

# Need to define root user explicitly (for remaining setup) and be numeric for k8s validation
USER 0

RUN apk add --update --virtual .build-deps \
        build-base=0.5-r1 ruby-dev=2.5.8-r0 \
 && gem install \
        fluent-plugin-elasticsearch:4.2.2 fluent-plugin-s3:1.3.0 \
        fluent-plugin-splunk-hec:1.1.2 fluent-plugin-sumologic_output:1.6.1 \
        fluent-plugin-cloudwatch-logs:0.8.0 \
        elasticsearch-xpack:7.6.0 \
        fluent-plugin-prometheus:2.0.0 \
 && fluent-gem install fluent-plugin-remote_syslog:1.0.0 \
 && gem sources --clear-all \
 && apk del .build-deps \
 && rm -rf /var/cache/apk/* \
           /home/fluent/.gem/ruby/2.3.0/cache/*.gem
RUN apk add --no-cache curl=7.64.0-r5 jq=1.6-r0
RUN apk add --no-cache ca-certificates && update-ca-certificates
RUN apk update && apk upgrade libcrypto1

ADD elastic_mapping_flows.template /fluentd/etc/elastic_mapping_flows.template
ADD elastic_mapping_dns.template /fluentd/etc/elastic_mapping_dns.template
ADD elastic_mapping_audits.template /fluentd/etc/elastic_mapping_audits.template
ADD elastic_mapping_bgp.template /fluentd/etc/elastic_mapping_bgp.template
ADD elastic_mapping_l7.template /fluentd/etc/elastic_mapping_l7.template
COPY fluent_sources.conf /fluentd/etc/fluent_sources.conf
COPY fluent_transforms.conf /fluentd/etc/fluent_transforms.conf
COPY output_match /fluentd/etc/output_match
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
ENV IDS_EVENT_LOG_FILE=/var/log/calico/ids/events.log
ENV L7_LOG_FILE=/var/log/calico/l7logs/l7.log
ENV EE_AUDIT_LOG_FILE=/var/log/calico/audit/tsee-audit.log

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
ENV ELASTIC_L7_INDEX_SHARDS=1
ENV ELASTIC_L7_INDEX_REPLICAS=0
ENV ELASTIC_TEMPLATE_OVERWRITE=true
ENV ELASTIC_BGP_INDEX_SHARDS=1
ENV ELASTIC_BGP_INDEX_REPLICAS=0

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
RUN mkdir /fluentd/etc/output_ids_events
RUN mkdir /fluentd/etc/output_l7

EXPOSE 24284

ENTRYPOINT []
CMD /bin/ee_entrypoint.sh fluentd -c /fluentd/etc/${FLUENTD_CONF} -p /fluentd/plugins $FLUENTD_OPT
