FROM scratch

ARG ENVOY_ARCH

COPY envoy/ci/docker-entrypoint.sh /docker-entrypoint.sh
COPY envoy/configs/envoyproxy_io_proxy.yaml /etc/envoy/envoy.yaml
ADD envoy/build/envoy/${ENVOY_ARCH}/bin/release.tar.zst /usr/local/bin/
