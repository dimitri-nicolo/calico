# Set repo config.
BUILD_IMAGE?=tigera/fluentd
SRC_DIR?=$(PWD)
# Overwrite configuration, e.g. GCR_REPO:=gcr.io/tigera-dev/experimental/gaurav

default: ci

-include makefile.tigera

# Setup custom target
.PHONY: ci cd
## simple tests
test: image
	cd $(SRC_DIR)/test && IMAGETAG=$(IMAGETAG) ./test.sh && cd $(SRC_DIR)

## build fluentd image, tagged to IMAGETAG
ci: image test

## push fluentd image to GCR_REPO
cd: push
