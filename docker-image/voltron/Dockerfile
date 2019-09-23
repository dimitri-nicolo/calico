FROM alpine:3.9

RUN apk update && apk add ca-certificates curl

RUN addgroup -S voltrongroup 
RUN adduser -S -D 999 voltrongroup 

# Add your binary to the image
ADD bin/voltron-amd64 /voltron
COPY templates/*.yaml.tmpl /tmp/

# Set the working directory to where you can run your application
WORKDIR /

RUN chown -R 999 /voltron
RUN chown -R 999 /tmp

USER 999

# Now define the command-line call to start your application
CMD ["sh", "-c", "/voltron"]
