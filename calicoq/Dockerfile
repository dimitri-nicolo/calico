FROM scratch as source

ARG TARGETARCH

COPY bin/calicoq-${TARGETARCH} /usr/bin/calicoq

FROM calico/base

COPY --from=source / /

ENTRYPOINT ["/usr/bin/calicoq"]
