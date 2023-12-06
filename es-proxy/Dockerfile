ARG QEMU_IMAGE

FROM ${QEMU_IMAGE} as qemu

FROM alpine:3 as base

# Add qemu static binaries when Calico Enterprise needs to support more architectures.
COPY --from=qemu /usr/bin/qemu-aarch64-static /usr/bin/qemu-aarch64-static

RUN mkdir -p /etc/es-proxy

FROM scratch as source

ARG TARGETARCH

COPY bin/es-proxy-${TARGETARCH} /usr/bin/es-proxy
# /etc/es-proxy directory used for storing self-signed certificates
COPY --chown=10001:10001 --from=base /etc/es-proxy /etc/es-proxy/

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/es-proxy"]
