FROM alpine:3 as base

RUN apk update && apk add iptables
ADD init_iptables.sh init_iptables.sh
RUN chmod u+x init_iptables.sh

ENTRYPOINT ["./init_iptables.sh"]
