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
	curl -sSfL --retry 5 -o $@ https://raw.githubusercontent.com/dexidp/dex/$(UPSTREAM_DEX_VERSION)/$@

# Commit the changes made by init-cmd if the tree is dirty
init-cmd: clean cmd/dex/config.go cmd/dex/main.go cmd/dex/serve.go cmd/dex/version.go
	patch -p2 < patches/0001-cmd-changes.patch
	$(MAKE) mod-tidy

.PHONY: bin/dex-$(ARCH)
bin/dex-$(ARCH): init-cmd
ifeq ($(FIPS),true)
	$(call build_cgo_boring_binary, cmd/dex/*.go, $@)
else
	$(call build_binary, cmd/dex/*.go, $@)
endif

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

###############################################################################
# Misc.
###############################################################################

clean:
	rm -rf bin cmd $(DEX_CONTAINER_CREATED)-*
	$(foreach arch, $(VALIDARCHES), docker rmi -f $(DEX_IMAGE):latest-$(arch);)
	$(foreach arch, $(VALIDARCHES), docker rmi -f tigera/$(DEX_IMAGE):latest-$(arch);)
