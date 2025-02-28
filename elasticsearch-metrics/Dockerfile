# Copyright (c) 2021-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY bin/elasticsearch-metrics-${TARGETARCH} /bin/elasticsearch_exporter

FROM ${CALICO_BASE}

COPY --from=source / /

EXPOSE 9081

ENTRYPOINT  ["/bin/elasticsearch_exporter"]
