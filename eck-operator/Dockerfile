# Copyright (c) 2024 Tigera, Inc. All rights reserved.

FROM scratch as source

ARG TARGETARCH

COPY bin/eck-operator-amd64 /elastic-operator
COPY cloud-on-k8s/config/eck.yaml /conf/eck.yaml

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/elastic-operator"]

CMD ["manager"]
