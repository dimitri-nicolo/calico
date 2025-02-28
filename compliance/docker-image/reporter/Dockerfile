# Copyright (c) 2023-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY bin/reporter-${TARGETARCH} /usr/bin/compliance-reporter

FROM ${CALICO_BASE}

COPY --from=source / /

ENTRYPOINT ["/usr/bin/compliance-reporter"]
