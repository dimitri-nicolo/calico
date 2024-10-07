# Copyright (c) 2023 Tigera, Inc. All rights reserved.

ARG QEMU_IMAGE

FROM ${QEMU_IMAGE} AS qemu

FROM alpine:3 AS builder

COPY --from=qemu /usr/bin/qemu-*-static /usr/bin/

RUN mkdir -p /etc/ui-apis

FROM scratch AS source

ARG TARGETARCH

COPY bin/ui-apis-${TARGETARCH} /usr/bin/ui-apis
# /etc/ui-apis directory used for storing self-signed certificates
COPY --chown=10001:10001 --from=builder /etc/ui-apis /etc/ui-apis/

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/ui-apis"]
