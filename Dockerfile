# Copyright (c) 2018-2024 Tigera, Inc. All rights reserved.
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

ARG QEMU_IMAGE

FROM ${QEMU_IMAGE} as qemu

FROM registry.access.redhat.com/ubi9/ubi as builder

COPY --from=qemu /usr/bin/qemu-*-static /usr/bin/

RUN dnf upgrade -y && dnf install -y \
       gcc \
       jq \
       ruby \
       ruby-devel

# From upstream fluentd Dockerfile and update them when upgrading fluentd
# https://github.com/fluent/fluentd-docker-image/blob/master/v1.16/debian/Dockerfile
RUN gem install async-http -v 0.60.2 && \
       gem install async -v 1.31.0 && \
       gem install fluentd -v 1.16.3 && \
       gem install json -v 2.6.3 && \
       gem install oj -v 3.16.1 && \
       gem install rexml -v 3.2.6 && \
       gem install uri -v 0.12.2

# Calico Enterprise dependencies
RUN gem install elasticsearch-api -v 7.17.10 && \
       gem install elasticsearch-transport -v 7.17.10 && \
       gem install elasticsearch-xpack -v 7.17.10 && \
       gem install elasticsearch -v 7.17.10 && \
       gem install fluent-plugin-cloudwatch-logs -v 0.14.3 && \
       gem install fluent-plugin-elasticsearch -v 5.4.3 && \
       gem install fluent-plugin-prometheus -v 2.1.0 && \
       gem install fluent-plugin-remote_syslog -v 1.1.0 && \
       gem install fluent-plugin-s3 -v 1.7.2 && \
       gem install fluent-plugin-splunk-hec -v 1.3.3 && \
       gem install fluent-plugin-sumologic_output -v 1.8.0

