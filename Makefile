# Copyright 2021 Tigera Inc. All rights reserved.

PACKAGE_NAME    ?= github.com/tigera/deep-packet-inspection
GO_BUILD_VER    ?= v0.65
GIT_USE_SSH      = true
LOCAL_CHECKS     = mod-download

GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go")

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_DEEP_PACKET_INSPECTION_PROJECT_ID)

#############################################
# Env vars related to packaging and releasing
#############################################
DEEP_PACKET_INSPECTION_IMAGE			?=tigera/deep-packet-inspection
SNORT_IMAGE      				?=tigera/snort3
SNORT3_VERSION      				?=3.1.21.0
BUILD_IMAGES					?=$(DEEP_PACKET_INSPECTION_IMAGE)
ARCHES             				?=amd64
DEV_REGISTRIES     				?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES 				?=quay.io
RELEASE_BRANCH_PREFIX 			?= release-calient
DEV_TAG_SUFFIX       		 	?= calient-0.dev

# Mount Semaphore configuration files.
ifdef ST_MODE
EXTRA_DOCKER_ARGS = -v /var/run/docker.sock:/var/run/docker.sock -v /tmp:/tmp:rw -v /home/runner/config:/home/runner/config:rw -v /home/runner/docker_auth.json:/home/runner/config/docker_auth.json:rw
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

###############################################################################
# Download and include Makefile.common
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

include Makefile.common

###############################################################################
# Env vars related to building
###############################################################################
BUILD_VERSION         ?= $(shell git describe --tags --dirty --always 2>/dev/null)
BUILD_BUILD_DATE      ?= $(shell date -u +'%FT%T%z')
BUILD_GIT_DESCRIPTION ?= $(shell git describe --tags 2>/dev/null)
BUILD_GIT_REVISION    ?= $(shell git rev-parse --short HEAD)

# We use -X to insert the version information into the placeholder variables
# in the version package.
VERSION_FLAGS   = -X $(PACKAGE_NAME)/pkg/version.BuildVersion=$(BUILD_VERSION) \
                  -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(BUILD_BUILD_DATE) \
                  -X $(PACKAGE_NAME)/pkg/version.GitDescription=$(BUILD_GIT_DESCRIPTION) \
                  -X $(PACKAGE_NAME)/pkg/version.GitRevision=$(BUILD_GIT_REVISION)

BUILD_LDFLAGS   = -ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS = -ldflags "$(VERSION_FLAGS) -s -w"

###############################################################################
# BUILD BINARY
###############################################################################
# This section builds the output binaries.
build: clean deep-packet-inspection

.PHONY: deep-packet-inspection bin/deep-packet-inspection bin/deep-packet-inspection-$(ARCH)
deep-packet-inspection: bin/deep-packet-inspection

bin/deep-packet-inspection: bin/deep-packet-inspection-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd bin && ln -s -T deep-packet-inspection-$(ARCH) deep-packet-inspection'

