# Envoy Log Collector

This implementation and code is largely taken from the implementation of [ingress-collector](https://github.com/tigera/ingress-collector). 
This collector runs in the same container as the envoy sidecar (another sidecar), collects logs written to a file specified in the config 
and sends them to felix through a socket using gRPC.

# Build and Testing

To build the image use

            make image
            
The image expects a logs file to be present in the location as specified in the config. To run the container locally create a dummy log file in the location specified in the config.

            docker run -v <log-file-location-in-config>:dummy.log <image-hash>
