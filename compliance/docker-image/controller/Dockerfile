FROM scratch as source

ARG TARGETARCH

COPY bin/controller-${TARGETARCH} /usr/bin/compliance-controller

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/compliance-controller"]
