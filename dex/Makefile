# Copyright 2024 Tigera Inc. All rights reserved.
include ../metadata.mk

PACKAGE_NAME ?= github.com/projectcalico/calico/dex

#############################################
# Env vars related to packaging and releasing
#############################################
DEX_IMAGE    ?= dex
BUILD_IMAGES ?= $(DEX_IMAGE)

UPSTREAM_DEX_VERSION = v2.37.0

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: bin/dex-$(ARCH)

cmd/dex/%.go:
	mkdir -p cmd/dex
	curl -o $@ https://raw.githubusercontent.com/dexidp/dex/$(UPSTREAM_DEX_VERSION)/$@

init-cmd: clean cmd/dex/config.go cmd/dex/main.go cmd/dex/serve.go cmd/dex/version.go
	patch -d ../ -p1 < patches/0001-cmd-changes.patch

.PHONY: bin/dex-$(ARCH)
bin/dex-$(ARCH): init-cmd
ifeq ($(FIPS),true)
	$(call build_cgo_boring_binary, cmd/dex/*.go, $@)
else
	$(call build_binary, cmd/dex/*.go, $@)
endif

clean:
	rm -rf bin cmd

###############################################################################
# BUILD IMAGE
###############################################################################
DEX_CONTAINER_CREATED=.dex.created-$(ARCH)

.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: $(BUILD_IMAGES)
image: $(DEX_IMAGE)

$(DEX_IMAGE): $(DEX_CONTAINER_CREATED)
$(DEX_CONTAINER_CREATED): bin/dex-$(ARCH)
	$(DOCKER_BUILD) -t $(DEX_IMAGE):latest-$(ARCH) -f docker-image/Dockerfile .
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

ci: image

cd: image-all cd-common
