FROM alpine:latest
RUN addgroup -S app
RUN adduser -S -D 999 app

USER 999
ENTRYPOINT ["/bin/sh", "-c", "while true;do sleep 5; done"]

