FROM scratch as source

ARG TARGETARCH

COPY bin/elasticsearch-metrics-${TARGETARCH} /usr/bin/elasticsearch_exporter

FROM calico/base

COPY --from=source / /

EXPOSE 9081

ENTRYPOINT  ["/usr/bin/elasticsearch_exporter"]
