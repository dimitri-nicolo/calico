FROM fluent/fluentd:v1.2-onbuild
MAINTAINER YOUR_NAME spike@tigera.io

RUN apk add --update --virtual .build-deps \
        sudo build-base ruby-dev \
 && sudo gem install \
        fluent-plugin-elasticsearch \
 && sudo gem sources --clear-all \
 && apk del .build-deps \
 && rm -rf /var/cache/apk/* \
           /home/fluent/.gem/ruby/2.3.0/cache/*.gem

ADD elastic_mapping_flows.template /fluentd/etc/elastic_mapping_flows.template

ENV FLOW_LOG_FILE=/var/log/calico/flowlogs/flows.log
ENV ELASTIC_HOST=elasticsearch
ENV ELASTIC_PORT=9200

EXPOSE 24284
