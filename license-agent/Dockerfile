FROM scratch as source

ARG TARGETARCH

COPY bin/license-agent-${TARGETARCH} /usr/bin/license-agent

FROM calico/base

COPY --from=source / /

ENTRYPOINT ["/usr/bin/license-agent"]
