FROM scratch as source

ARG TARGETARCH

COPY bin/firewall-integration-${TARGETARCH} /usr/bin/firewall-integration

FROM calico/base

COPY --from=source / /

USER 10001:10001

ENTRYPOINT ["/usr/bin/firewall-integration"]
