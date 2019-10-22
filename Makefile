# Set repo config.
BUILD_IMAGE?=tigera/fluentd
SRC_DIR?=$(PWD)
# Overwrite configuration, e.g. GCR_REPO:=gcr.io/tigera-dev/experimental/gaurav

default: ci

-include makefile.tigera

# Setup custom target
.PHONY: ci cd
## build cloudwatch plugin initializer
eks-log-forwarder-startup: eks/bin/eks-log-forwarder-startup
eks/bin/eks-log-forwarder-startup:
	$(MAKE) -C eks/

## clean slate cloudwatch plugin initializer
clean-eks-log-forwarder-startup:
	$(MAKE) -C eks/ clean

## test cloudwatch plugin initializer
test-eks-log-forwarder-startup: eks-log-forwarder-startup
	$(MAKE) -C eks/ ut

## fluentd config tests
test: image eks-log-forwarder-startup
	cd $(SRC_DIR)/test && IMAGETAG=$(IMAGETAG) ./test.sh && cd $(SRC_DIR)
	$(MAKE) -C eks/ ut

clean: clean-eks-log-forwarder-startup

## build fluentd image, tagged to IMAGETAG
ci: eks-log-forwarder-startup test image

## push fluentd image to GCR_REPO
cd: push
