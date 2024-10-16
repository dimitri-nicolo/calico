# Copyright 2024 Tigera Inc. All rights reserved.
include ../metadata.mk

PACKAGE_NAME   ?= github.com/projectcalico/calico/compliance-configuration-security

REGO_LIBRARY_RELEASE_VER = v2.0.11
KUBESCAPE_VER = v3.0.15

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile


#############################################
# Generate Controls
#############################################

.PHONY: gen-controls
gen-controls:
	git clone --branch $(REGO_LIBRARY_RELEASE_VER) --single-branch https://github.com/kubescape/regolibrary.git
	$(DOCKER_RUN) $(CALICO_BUILD) go run control-details-scraper/cmd/main.go -writeToPath=./docker-image/control-details-indexer/controls/ -regoPath=./regolibrary 
	rm -rf ./regolibrary
