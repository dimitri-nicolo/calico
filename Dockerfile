FROM alpine:3.6

ADD bin/calicoq ./calicoq

ENV PATH=$PATH:/

RUN apk add --no-cache bash curl docker wget

WORKDIR /root
ENTRYPOINT ["/calicoq"]
