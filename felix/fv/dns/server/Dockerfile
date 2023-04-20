ARG UBI_IMAGE

FROM ${UBI_IMAGE}

COPY dns-server /dns-server
COPY entrypoint.sh /entrypoint.sh

EXPOSE 54/udp

ENTRYPOINT ["/entrypoint.sh"]
