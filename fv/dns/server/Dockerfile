FROM alpine:latest

ADD dns-server /dns-server
ADD entrypoint.sh /entrypoint.sh
EXPOSE 54/udp
ENTRYPOINT ["./entrypoint.sh"]