bin/deep-packet-inspection-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v $(LDFLAGS) cmd/$*/*.go && \
				( ldd $@ 2>&1 | \
					grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $@ was not statically linked"; false ) )'

###############################################################################
# BUILD IMAGE
###############################################################################
# Build the docker image.
.PHONY: image $(BUILD_IMAGES)

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image $(SNORT_IMAGE) $(DEEP_PACKET_INSPECTION_IMAGE)
image: $(BUILD_IMAGES)
$(SNORT_IMAGE): $(SNORT_IMAGE)-$(ARCH)
$(SNORT_IMAGE)-$(ARCH):
	if docker manifest inspect $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH); then \
  		echo "Using existing snort image $(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH)"; \
  		docker pull $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) ;\
  		docker tag $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) $(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) ;\
  	else \
  	  	echo "Snort image  $(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) doesn't exist in $(DEV_REGISTRIES), building it" ; \
  	  	rm -rf docker-image/bin; \
  	  	mkdir -p docker-image/bin; \
  	  	docker build -t $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) -t $(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) --build-arg SNORT3_VERSION=$(SNORT3_VERSION) --file ./docker-image/Dockerfile.snort.$(ARCH) docker-image; \
  	  	docker tag $(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) ; \
  	fi
ifeq ($(ARCH),amd64)
	docker tag $(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) $(SNORT_IMAGE):$(SNORT3_VERSION)
	docker tag $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH) $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)
endif

$(DEEP_PACKET_INSPECTION_IMAGE): $(SNORT_IMAGE) $(DEEP_PACKET_INSPECTION_IMAGE)-$(ARCH)
$(DEEP_PACKET_INSPECTION_IMAGE)-$(ARCH): bin/deep-packet-inspection-$(ARCH)
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp bin/deep-packet-inspection-$(ARCH) docker-image/bin/
	docker build -t $(DEEP_PACKET_INSPECTION_IMAGE):latest-$(ARCH) --build-arg SNORT3_VERSION=$(SNORT3_VERSION) --file ./docker-image/Dockerfile.$(ARCH) docker-image
ifeq ($(ARCH),amd64)
	docker tag $(DEEP_PACKET_INSPECTION_IMAGE):latest-$(ARCH) $(DEEP_PACKET_INSPECTION_IMAGE):latest
endif

.PHONY: push-snort-image
push-snort-image:
	docker push $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT3_VERSION)-$(ARCH)

.PHONY: clean
clean:
	rm -rf .go-pkg-cache \
		bin \
		docker-image/bin \
		report/*.xml \
		release-notes-* \
		vendor \
		Makefile.common* \
		config/
	find . -name '*.coverprofile' -type f -delete
	docker rmi -f $(DEEP_PACKET_INSPECTION_IMAGE) > /dev/null 2>&1

###############################################################################
# Testing
###############################################################################
MOCKERY_FILE_PATHS= \
	pkg/processor/Processor \
	pkg/exec/Exec \
	pkg/dispatcher/Dispatcher \
	pkg/eventgenerator/EventGenerator \
	pkg/elastic/Client \
	pkg/elastic/ESForwarder \
	pkg/dpiupdater/DPIStatusUpdater \
	pkg/file/FileMaintainer

GINKGO_ARGS += -cover -timeout 20m
GINKGO = ginkgo $(GINKGO_ARGS)

#############################################
# Run unit level tests
#############################################
GINKGO_FOCUS?=Syncer

# Comma separated paths to packages containing fv tests
FV_PACKAGE=pkg/syncer

.PHONY: ut
## Run only Unit Tests.
ut:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod download && $(GINKGO) -r -skipPackage=$(FV_PACKAGE) pkg/*'


###############################################################################
# FV Tests
###############################################################################
## Run the ginkgo FVs
fv: run-k8s-apiserver
	 $(DOCKER_RUN) -e ETCD_ENDPOINTS=http://$(LOCAL_IP_ENV):2379 $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
		ginkgo -r --focus="$(GINKGO_FOCUS)" $(GINKGO_ARGS)'

ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)
ifneq ($(BUILDARCH),amd64)
	ETCD_IMAGE=$(ETCD_IMAGE)-$(ARCH)
endif

# Temporary until hyperkube comes out with a 1.22.1 image to match the KUBE_VERSION env variable
TEMP_HYPERKUBE_VERSION=v1.18.4-rc.0

HYPERKUBE_IMAGE?=gcr.io/google_containers/hyperkube-$(ARCH):$(TEMP_HYPERKUBE_VERSION)
TEST_CONTAINER_FILES=$(shell find tests/ -type f ! -name '*.created' ! -name '*.pyc')

# etcd is used by the FVs
.PHONY: run-etcd
run-etcd:
	@-docker rm -f calico-etcd
	docker run --detach \
	--net=host \
	--name calico-etcd $(ETCD_IMAGE) \
	etcd \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379" \
	--listen-client-urls "http://0.0.0.0:2379"

remote-deps: mod-download
	# Recreate the directory so that we are sure to clean up any old files.
	$(DOCKER_RUN) $(CALICO_BUILD) sh -ec ' \
		$(GIT_CONFIG_SSH) \
		cp -r `go list -mod=mod -m -f "{{.Dir}}" github.com/projectcalico/calico`/libcalico-go/config config; \
		chmod -R +w config/ '

# Kubernetes apiserver used for tests
run-k8s-apiserver: remote-deps stop-k8s-apiserver run-etcd
	docker run \
		--net=host --name st-apiserver \
		-v $(CURDIR):/manifests \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
		--detach \
		${HYPERKUBE_IMAGE} kube-apiserver \
			--bind-address=0.0.0.0 \
			--insecure-bind-address=0.0.0.0 \
			--etcd-servers=http://127.0.0.1:2379 \
			--admission-control=NamespaceLifecycle,LimitRanger,DefaultStorageClass,ResourceQuota \
			--authorization-mode=RBAC \
			--service-cluster-ip-range=10.101.0.0/16 \
			--v=10 \
			--logtostderr=true

	# Wait until we can configure a cluster role binding which allows anonymous auth.
	while ! docker exec st-apiserver kubectl create \
		clusterrolebinding anonymous-admin \
		--clusterrole=cluster-admin \
		--user=system:anonymous 2>/dev/null ; \
		do echo "Waiting for st-apiserver to come up"; \
		sleep 1; \
		done

	# ClusterRoleBinding created

	# Create CustomResourceDefinition (CRD) for Calico resources
	while ! docker exec st-apiserver kubectl \
		apply -f /manifests/config/crd/; \
		do echo "Trying to create CRDs"; \
		sleep 1; \
		done

# Stop Kubernetes apiserver
stop-k8s-apiserver:
	@-docker rm -f st-apiserver

###############################################################################
# Updating pins
###############################################################################
# Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

## Update dependency pins
update-calico-pin:
	$(call update_replace_pin,github.com/projectcalico/calico,github.com/tigera/calico-private,$(PIN_BRANCH))
	$(call update_replace_submodule_pin,github.com/tigera/api,github.com/tigera/calico-private/api,$(PIN_BRANCH))

update-pins: guard-ssh-forwarding-bug update-calico-pin

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

st:

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean static-checks ut fv

## Deploys images to registry
cd: image-all push-snort-image cd-common
