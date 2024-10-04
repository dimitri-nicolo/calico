# Copyright (c) 2023 Tigera, Inc. All rights reserved.
ARG K8S_VERSION=1.29.4

FROM alpine:3 AS builder

ARG TARGETARCH
ARG K8S_VERSION
ARG CONTROLLER_RUNTIME_VERSION=v0.19.0

RUN apk add --no-cache curl

RUN curl -sfL https://github.com/kubernetes-sigs/controller-runtime/releases/download/${CONTROLLER_RUNTIME_VERSION}/setup-envtest-linux-${TARGETARCH} -o /usr/bin/setup-envtest && chmod +x /usr/bin/setup-envtest

# Download etcd, kube-apiserver, and kubectl binaries using setup-envtest
RUN setup-envtest use ${K8S_VERSION} --bin-dir /usr/bin

FROM scratch AS source

FROM calico/base

ARG TARGETARCH
ARG K8S_VERSION

COPY --from=builder /usr/bin/k8s/${K8S_VERSION}-linux-${TARGETARCH}/ /usr/share/kubebuilder-envtest/
COPY bin/fake-guardian-${TARGETARCH} /usr/bin/fake-guardian

ARG GIT_VERSION=unknown

# Required labels for certification
LABEL description="Fake Guardian for managed cluster scale testing"
LABEL maintainers="Glendon Ngo (glen@tigera.io), Gordon Cosgrave (gordon@tigera.io)"
LABEL name="Fake Guardian"
LABEL release="1"
LABEL summary="Fake Guardian for managed cluster scale testing"
LABEL vendor="Tigera"
LABEL version=${GIT_VERSION}

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/fake-guardian"]
