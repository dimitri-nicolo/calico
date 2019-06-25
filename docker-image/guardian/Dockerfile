FROM alpine:3.9

RUN apk update && apk add ca-certificates

# Add your binary to the image
ADD bin/guardian-amd64 /guardian

# Set the working directory to where you can run your application
WORKDIR /

# Now define the command-line call to start your application
ENTRYPOINT ["/guardian"]
