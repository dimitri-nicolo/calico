# Copyright (c) 2023 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/webhooks-processor-${TARGETARCH} /webhooks-processor

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/webhooks-processor"]
