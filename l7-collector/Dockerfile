# Copyright (c) 2023 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/l7-collector-${TARGETARCH} /usr/bin/l7-collector

FROM calico/base

COPY --from=source / /

ENTRYPOINT ["/usr/bin/l7-collector"]
