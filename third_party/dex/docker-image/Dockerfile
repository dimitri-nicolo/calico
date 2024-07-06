# Copyright (c) 2024 Tigera, Inc. All rights reserved.

FROM scratch AS source

ARG TARGETARCH

COPY bin/dex-${TARGETARCH} /dex
COPY docker-image/web /srv/dex/web

FROM calico/base

COPY --from=source / /

ENV DEX_FRONTEND_DIR=/srv/dex/web

USER 10001:10001

# Tigera operator overrides this entry point and passes a config
# with values defined by config map tigera-dex in namespace tigera-dex
# In order to run this locally, replace /etc/dex/config.docker.yaml with
# a config that uses K8S as a datastore as gomplate will not be invoked
# to tailor the config file based on environment variables
ENTRYPOINT ["/dex", "serve", "/etc/dex/config.docker.yaml"]
