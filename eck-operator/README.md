# Tigera ECK-Operator

Upstream ECK-Operator for Tigera. This library modifies the image to remove CVEs and fix bugs.
The original image is built from: https://github.com/elastic/cloud-on-k8s/blob/1.8/Dockerfile

### Building the image

Run `make image` to create the image, run `make compressed-image` to create an image with the extra layers removed.

### Releasing / Deploying on ECK

The image released must be the compressed image, e.g. the one created by running `make compressed-image`. The readiness 
probe for ECK must be changed to use the one added to the image at /readiness-probe.