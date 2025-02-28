# Copyright (c) 2024-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY bin/eck-operator-${TARGETARCH} /usr/bin/elastic-operator
COPY cloud-on-k8s/config/eck.yaml /conf/eck.yaml

FROM ${CALICO_BASE}

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/elastic-operator"]

CMD ["manager"]
