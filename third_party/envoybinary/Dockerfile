FROM scratch AS binary

ARG ENVOY_ARCH

COPY envoy/ci/docker-entrypoint.sh /docker-entrypoint.sh
COPY envoy/configs/envoyproxy_io_proxy.yaml /etc/envoy/envoy.yaml
ADD envoy/build/envoy/${ENVOY_ARCH}/bin/release.tar.zst /usr/local/bin/

FROM calico/base AS proxy
EXPOSE 10000
ENTRYPOINT ["/usr/local/bin/envoy"]
CMD ["-c", "/etc/envoy/envoy.yaml"]
COPY --from=binary --chown=0:0 --chmod=644 \
    /etc/envoy/envoy.yaml /etc/envoy/envoy.yaml
COPY --from=binary --chown=0:0 --chmod=755 \
    /usr/local/bin/envoy /usr/local/bin/
