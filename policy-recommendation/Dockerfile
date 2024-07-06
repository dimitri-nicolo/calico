# Copyright (c) 2024 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/policy-recommendation-${TARGETARCH} /usr/bin/policy-recommendation

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/policy-recommendation"]
