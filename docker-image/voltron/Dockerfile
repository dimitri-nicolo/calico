FROM alpine:3.9

RUN apk update && apk add ca-certificates curl

# Add your binary to the image
ADD bin/voltron-amd64 /voltron
COPY scripts/*.bash /scripts/
COPY templates/*.yaml.tmpl /tmp/

# Set the working directory to where you can run your application
WORKDIR /

# Now define the command-line call to start your application
CMD ["sh", "-c", "/voltron"]
