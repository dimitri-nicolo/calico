# Copyright (c) 2023 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/controller-${TARGETARCH} /usr/bin/compliance-controller

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/compliance-controller"]
