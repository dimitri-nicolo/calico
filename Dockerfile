FROM alpine:3.6

ADD bin/calicoq ./calicoq

RUN apk add --no-cache bash curl docker wget
