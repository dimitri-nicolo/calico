# Copyright (c) 2023 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/snapshotter-${TARGETARCH} /usr/bin/compliance-snapshotter

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/compliance-snapshotter"]
