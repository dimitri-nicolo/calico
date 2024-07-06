FROM scratch AS source

ARG TARGETARCH

COPY bin/prometheus-config-reloader-${TARGETARCH} /bin/prometheus-config-reloader

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/bin/prometheus-config-reloader"]
