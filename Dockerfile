FROM alpine:3.4

LABEL maintainer="tom@tigera.io"

COPY bin/calicoq /
ENTRYPOINT ["/calicoq"]
