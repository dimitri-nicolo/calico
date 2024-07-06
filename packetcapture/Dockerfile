# Copyright (c) 2023 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/packetcapture-${TARGETARCH} /usr/bin/packetcapture

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/packetcapture"]
