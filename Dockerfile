FROM scratch

LABEL maintainer="tom@tigera.io"

COPY bin/calicoq /
ENTRYPOINT ["/calicoq"]
