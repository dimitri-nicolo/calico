# Copyright (c) 2023-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY bin/guardian-${TARGETARCH} /usr/bin/guardian

FROM ${CALICO_BASE}

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/guardian"]
