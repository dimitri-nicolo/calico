# Inspired by: https://github.com/envoyproxy/gateway/blob/v1.1.2/tools/docker/envoy-gateway/Dockerfile
# but modified to use calico/base instead of distroless

FROM alpine:3 AS builder
# Create the data directory for eg
RUN mkdir -p /var/lib/eg

FROM scratch as source
ARG TARGETARCH
COPY bin/envoy-gateway-$TARGETARCH /usr/bin/envoy-gateway
COPY --from=builder --chown=10001:10001 /var/lib/eg /var/lib/eg

FROM calico/base
COPY --from=source / /
USER 10001:10001
ENTRYPOINT ["/usr/bin/envoy-gateway"]
