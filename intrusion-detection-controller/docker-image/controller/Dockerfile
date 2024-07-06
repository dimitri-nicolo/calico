# Copyright (c) 2023 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/database/GeoLite2-ASN.mmdb /etc/maxmind/GeoLite2-ASN.mmdb
COPY bin/database/GeoLite2-City.mmdb /etc/maxmind/GeoLite2-City.mmdb

COPY bin/controller-${TARGETARCH} /usr/bin/intrusion-detection-controller
COPY bin/healthz-${TARGETARCH} /usr/bin/healthz

FROM calico/base

COPY --from=source / /

# IDS controller syslog forwarding feature requires privileged user and group (0)
# to write event logs into /var/log/calico. The host path is owned by root.
# Tigera operator /operator/pkg/render/intrusion_detection.go controls UID and GID
# for IDS controller container. Setting UID/GID here to non-root as a fallback.
USER 10001:10001

ENTRYPOINT ["/usr/bin/intrusion-detection-controller"]
