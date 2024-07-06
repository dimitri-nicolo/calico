# Copyright (c) 2023 Tigera, Inc. All rights reserved.

ARG QEMU_IMAGE

FROM ${QEMU_IMAGE} AS qemu

FROM alpine:3 AS builder

ARG K8S_VERSION
ARG KUBE_BENCH_VERSION
ARG TARGETARCH

COPY --from=qemu /usr/bin/qemu-*-static /usr/bin/

RUN apk add --no-cache curl

# install kube-bench configs
# ref: https://github.com/aquasecurity/kube-bench/blob/main/docs/installation.md
RUN curl -sfL https://github.com/aquasecurity/kube-bench/archive/refs/tags/${KUBE_BENCH_VERSION}.tar.gz -o /tmp/kube-bench.tar.gz && \
	mkdir -p /tmp/kube-bench && tar -xf /tmp/kube-bench.tar.gz --strip-components 1 -C /tmp/kube-bench && \
	mkdir -p /etc/kube-bench && mv /tmp/kube-bench/cfg /etc/kube-bench/

# install kubectl
RUN curl -sfL https://dl.k8s.io/release/${K8S_VERSION}/bin/linux/${TARGETARCH}/kubectl -o /usr/bin/kubectl && \
	chmod 755 /usr/bin/kubectl

FROM scratch AS source

ARG TARGETARCH

COPY bin/benchmarker-${TARGETARCH} /usr/bin/compliance-benchmarker
COPY bin/kube-bench-${TARGETARCH} /usr/bin/kube-bench

COPY --from=builder /etc/kube-bench /etc/kube-bench/
COPY --from=builder /usr/bin/kubectl /usr/bin/kubectl

FROM calico/base

COPY --from=source / /

ENTRYPOINT ["/usr/bin/compliance-benchmarker"]
