# Copyright (c) 2024-2025 Tigera, Inc. All rights reserved.

ARG CALICO_BASE

FROM scratch AS source

ARG TARGETARCH

COPY dex/bin/dex-${TARGETARCH} /usr/bin/dex
COPY docker-image/web /srv/dex/web

FROM ${CALICO_BASE}

COPY --from=source / /

ENV DEX_FRONTEND_DIR=/srv/dex/web

USER 10001:10001

# Tigera operator overrides this entry point and passes a config
# with values defined by config map tigera-dex in namespace tigera-dex
# In order to run this locally, replace /etc/dex/config.docker.yaml with
# a config that uses K8S as a datastore as gomplate will not be invoked
# to tailor the config file based on environment variables
ENTRYPOINT ["/usr/bin/dex", "serve", "/etc/dex/config.docker.yaml"]
