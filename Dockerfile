FROM fluent/fluentd:v1.2-onbuild
MAINTAINER spike@tigera.io

RUN apk add --update --virtual .build-deps \
        sudo build-base ruby-dev \
 && sudo gem install \
        fluent-plugin-elasticsearch fluent-plugin-s3 fluent-plugin-splunk-hec fluent-plugin-sumologic_output \
 && sudo fluent-gem install fluent-plugin-remote_syslog \
 && sudo gem sources --clear-all \
 && apk del .build-deps \
 && rm -rf /var/cache/apk/* \
           /home/fluent/.gem/ruby/2.3.0/cache/*.gem
RUN apk add --no-cache curl
RUN apk add --no-cache jq

ADD elastic_mapping_flows.template /fluentd/etc/elastic_mapping_flows.template
ADD elastic_mapping_audits.template /fluentd/etc/elastic_mapping_audits.template
COPY fluent_sources.conf /fluentd/etc/fluent_sources.conf
COPY fluent_transforms.conf /fluentd/etc/fluent_transforms.conf
COPY fluent_output.conf /fluentd/etc/fluent_output.conf
COPY outputs /fluentd/etc/outputs

ENV FLOW_LOG_FILE=/var/log/calico/flowlogs/flows.log
ENV POS_DIR=/var/log/calico
ENV ELASTIC_HOST=elasticsearch
ENV ELASTIC_PORT=9200
ENV ELASTIC_FLUSH_INTERVAL=5s
ENV KUBE_AUDIT_LOG=/var/log/calico/audit/kube-audit.log
ENV KUBE_AUDIT_POS=${POS_DIR}/kube-audit.log.pos
ENV ELASTIC_INDEX_SUFFIX=cluster
ENV S3_FLUSH_INTERVAL=5s
ENV S3_STORAGE=false
ENV ELASTIC_FLOWS_INDEX_SHARDS=5

COPY readiness.sh /bin/
RUN chmod +x /bin/readiness.sh

COPY liveness.sh /bin/
RUN chmod +x /bin/liveness.sh

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

RUN mkdir /fluentd/etc/output_flows
RUN mkdir /fluentd/etc/output_tsee_audit
RUN mkdir /fluentd/etc/output_kube_audit

EXPOSE 24284

CMD exec /bin/ee_entrypoint.sh fluentd -c /fluentd/etc/${FLUENTD_CONF} -p /fluentd/plugins $FLUENTD_OPT
