# Copyright (c) 2021-2024 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/elasticsearch-metrics-${TARGETARCH} /bin/elasticsearch_exporter

FROM calico/base

COPY --from=source / /

EXPOSE 9081

ENTRYPOINT  ["/bin/elasticsearch_exporter"]
