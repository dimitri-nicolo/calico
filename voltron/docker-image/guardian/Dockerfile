FROM scratch as source

ARG TARGETARCH

COPY bin/guardian-${TARGETARCH} /usr/bin/guardian

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/guardian"]
