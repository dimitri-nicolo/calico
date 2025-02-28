ARG KIBANA_VERSION

ARG TARGETARCH

FROM docker.elastic.co/kibana/kibana-ubi:${KIBANA_VERSION}-${TARGETARCH}

USER 0

ARG GTM_INTEGRATION
ARG LATEST_IMAGE_TAG

ENV PATH=/usr/share/kibana/bin:$PATH

WORKDIR /usr/share/kibana

COPY kibana/plugins/tigera/build/tigera-${LATEST_IMAGE_TAG}.zip /tmp/tigera-${LATEST_IMAGE_TAG}.zip
RUN /usr/share/kibana/bin/kibana-plugin install file:///tmp/tigera-${LATEST_IMAGE_TAG}.zip && \
    rm -f /tmp/tigera-${LATEST_IMAGE_TAG}.zip

COPY kibana/plugins/googletagmanager/build/googletagmanager-${LATEST_IMAGE_TAG}.zip /tmp/googletagmanager-${LATEST_IMAGE_TAG}.zip
RUN if [ "$GTM_INTEGRATION" = "enabled" ]; then \
        echo "Installing Google Tag Manager plugin..."; \
        /usr/share/kibana/bin/kibana-plugin install file:///tmp/googletagmanager-${LATEST_IMAGE_TAG}.zip; \
    else \
        echo  "Google Tag Manager integration is not enabled"; \
    fi

RUN rm -f /tmp/googletagmanager-${LATEST_IMAGE_TAG}.zip


CMD ["/usr/local/bin/kibana-docker"]

