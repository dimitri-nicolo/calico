#!/bin/bash

cat > Dockerfile <<EOF
FROM docker.elastic.co/kibana/kibana:${KIBANA_VERSION}

USER root

RUN yum -y update && yum -y upgrade && yum clean all

COPY cleanup.sh /
RUN /cleanup.sh

ARG GTM_INTEGRATION=disable
COPY createKibanaConfig.sh /
RUN /createKibanaConfig.sh /usr/share/kibana/config/kibana.yml

USER kibana
EOF