# cleanup
RUN gem sources --clear-all && rm -fr /tmp/*

RUN mkdir -p /fluentd/etc/output_bgp
RUN mkdir -p /fluentd/etc/output_compliance_reports
RUN mkdir -p /fluentd/etc/output_dns
RUN mkdir -p /fluentd/etc/output_flows
RUN mkdir -p /fluentd/etc/output_ids_events
RUN mkdir -p /fluentd/etc/output_kube_audit
RUN mkdir -p /fluentd/etc/output_l7
RUN mkdir -p /fluentd/etc/output_runtime
RUN mkdir -p /fluentd/etc/output_tsee_audit
RUN mkdir -p /fluentd/etc/output_waf
RUN mkdir -p /fluentd/log

FROM scratch as source

ARG TARGETARCH

# Dependent binaries and shared libraries
COPY --from=builder /bin/sh /bin/sh
COPY --from=builder /usr/bin/awk /usr/bin/awk
COPY --from=builder /usr/bin/cat /usr/bin/cat
COPY --from=builder /usr/bin/coreutils /usr/bin/coreutils
COPY --from=builder /usr/bin/cp /usr/bin/cp
COPY --from=builder /usr/bin/curl /usr/bin/curl
COPY --from=builder /usr/bin/echo /usr/bin/echo
COPY --from=builder /usr/bin/find /usr/bin/find
COPY --from=builder /usr/bin/jq /usr/bin/jq
COPY --from=builder /usr/bin/ls /usr/bin/ls
COPY --from=builder /usr/bin/rm /usr/bin/rm
COPY --from=builder /usr/bin/sed /usr/bin/sed
COPY --from=builder /usr/bin/sort /usr/bin/sort
COPY --from=builder /usr/bin/tar /usr/bin/tar
COPY --from=builder /usr/bin/test /usr/bin/test
COPY --from=builder /usr/bin/which /usr/bin/which

# arm64 loader under /lib/ld-linux-aarch64.so.1
COPY --from=builder /lib/ld-linux-*.so.? /lib/
# amd64 loader under /lib64/ld-linux-x86-64.so.2
COPY --from=builder /lib64/ld-linux-*.so.? /lib64/
COPY --from=builder /lib64/libacl.so.1 /lib64/libacl.so.1
COPY --from=builder /lib64/libattr.so.1 /lib64/libattr.so.1
COPY --from=builder /lib64/libc.so.6 /lib64/libc.so.6
COPY --from=builder /lib64/libcap.so.2 /lib64/libcap.so.2
COPY --from=builder /lib64/libcom_err.so.2 /lib64/libcom_err.so.2
COPY --from=builder /lib64/libcrypto.so.3 /lib64/libcrypto.so.3
COPY --from=builder /lib64/libcurl.so.4 /lib64/libcurl.so.4
COPY --from=builder /lib64/libgmp.so.10 /lib64/libgmp.so.10 
COPY --from=builder /lib64/libgssapi_krb5.so.2 /lib64/libgssapi_krb5.so.2
COPY --from=builder /lib64/libjq.so.1 /lib64/libjq.so.1
COPY --from=builder /lib64/libk5crypto.so.3 /lib64/libk5crypto.so.3
COPY --from=builder /lib64/libkeyutils.so.1 /lib64/libkeyutils.so.1
COPY --from=builder /lib64/libkrb5.so.3 /lib64/libkrb5.so.3
COPY --from=builder /lib64/libkrb5support.so.0 /lib64/libkrb5support.so.0
COPY --from=builder /lib64/libm.so.6 /lib64/libm.so.6
COPY --from=builder /lib64/libmpfr.so.6 /lib64/libmpfr.so.6
COPY --from=builder /lib64/libnghttp2.so.14 /lib64/libnghttp2.so.14
COPY --from=builder /lib64/libonig.so.5 /lib64/libonig.so.5
COPY --from=builder /lib64/libpcre2-8.so.0 /lib64/libpcre2-8.so.0
COPY --from=builder /lib64/libreadline.so.8 /lib64/libreadline.so.8
COPY --from=builder /lib64/libresolv.so.2 /lib64/libresolv.so.2
COPY --from=builder /lib64/libselinux.so.1 /lib64/libselinux.so.1
COPY --from=builder /lib64/libsigsegv.so.2 /lib64/libsigsegv.so.2
COPY --from=builder /lib64/libssl.so.3 /lib64/libssl.so.3
COPY --from=builder /lib64/libtinfo.so.6 /lib64/libtinfo.so.6

# glibc NSS plugins and config files.
COPY --from=builder /lib64/libnss_dns.so.2 /lib64/libnss_dns.so.2
COPY --from=builder /lib64/libnss_files.so.2 /lib64/libnss_files.so.2

COPY --from=builder /etc/host.conf /etc/host.conf
COPY --from=builder /etc/hosts /etc/hosts
COPY --from=builder /etc/networks /etc/networks
COPY --from=builder /etc/nsswitch.conf /etc/nsswitch.conf

# ruby and fluentd
COPY --from=builder /usr/bin/gem /usr/bin/gem
COPY --from=builder /usr/bin/ruby /usr/bin/ruby
COPY --from=builder /usr/lib64/gems/ /usr/lib64/gems/
COPY --from=builder /usr/local/lib64/gems/ /usr/local/lib64/gems/
COPY --from=builder /usr/local/share/gems/ /usr/local/share/gems/
COPY --from=builder /usr/share/gems/ /usr/share/gems/
COPY --from=builder /usr/share/ruby/ /usr/share/ruby/
COPY --from=builder /usr/share/rubygems/ /usr/share/rubygems/

COPY --from=builder /lib64/libcrypt.so.2 /lib64/libcrypt.so.2
COPY --from=builder /lib64/libffi.so.8 /lib64/libffi.so.8
COPY --from=builder /lib64/libgdbm.so.6 /lib64/libgdbm.so.6
COPY --from=builder /lib64/libgdbm_compat.so.4 /lib64/libgdbm_compat.so.4
COPY --from=builder /lib64/libruby.so.3.0 /lib64/libruby.so.3.0
COPY --from=builder /lib64/libyaml-0.so.2 /lib64/libyaml-0.so.2
COPY --from=builder /lib64/libz.so.1 /lib64/libz.so.1
COPY --from=builder /usr/lib64/ruby/ /usr/lib64/ruby/

COPY --from=builder /fluentd/ /fluentd/
COPY --from=builder /usr/local/bin/fluent-gem /usr/bin/fluent-gem
COPY --from=builder /usr/local/bin/fluentd /usr/bin/fluentd

COPY --from=builder /tmp/ /tmp/

# Copy scripts needed by fluentd
COPY ee_entrypoint.sh /bin/ee_entrypoint.sh
COPY liveness.sh /bin/liveness.sh
COPY readiness.sh /bin/readiness.sh
COPY splunk-config.sh /bin/splunk-config.sh
COPY splunk-environment.sh /bin/splunk-environment.sh
COPY sumo-config.sh /bin//sumo-config.sh
COPY sumo-environment.sh /bin/sumo-environment.sh
COPY syslog-config.sh /bin/syslog-config.sh
COPY syslog-environment.sh /bin/syslog-environment.sh

COPY eks/bin/eks-log-forwarder-startup-${TARGETARCH} /bin/eks-log-forwarder-startup

# Copy scripts and configuration files for fluentd
COPY elastic_mapping_audits.template /fluentd/etc/elastic_mapping_audits.template
COPY elastic_mapping_bgp.template /fluentd/etc/elastic_mapping_bgp.template
COPY elastic_mapping_dns.template /fluentd/etc/elastic_mapping_dns.template
COPY elastic_mapping_flows.template /fluentd/etc/elastic_mapping_flows.template
COPY elastic_mapping_l7.template /fluentd/etc/elastic_mapping_l7.template
COPY elastic_mapping_runtime.template /fluentd/etc/elastic_mapping_runtime.template
COPY elastic_mapping_waf.template /fluentd/etc/elastic_mapping_waf.template
COPY filters/ /fluentd/etc/filters/
COPY fluent_sources.conf /fluentd/etc/fluent_sources.conf
COPY fluent_transforms.conf /fluentd/etc/fluent_transforms.conf
COPY inputs/ /fluentd/etc/inputs/
COPY output_match/ /fluentd/etc/output_match/
COPY outputs/ /fluentd/etc/outputs/
COPY rubyplugin/ /fluentd/plugins/

FROM scratch

ENV PATH=$PATH:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Compliance reports logs needs a regex pattern because there will be
# multiple logs (one per report type), e.g. compliance.network-access.reports.log
ENV BIRD6_LOG_FILE=/var/log/calico/bird6/current
ENV BIRD_LOG_FILE=/var/log/calico/bird/current
ENV COMPLIANCE_LOG_FILE=/var/log/calico/compliance/compliance.*.reports.log
ENV DNS_LOG_FILE=/var/log/calico/dnslogs/dns.log
ENV EE_AUDIT_LOG_FILE=/var/log/calico/audit/tsee-audit.log
ENV FLOW_LOG_FILE=/var/log/calico/flowlogs/flows.log
ENV IDS_EVENT_LOG_FILE=/var/log/calico/ids/events.log
ENV L7_LOG_FILE=/var/log/calico/l7logs/l7.log
ENV RUNTIME_LOG_FILE=/var/log/calico/runtime-security/report.log
ENV WAF_LOG_FILE=/var/log/calico/waf/waf.log

# TLS Settings
ENV CA_CRT_PATH=/etc/pki/tigera/tigera-ca-bundle.crt
ENV TLS_CRT_PATH=/tls/tls.crt
ENV TLS_KEY_PATH=/tls/tls.key

ENV POS_DIR=/var/log/calico

ENV ELASTIC_FLUSH_INTERVAL=5s
ENV ELASTIC_HOST=elasticsearch
ENV ELASTIC_PORT=9200

ENV KUBE_AUDIT_LOG=/var/log/calico/audit/kube-audit.log
ENV KUBE_AUDIT_POS=${POS_DIR}/kube-audit.log.pos

ENV S3_FLUSH_INTERVAL=5s
ENV S3_STORAGE=false

ENV ELASTIC_INDEX_SUFFIX=cluster

ENV ELASTIC_AUDIT_INDEX_REPLICAS=0
ENV ELASTIC_BGP_INDEX_REPLICAS=0
ENV ELASTIC_BGP_INDEX_SHARDS=1
ENV ELASTIC_DNS_INDEX_REPLICAS=0
ENV ELASTIC_DNS_INDEX_SHARDS=1
ENV ELASTIC_FLOWS_INDEX_REPLICAS=0
ENV ELASTIC_FLOWS_INDEX_SHARDS=1
ENV ELASTIC_L7_INDEX_REPLICAS=0
ENV ELASTIC_L7_INDEX_SHARDS=1
ENV ELASTIC_RUNTIME_INDEX_REPLICAS=0
ENV ELASTIC_RUNTIME_INDEX_SHARDS=1
ENV ELASTIC_TEMPLATE_OVERWRITE=true
ENV ELASTIC_WAF_INDEX_REPLICAS=0
ENV ELASTIC_WAF_INDEX_SHARDS=1

ENV SYSLOG_PACKET_SIZE=1024

# Linseed default params
ENV LINSEED_CA_PATH=/etc/flu/ca.pem
ENV LINSEED_CERT_PATH=/etc/flu/crt.pem
ENV LINSEED_ENABLED=false
ENV LINSEED_ENDPOINT=ENDPOINT
ENV LINSEED_FLUSH_INTERVAL=5s
ENV LINSEED_KEY_PATH=/etc/flu/key.pem
ENV LINSEED_TOKEN=/var/run/secrets/kubernetes.io/serviceaccount/token

COPY --from=source / /

EXPOSE 24284

ENTRYPOINT ["/bin/ee_entrypoint.sh"]

CMD ["fluentd", "-c", "/fluentd/etc/fluent.conf", "-p", "/fluentd/plugins"]
