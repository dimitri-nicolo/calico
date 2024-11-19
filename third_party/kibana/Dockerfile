ARG KIBANA_VERSION
ARG QEMU_IMAGE

FROM ${QEMU_IMAGE} AS qemu

ARG TARGETARCH

FROM docker.elastic.co/kibana/kibana-ubi:${KIBANA_VERSION}-${TARGETARCH}

COPY --from=qemu /usr/bin/qemu-*-static /usr/bin/

USER 0

ARG GTM_INTEGRATION
ARG LATEST_IMAGE_TAG

ENV PATH=/usr/share/kibana/bin:$PATH

WORKDIR /usr/share/kibana

COPY kibana/plugins/tigera/build/tigera-${LATEST_IMAGE_TAG}.zip /tmp/tigera-${LATEST_IMAGE_TAG}.zip
RUN /usr/share/kibana/bin/kibana-plugin install file:///tmp/tigera-${LATEST_IMAGE_TAG}.zip && \
    rm -f /tmp/tigera-${LATEST_IMAGE_TAG}.zip

RUN if [ "$GTM_INTEGRATION" = "enabled" ]; then \
        cp kibana/plugins/google_tag_manager/build/googleTagManager-${LATEST_IMAGE_TAG}.zip /tmp/googleTagManager-${LATEST_IMAGE_TAG}.zip && \
        /usr/share/kibana/bin/kibana-plugin install file:///tmp/googleTagManager-${LATEST_IMAGE_TAG}.zip; \
        rm -f /tmp/googleTagManager-${LATEST_IMAGE_TAG}.zip; \
    fi


CMD ["/usr/local/bin/kibana-docker"]

