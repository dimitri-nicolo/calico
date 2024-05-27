FROM scratch as source

ARG TARGETARCH

COPY bin/elasticsearch-metrics-${TARGETARCH} /bin/elasticsearch_exporter

FROM calico/base

COPY --from=source / /

EXPOSE 9081

ENTRYPOINT  ["/bin/elasticsearch_exporter"]
