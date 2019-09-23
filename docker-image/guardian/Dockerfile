FROM alpine:3.9

RUN apk update && apk add ca-certificates

RUN addgroup -S voltrongroup 
RUN adduser -S -D 999 voltrongroup 

# Add your binary to the image
ADD bin/guardian-amd64 /guardian

# Set the working directory to where you can run your application
WORKDIR /

RUN chown -R 999 /guardian

USER 999

# Now define the command-line call to start your application
ENTRYPOINT ["/guardian"]
