# Copyright (c) 2023-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY bin/ingress-collector-${TARGETARCH} /usr/bin/ingress-collector

FROM ${CALICO_BASE}

COPY --from=source / /

USER 10001:10001

CMD ["/usr/bin/ingress-collector"]
