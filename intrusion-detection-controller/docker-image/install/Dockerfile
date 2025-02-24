# Copyright (c) 2023-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY bin/intrusion-detection-job-installer-${TARGETARCH} /usr/bin/intrusion-detection-job-installer
COPY cmd/install/data/flowlog-dashboard.ndjson /etc/dashboards/flowlog-dashboard.ndjson

FROM ${CALICO_BASE}

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/intrusion-detection-job-installer"]
