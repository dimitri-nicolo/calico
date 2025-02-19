# Copyright (c) 2023-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY bin/l7-collector-${TARGETARCH} /usr/bin/l7-collector

FROM ${CALICO_BASE}

COPY --from=source / /

ENTRYPOINT ["/usr/bin/l7-collector"]
