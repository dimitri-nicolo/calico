FROM alpine:3.13.6

COPY dns-server /dns-server
COPY entrypoint.sh /entrypoint.sh
EXPOSE 54/udp
ENTRYPOINT ["./entrypoint.sh"]
