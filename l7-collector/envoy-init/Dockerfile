ARG QEMU_IMAGE

FROM ${QEMU_IMAGE} AS qemu

FROM alpine:3.18 AS base

COPY --from=qemu /usr/bin/qemu-*-static /usr/bin/

RUN apk update && apk upgrade && apk add iptables

# Set permissions specifically for OpenShift
RUN chgrp -R 0 /sbin/iptables && \
    chmod -R g=u /sbin/iptables

RUN chgrp -R 0 /usr/lib/xtables && \
    chmod -R g=u /usr/lib/xtables


FROM scratch

COPY --from=base /bin/sh /bin/sh
COPY --from=base /sbin/iptables /sbin/iptables

COPY --from=base /lib/ld-musl-*.so.1 /lib/

COPY --from=base /usr/lib/libip4tc.so.2 /usr/lib/libip4tc.so.2
COPY --from=base /usr/lib/libip6tc.so.2 /usr/lib/libip6tc.so.2
COPY --from=base /usr/lib/libxtables.so.12 /usr/lib/libxtables.so.12
COPY --from=base /usr/lib/xtables/* /usr/lib/xtables/

COPY init_iptables.sh /usr/bin/init_iptables.sh

ENV PATH=/usr/local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

ENTRYPOINT ["/usr/bin/init_iptables.sh"]
