# Set repo config.
BUILD_IMAGE?=tigera/fluentd-docker
# Overwrite configuration, e.g. GCR_REPO:=gcr.io/tigera-dev/experimental/gaurav

default: ci

-include makefile.tigera

# Setup custom target
.PHONY: ci cd
## build fluentd image, tagged to IMAGETAG
ci: image

## push fluentd image to GCR_REPO
cd: push
