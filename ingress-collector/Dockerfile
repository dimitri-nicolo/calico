# Copyright (c) 2023 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/ingress-collector-${TARGETARCH} /usr/bin/ingress-collector

FROM calico/base

COPY --from=source / /

USER 10001:10001

CMD ["/usr/bin/ingress-collector"]
