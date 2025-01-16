# This is a copy of
# https://github.com/envoyproxy/gateway/blob/v1.1.2/tools/docker/envoy-gateway/Dockerfile
# but:
# - using calico/base instead of distroless
# - using ARCH instead of TARGETPLATFORM.

FROM busybox AS source
# Create the data directory for eg
RUN mkdir -p /var/lib/eg

FROM calico/base
ARG ARCH
COPY bin/envoy-gateway-$ARCH /usr/local/bin/
COPY --from=source --chown=65532:65532 /var/lib /var/lib

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/envoy-gateway"]
