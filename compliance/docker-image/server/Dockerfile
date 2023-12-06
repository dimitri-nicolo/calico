FROM scratch as source

ARG TARGETARCH

COPY bin/server-${TARGETARCH} /usr/bin/compliance-server

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/compliance-server"]
