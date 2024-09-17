# Copyright (c) 2024 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/l7-admission-controller-${TARGETARCH} /usr/bin/l7-admission-controller

FROM calico/base

ARG GIT_VERSION=unknown

LABEL description="Calico L7AdmissionController enables mutating webhooks for Application Layer Policy sidecar injection"
LABEL maintainer="maintainers@tigera.io"
LABEL name="Calico L7AdmissionController"
LABEL release="1"
LABEL summary="Calico L7AdmissionController enables mutating webhooks for Application Layer Policy sidecar injection"
LABEL vendor="Project Calico"
LABEL version=${GIT_VERSION}

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/l7-admission-controller"]
