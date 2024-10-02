ARG QEMU_IMAGE
ARG THIRD_PARTY_REGISTRY
ARG UBI9_IMAGE

FROM ${QEMU_IMAGE} AS qemu

FROM ${UBI9_IMAGE} AS ubi

COPY --from=qemu /usr/bin/qemu-*-static /usr/bin/

# Update base packages to pick up security updates. 
RUN microdnf upgrade

RUN microdnf install -y shadow-utils

FROM envoyproxy/envoy:v1.29.9 AS envoy

FROM ${THIRD_PARTY_REGISTRY}/envoybinary:v1.29.9 AS envoybinary

FROM scratch AS source

# dependencies
COPY --from=ubi /bin/sh /bin/sh
COPY --from=ubi /usr/bin/chown /usr/bin/chown
COPY --from=ubi /usr/bin/coreutils /usr/bin/coreutils
COPY --from=ubi /usr/bin/env /usr/bin/env
COPY --from=ubi /usr/bin/id /usr/bin/id
COPY --from=ubi /usr/sbin/groupmod /usr/sbin/groupmod
COPY --from=ubi /usr/sbin/usermod /usr/sbin/usermod

# arm64 loader under /lib/ld-linux-aarch64.so.1
COPY --from=ubi /lib/ld-linux-*.so.? /lib/
# amd64 loader under /lib64/ld-linux-x86-64.so.2
COPY --from=ubi /lib64/ld-linux-*.so.? /lib64/
COPY --from=ubi /lib64/libacl.so.1 /lib64/libacl.so.1
COPY --from=ubi /lib64/libattr.so.1 /lib64/libattr.so.1
COPY --from=ubi /lib64/libaudit.so.1 /lib64/libaudit.so.1
COPY --from=ubi /lib64/libc.so.6 /lib64/libc.so.6
COPY --from=ubi /lib64/libcap-ng.so.0 /lib64/libcap-ng.so.0
COPY --from=ubi /lib64/libcap.so.2 /lib64/libcap.so.2
COPY --from=ubi /lib64/libdl.so.2 /lib64/libdl.so.2
COPY --from=ubi /lib64/libm.so.6 /lib64/libm.so.6
COPY --from=ubi /lib64/libpcre2-8.so.0 /lib64/libpcre2-8.so.0
COPY --from=ubi /lib64/libpthread.so.0 /lib64/libpthread.so.0
COPY --from=ubi /lib64/librt.so.1 /lib64/librt.so.1
COPY --from=ubi /lib64/libselinux.so.1 /lib64/libselinux.so.1
COPY --from=ubi /lib64/libtinfo.so.6 /lib64/libtinfo.so.6

# envoy
COPY --from=envoybinary --chmod=755 /usr/local/bin/envoy /usr/bin/envoy

COPY --from=envoy /docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=envoy /usr/local/bin/su-exec /usr/bin/su-exec

COPY --from=envoy /etc/envoy/VERSION.txt /etc/envoy/VERSION.txt
COPY --from=envoy /etc/envoy/envoy.yaml /etc/envoy/envoy.yaml
COPY --from=envoy /etc/group /etc/group
COPY --from=envoy /etc/passwd /etc/passwd

FROM scratch

COPY --from=source / /

ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["envoy", "-c", "/etc/envoy/envoy.yaml"]
