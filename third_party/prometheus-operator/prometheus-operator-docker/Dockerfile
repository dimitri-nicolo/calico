FROM scratch AS source

ARG TARGETARCH

COPY bin/prometheus-operator-${TARGETARCH} /usr/bin/prometheus-operator

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/prometheus-operator"]
