ARG QEMU_IMAGE

FROM ${QEMU_IMAGE} as qemu

FROM registry.access.redhat.com/ubi8/ubi:latest as builder

ARG ELASTIC_ARCH
ARG ELASTIC_VERSION
ARG TARGETARCH
ARG TINI_VERSION

COPY --from=qemu /usr/bin/qemu-*-static /usr/bin/

RUN dnf upgrade -y

################################################################################
# Install and config the elasticsearch archive
#
# Recreating the elasticsearch container image by copying essential files from
# UBI. Certain Dockerfile instructions in the scratch stage are extracted from:
# https://github.com/elastic/dockerfiles/blob/vx.y.z/elasticsearch/Dockerfile.
# We need to review and update them when upgrading to a newer version of
# elasticsearch.
################################################################################
COPY build/elasticsearch-${ELASTIC_VERSION}-linux-${ELASTIC_ARCH}.tar.gz /tmp/elasticsearch.tar.gz

RUN mkdir /usr/share/elasticsearch && \
    tar -zxf /tmp/elasticsearch.tar.gz --strip-components=1 -C /usr/share/elasticsearch/

COPY build/elasticsearch-${ELASTIC_VERSION}/distribution/docker/src/docker/bin/docker-entrypoint.sh /usr/share/elasticsearch/bin/docker-entrypoint.sh

# `tini` is a tiny but valid init for containers. This is used to cleanly
# control how ES and any child processes are shut down.
RUN curl -sfL https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini-${TARGETARCH} -o /usr/bin/tini && \
    chmod 0755 /usr/bin/tini

# Add storage plugin
RUN /usr/share/elasticsearch/bin/elasticsearch-plugin install --batch repository-gcs

# Change /usr/share/elasticsearch folder ownership to elasticsearch user and group.
# Elastic gradle step installs bundle JDK into the /usr/share/elasticsearch/jdk folder.
# The folder permisison is 0750 and when we deploy this UBI based container to Openshift,
# which has SELinux enabled by default, elasticsearch user get permissions errors on
# accessing JDK files. This is not an issue on non-SELinux enabled hosts like Ubuntu.
# Changing the folder ownership works on both hosts.
RUN groupadd -g 1000 elasticsearch && \
    adduser --uid 1000 --gid 1000 --home /usr/share/elasticsearch elasticsearch && \
    chown -R 1000:1000 /usr/share/elasticsearch

################################################################################
# Build the actual scratch-based elasticsearch image
################################################################################
FROM scratch as source

ARG TARGETARCH

# binary dependencies
COPY --from=builder /bin/bash /bin/bash
COPY --from=builder /bin/sh /bin/sh
COPY --from=builder /usr/bin/basename /usr/bin/basename
COPY --from=builder /usr/bin/bash /usr/bin/bash
COPY --from=builder /usr/bin/cat /usr/bin/cat
COPY --from=builder /usr/bin/chown /usr/bin/chown
COPY --from=builder /usr/bin/coreutils /usr/bin/coreutils
COPY --from=builder /usr/bin/cp /usr/bin/cp
COPY --from=builder /usr/bin/date /usr/bin/date
COPY --from=builder /usr/bin/dirname /usr/bin/dirname
COPY --from=builder /usr/bin/env /usr/bin/env
COPY --from=builder /usr/bin/grep /usr/bin/grep
COPY --from=builder /usr/bin/id /usr/bin/id
COPY --from=builder /usr/bin/ln /usr/bin/ln
COPY --from=builder /usr/bin/ls /usr/bin/ls
COPY --from=builder /usr/bin/mkdir /usr/bin/mkdir
COPY --from=builder /usr/bin/sh /usr/bin/sh
COPY --from=builder /usr/bin/sleep /usr/bin/sleep
COPY --from=builder /usr/bin/touch /usr/bin/touch
COPY --from=builder /usr/bin/uname /usr/bin/uname
COPY --from=builder /usr/bin/yes /usr/bin/yes
COPY --from=builder /usr/sbin/chroot /usr/sbin/chroot

COPY --from=builder /lib64/libacl.so.1 /lib64/libacl.so.1
COPY --from=builder /lib64/libattr.so.1 /lib64/libattr.so.1
COPY --from=builder /lib64/libcap.so.2 /lib64/libcap.so.2
COPY --from=builder /lib64/libdl.so.2 /lib64/libdl.so.2
COPY --from=builder /lib64/libpcre.so.1 /lib64/libpcre.so.1
COPY --from=builder /lib64/libpcre2-8.so.0 /lib64/libpcre2-8.so.0
COPY --from=builder /lib64/librt.so.1 /lib64/librt.so.1
COPY --from=builder /lib64/libselinux.so.1 /lib64/libselinux.so.1
COPY --from=builder /lib64/libtinfo.so.6 /lib64/libtinfo.so.6

# required by jvm
COPY --from=builder /lib64/libm.so.6 /lib64/libm.so.6
COPY --from=builder /lib64/libz.so.1 /lib64/libz.so.1

# user and groups
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/shadow /etc/shadow

# elasticsearch jars and binaries
COPY --from=builder /usr/share/elasticsearch /usr/share/elasticsearch/
COPY --from=builder /usr/bin/tini /usr/bin/tini
# tigera custom elasticsearch readiness-probe
COPY bin/readiness-probe-${TARGETARCH} /usr/bin/readiness-probe

FROM calico/base

ENV ELASTIC_CONTAINER=true
ENV PATH=/usr/share/elasticsearch/bin:/usr/sbin:/usr/bin:/sbin:/bin

COPY --from=source / /

WORKDIR /usr/share/elasticsearch

EXPOSE 9200 9300

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/share/elasticsearch/bin/docker-entrypoint.sh"]

CMD ["eswrapper"]
