FROM scratch as source

ARG TARGETARCH

COPY bin/reporter-${TARGETARCH} /usr/bin/compliance-reporter

FROM calico/base

COPY --from=source / /

ENTRYPOINT ["/usr/bin/compliance-reporter"]
