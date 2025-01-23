# This is a copy of
# https://github.com/envoyproxy/gateway/blob/v1.1.2/tools/docker/envoy-gateway/Dockerfile
# but:
# - using calico/base instead of distroless
# - using TARGETARCH instead of TARGETPLATFORM.

ARG QEMU_IMAGE
FROM ${QEMU_IMAGE} AS qemu

FROM busybox AS source
COPY --from=qemu /usr/bin/qemu-*-static /usr/bin/
# Create the data directory for eg
RUN mkdir -p /var/lib/eg

FROM calico/base:latest-${TARGETARCH}
ARG TARGETARCH
COPY bin/envoy-gateway-${TARGETARCH} /usr/local/bin/
COPY --from=source --chown=65532:65532 /var/lib /var/lib

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/envoy-gateway"]
