# Copyright (c) 2023-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

# Copy in the fake-log-generator and example flows
COPY bin/fake-log-generator-${TARGETARCH} /usr/bin/fake-log-generator
COPY example-flows /home/log-gen/

FROM ${CALICO_BASE}

ARG GIT_VERSION=unknown

# Required labels for certification
LABEL description="Calico fake log generator for scale testing"
LABEL maintainer="lance@tigera.io"
LABEL name="Calico fake log generator"
LABEL release="1"
LABEL summary="Calico fake log generator for scale testing"
LABEL vendor="Tigera"
LABEL version=${GIT_VERSION}

COPY --from=source / /

WORKDIR /home/log-gen

ENTRYPOINT ["/usr/bin/fake-log-generator"]
