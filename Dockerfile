################################################################################
# This Dockerfile was generated from the template at distribution/src/docker/Dockerfile
#
# Beginning of multi stage Dockerfile
################################################################################

################################################################################
# Build stage 0 `builder`:
# Extract elasticsearch artifact
# Install required plugins
# Set gid=0 and make group perms==owner perms
################################################################################

FROM centos:7 AS builder

ENV PATH /usr/share/elasticsearch/bin:$PATH

RUN groupadd -g 1000 elasticsearch &&     adduser -u 1000 -g 1000 -d /usr/share/elasticsearch elasticsearch

WORKDIR /usr/share/elasticsearch

RUN cd /opt && curl --retry 8 -s -L -O https://artifacts.elastic.co/downloads/elasticsearch/elasticsearch-7.3.2-linux-x86_64.tar.gz && cd -

RUN tar zxf /opt/elasticsearch-7.3.2-linux-x86_64.tar.gz --strip-components=1
RUN grep ES_DISTRIBUTION_TYPE=tar /usr/share/elasticsearch/bin/elasticsearch-env     && sed -ie 's/ES_DISTRIBUTION_TYPE=tar/ES_DISTRIBUTION_TYPE=docker/' /usr/share/elasticsearch/bin/elasticsearch-env
RUN mkdir -p config data logs
RUN chmod 0775 config data logs
COPY config/elasticsearch.yml config/log4j2.properties config/

################################################################################
# Build stage 1 (the actual elasticsearch image):
# Copy elasticsearch from stage 0
# Add entrypoint
################################################################################

FROM centos:7

ENV ELASTIC_CONTAINER true

RUN for iter in {1..10};do yum update -y &&     yum install -y nc &&     yum clean all && exit_code=0 && break || exit_code=$? && echo "yum error: retry $iter in 10s" && sleep 10; done;     (exit $exit_code)

RUN groupadd -g 1000 elasticsearch &&     adduser -u 1000 -g 1000 -G 0 -d /usr/share/elasticsearch elasticsearch &&     chmod 0775 /usr/share/elasticsearch &&     chgrp 0 /usr/share/elasticsearch

WORKDIR /usr/share/elasticsearch
COPY --from=builder --chown=1000:0 /usr/share/elasticsearch /usr/share/elasticsearch

# Replace OpenJDK's built-in CA certificate keystore with the one from the OS
# vendor. The latter is superior in several ways.
# REF: https://github.com/elastic/elasticsearch-docker/issues/171
RUN ln -sf /etc/pki/ca-trust/extracted/java/cacerts /usr/share/elasticsearch/jdk/lib/security/cacerts

ENV PATH /usr/share/elasticsearch/bin:$PATH

COPY --chown=1000:0 bin/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# Openshift overrides USER and uses ones with randomly uid>1024 and gid=0
# Allow ENTRYPOINT (and ES) to run even with a different user
RUN chgrp 0 /usr/local/bin/docker-entrypoint.sh &&     chmod g=u /etc/passwd &&     chmod 0775 /usr/local/bin/docker-entrypoint.sh

EXPOSE 9200 9300

LABEL org.label-schema.build-date="2019-09-06T14:40:30.410020Z"   org.label-schema.license="Elastic-License"   org.label-schema.name="Elasticsearch"   org.label-schema.schema-version="1.0"   org.label-schema.url="https://www.elastic.co/products/elasticsearch"   org.label-schema.usage="https://www.elastic.co/guide/en/elasticsearch/reference/index.html"   org.label-schema.vcs-ref="1c1faf179b40cccb785fb00bf32b2a91176d6c85"   org.label-schema.vcs-url="https://github.com/elastic/elasticsearch"   org.label-schema.vendor="Elastic"   org.label-schema.version="7.3.2"   org.opencontainers.image.created="2019-09-06T14:40:30.410020Z"   org.opencontainers.image.documentation="https://www.elastic.co/guide/en/elasticsearch/reference/index.html"   org.opencontainers.image.licenses="Elastic-License"   org.opencontainers.image.revision="1c1faf179b40cccb785fb00bf32b2a91176d6c85"   org.opencontainers.image.source="https://github.com/elastic/elasticsearch"   org.opencontainers.image.title="Elasticsearch"   org.opencontainers.image.url="https://www.elastic.co/products/elasticsearch"   org.opencontainers.image.vendor="Elastic"   org.opencontainers.image.version="7.3.2"

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
# Dummy overridable parameter parsed by entrypoint
CMD ["eswrapper"]

RUN rpm -e curl --nodeps
RUN rpm -e libcurl --nodeps
RUN rpm -e file-libs --nodeps
RUN rpm -e libdb-utils --nodeps
RUN rpm -e procps-ng --nodeps
RUN rpm -e libssh2 --nodeps
RUN rpm -e libsmartcols --nodeps
RUN rpm -e krb5-libs --nodeps
RUN rpm -e libblkid --nodeps
RUN rpm -e libuuid --nodeps
RUN rpm -e libmount --nodeps
RUN rpm -e util-linux --nodeps
RUN rpm -e libgcc --nodeps
RUN rpm -e python --nodeps
RUN rpm -e python-libs --nodeps
RUN rpm -e libstdc++ --nodeps
RUN rpm -e glib2 --nodeps
RUN rpm -e glibc-common --nodeps
#RUN rpm -e glibc --nodeps
#RUN rpm -e coreutils --nodeps
RUN rpm -e binutils --nodeps
RUN rpm -e lz4 --nodeps
RUN rpm -e libxml2 --nodeps
RUN rpm -e libxml2-python --nodeps
RUN rpm -e readline --nodeps
#RUN rpm -e libcap --nodeps
RUN rpm -e elfutils-libs --nodeps
#RUN rpm -e elfutils-libelf --nodeps
RUN rpm -e nss-sysinit --nodeps
RUN rpm -e nss-tools --nodeps
#RUN rpm -e nss --nodeps 
#RUN rpm -e nss-softokn-freebl --nodeps
#RUN rpm -e nss-softokn --nodeps
RUN rpm -e expat --nodeps
RUN rpm -e vim-minimal --nodeps
RUN rpm -e elfutils-default-yama-scope --nodeps
#RUN rpm -e bzip2-libs --nodeps
RUN rpm -e ncurses-base --nodeps
RUN rpm -e openldap --nodeps
RUN rpm -e libidn --nodeps
RUN rpm -e gnupg2 --nodeps
RUN rpm -e gpgme --nodeps
#RUN rpm -e lua --nodeps
#RUN rpm -e ncurses-libs --nodeps
RUN rpm -e ncurses --nodeps
RUN rpm -e libtasn1 --nodeps
RUN rpm -e json-c  --nodeps
#RUN rpm -e bash --nodeps
#RUN rpm -e zlib --nodeps
#RUN rpm -e libdb --nodeps
RUN rpm -e systemd-libs --nodeps
RUN rpm -e sqlite --nodeps
#RUN rpm -e libdb --nodeps
#RUN rpm -e rpm-libs --nodeps
#RUN rpm -e rpm --nodeps
