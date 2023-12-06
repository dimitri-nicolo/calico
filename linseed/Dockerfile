FROM scratch as source

ARG TARGETARCH

COPY bin/linseed-${TARGETARCH} /linseed

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/linseed"]
