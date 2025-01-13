We need to build Tigera equivalents of these 3 images that are used for Gateway
API function:

```
	ComponentGatewayAPIEnvoyGateway = Component{
		Version:  "v1.1.2",
		Image:    "envoyproxy/gateway",
		Registry: "docker.io/",
	}

	ComponentGatewayAPIEnvoyProxy = Component{
		Version:  "distroless-v1.31.0",
		Image:    "envoyproxy/envoy",
		Registry: "docker.io/",
	}

	ComponentGatewayAPIEnvoyRatelimit = Component{
		Version:  "26f28d78",
		Image:    "envoyproxy/ratelimit",
		Registry: "docker.io/",
	}
```

# envoyproxy/gateway:v1.1.2

https://hub.docker.com/layers/envoyproxy/gateway/v1.1.2/images/sha256-8c834a178061fa9f08049fd11773f3c0e044b75d6495256c5e32f3c34cd11072
says that the layers are formed from the following commands:

```
ARG TARGETPLATFORM=linux/amd64
COPY linux/amd64/envoy-gateway /usr/local/bin/ # buildkit
COPY --chown=65532:65532 /var/lib /var/lib #
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/envoy-gateway"]
```

which matches https://github.com/envoyproxy/gateway/blob/v1.1.2/tools/docker/envoy-gateway/Dockerfile

So we just need to use and build that Dockerfile.
