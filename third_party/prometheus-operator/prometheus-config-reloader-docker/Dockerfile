# Copyright (c) 2021-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY bin/prometheus-config-reloader-${TARGETARCH} /bin/prometheus-config-reloader

FROM ${CALICO_BASE}

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/bin/prometheus-config-reloader"]
