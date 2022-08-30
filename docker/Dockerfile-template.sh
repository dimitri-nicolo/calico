#!/bin/bash

cat > Dockerfile <<EOF
FROM docker.elastic.co/kibana/kibana:${KIBANA_VERSION}

USER root

RUN apt-get -y update && apt-get -y upgrade && apt-get clean

ARG GTM_INTEGRATION=disable

COPY createKibanaConfig.sh /
RUN /createKibanaConfig.sh /usr/share/kibana/config/kibana.yml

COPY gtmSetup.sh /
RUN /gtmSetup.sh

USER kibana
EOF

