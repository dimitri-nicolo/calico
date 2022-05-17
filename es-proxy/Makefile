PACKAGE_NAME    ?= github.com/tigera/es-proxy
GO_BUILD_VER    ?= v0.65
GIT_USE_SSH      = true
LOCAL_CHECKS     = mod-download

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_ES_PROXY_IMAGE_PROJECT_ID)

# Mocks auto generated testify mocks by mockery. Run `make gen-mocks` to regenerate the testify mocks.
MOCKERY_FILE_PATHS= \
	pkg/kibana/Client \

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*
# Allow local libcalico-go to be mapped into the build container.
ifdef CALICO_PATH
EXTRA_DOCKER_ARGS += -v $(CALICO_PATH):/go/src/github.com/projectcalico/calico/:ro
endif

# SSH_AUTH_DIR doesn't work with MacOS but we can optionally volume mount keys
ifdef SSH_AUTH_DIR
EXTRA_DOCKER_ARGS += --tmpfs /home/user -v $(SSH_AUTH_DIR):/home/user/.ssh:ro
endif

ifdef LOCAL_BUILD
EXTRA_DOCKER_ARGS += -v $(CURDIR)/../calico-private:/go/src/github.com/tigera/calico-private:rw
local_build:
	go mod edit -replace=github.com/projectcalico/calico=../calico-private
	go mod edit -replace=github.com/tigera/api=../calico-private/api
else
local_build:
endif

TESLA ?= false

ES_PROXY_IMAGE        ?=tigera/es-proxy
BUILD_IMAGES          ?=$(ES_PROXY_IMAGE)
DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

ifeq ($(TESLA),true)
	RELEASE_REGISTRIES    = gcr.io/tigera-tesla
	BUILD_TAGS            ?= -tags tesla
	RELEASE_BRANCH_PREFIX = release-cloud
	DEV_TAG_SUFFIX        = cloud-0.dev
	IMAGETAG_PREFIX       ?= tesla
endif

##############################################################################
# Download and include Makefile.common before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
MAKE_BRANCH?=$(GO_BUILD_VER)
MAKE_REPO?=https://raw.githubusercontent.com/projectcalico/go-build/$(MAKE_BRANCH)

Makefile.common: Makefile.common.$(MAKE_BRANCH)
	cp "$<" "$@"
Makefile.common.$(MAKE_BRANCH):
	# Clean up any files downloaded from other branches so they don't accumulate.
	rm -f Makefile.common.*
	curl --fail $(MAKE_REPO)/Makefile.common -o "$@"

include Makefile.common

build: local_build es-proxy

ETCD_VERSION?=v3.3.7
# If building on amd64 omit the arch in the container name.  Fixme!
ETCD_IMAGE?=quay.io/coreos/etcd:$(ETCD_VERSION)

K8S_VERSION?=v1.11.3
HYPERKUBE_IMAGE?=gcr.io/google_containers/hyperkube-$(ARCH):$(K8S_VERSION)

ELASTICSEARCH_VERSION?=7.16.2
ELASTICSEARCH_IMAGE?=docker.elastic.co/elasticsearch/elasticsearch:$(ELASTICSEARCH_VERSION)

K8S_VERSION    = v1.11.3
BINDIR        ?= bin
BUILD_DIR     ?= build
TOP_SRC_DIRS   = pkg
SRC_DIRS       = $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*.go \
                   -exec dirname {} \\; | sort | uniq")
TEST_DIRS     ?= $(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go \
                   -exec dirname {} \\; | sort | uniq")
GO_FILES       = $(shell sh -c "find pkg cmd -name \\*.go")
ifeq ($(shell uname -s),Darwin)
STAT           = stat -f '%c %N'
else
STAT           = stat -c '%Y %n'
endif
ifdef UNIT_TESTS
UNIT_TEST_FLAGS=-run $(UNIT_TESTS) -v
endif

ES_PROXY_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12)
ES_PROXY_BUILD_DATE?=$(shell date -u +'%FT%T%z')
ES_PROXY_GIT_COMMIT?=$(shell git rev-parse --short HEAD)
ES_PROXY_GIT_TAG?=$(shell git describe --tags)

VERSION_FLAGS=-X $(PACKAGE_NAME)/pkg/handler.VERSION=$(ES_PROXY_VERSION) \
	-X $(PACKAGE_NAME)/pkg/handler.BUILD_DATE=$(ES_PROXY_BUILD_DATE) \
	-X $(PACKAGE_NAME)/pkg/handler.GIT_TAG=$(ES_PROXY_GIT_TAG) \
	-X $(PACKAGE_NAME)/pkg/handler.GIT_COMMIT=$(ES_PROXY_GIT_COMMIT) \
	-X main.VERSION=$(ES_PROXY_VERSION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

###############################################################################
# This section builds the output binaries.
# Some will have dedicated targets to make it easier to type, for example
# "es-proxy" instead of "$(BINDIR)/es-proxy".
es-proxy: $(BINDIR)/es-proxy

$(BINDIR)/es-proxy: $(BINDIR)/es-proxy-amd64
	$(DOCKER_GO_BUILD) \
		sh -c 'cd $(BINDIR) && ln -sf es-proxy-$(ARCH) es-proxy'

$(BINDIR)/es-proxy-$(ARCH): $(GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v -x $(LDFLAGS) $(BUILD_TAGS) "$(PACKAGE_NAME)/cmd/server" && \
				( ldd $(BINDIR)/es-proxy-$(ARCH) 2>&1 | \
	                grep -q -e "Not a valid dynamic program" -e "not a dynamic executable" || \
				( echo "Error: $(BINDIR)/es-proxy-$(ARCH) was not statically linked"; false ) )'

# Build the docker image.
.PHONY: $(BUILD_IMAGES) $(addsuffix -$(ARCH),$(BUILD_IMAGES))

# by default, build the image for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(ARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

image: $(ES_PROXY_IMAGE)
$(ES_PROXY_IMAGE): $(ES_PROXY_IMAGE)-$(ARCH)
$(ES_PROXY_IMAGE)-$(ARCH): $(BINDIR)/es-proxy-$(ARCH)
	docker build --pull -t $(ES_PROXY_IMAGE):latest-$(ARCH) --file ./Dockerfile.$(ARCH) .
ifeq ($(ARCH),amd64)
	docker tag $(ES_PROXY_IMAGE):latest-$(ARCH) $(ES_PROXY_IMAGE):latest
endif

##########################################################################
# Testing
##########################################################################
report-dir:
	mkdir -p report

.PHONY: ut
ut: report-dir
	$(DOCKER_GO_BUILD) \
		sh -c 'git config --global url.ssh://git@github.com.insteadOf https://github.com && \
			go test $(UNIT_TEST_FLAGS) \
			$(addprefix $(PACKAGE_NAME)/,$(TEST_DIRS))'

.PHONY: fv
fv: signpost image report-dir run-k8s-apiserver
	$(MAKE) fv-no-setup

## Developer friendly target to only run fvs and skip other
## setup steps.
.PHONY: fv-no-setup
fv-no-setup:
	PACKAGE_ROOT=$(CURDIR) \
		       GO_BUILD_IMAGE=$(CALICO_BUILD) \
		       PACKAGE_NAME=$(PACKAGE_NAME) \
		       GINKGO_ARGS='$(GINKGO_ARGS)' \
		       FV_ELASTICSEARCH_IMAGE=$(ELASTICSEARCH_IMAGE) \
		       GOMOD_CACHE=$(GOMOD_CACHE) \
		       ./test/run_test.sh

.PHONY: clean
clean: stop-k8s-apiserver stop-etcd stop-elasticsearch
	-docker rmi -f $(ES_PROXY_IMAGE) > /dev/null 2>&1
	-rm -rf $(BINDIR) .go-pkg-cache Makefile.common*


.PHONY: signpost
signpost:
	@echo "------------------------------------------------------------------------------"

###############################################################################
# Static checks
###############################################################################
# See .golangci.yml for golangci-lint config
# SA1019 are deprecation checks, we don't want to fail on those because it means a library update that deprecates something
# requires immediate removing of the deprecated functions.
LINT_ARGS += --exclude SA1019

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci cd

## run CI cycle - build, test, etc.
## Run UTs and only if they pass build image and continue along.
## Building the image is required for fvs.
ci: clean image-all static-checks ut fv

## Deploys images to registry
cd: image-all cd-common

###############################################################################
# Update pins
###############################################################################
# Guard so we don't run this on osx because of ssh-agent to docker forwarding bug
guard-ssh-forwarding-bug:
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "ERROR: This target requires ssh-agent to docker key forwarding and is not compatible with OSX/Mac OS"; \
		echo "$(MAKECMDGOALS)"; \
		exit 1; \
	fi;

COMPLIANCE_BRANCH?=$(PIN_BRANCH)
COMPLIANCE_REPO?=github.com/tigera/compliance

update-calico-pin:
	$(call update_replace_pin,github.com/projectcalico/calico,github.com/tigera/calico-private,$(PIN_BRANCH))
	$(call update_replace_submodule_pin,github.com/tigera/api,github.com/tigera/calico-private/api,$(PIN_BRANCH))

update-compliance-pin:
	$(call update_pin,$(COMPLIANCE_REPO),$(COMPLIANCE_REPO),$(COMPLIANCE_BRANCH))

## Update dependency pins
update-pins: guard-ssh-forwarding-bug update-calico-pin update-compliance-pin

###############################################################################
# Utilities
###############################################################################
LOCAL_IP_ENV?=$(shell ip route get 8.8.8.8 | head -1 | awk '{print $$7}')

# etcd is used by the FVs
.PHONY: run-etcd
run-etcd: stop-etcd
	@-docker rm -f calico-etcd
	docker run --detach \
		--net=host \
		--name calico-etcd $(ETCD_IMAGE) \
		etcd \
		--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379" \
		--listen-client-urls "http://0.0.0.0:2379"

stop-etcd:
	@-docker rm -f calico-etcd


# Kubernetes apiserver used for FVs
.PHONY: run-k8s-apiserver
run-k8s-apiserver: stop-k8s-apiserver run-etcd
	docker run \
		--net=host --name st-apiserver \
		-v  $(CURDIR)/test:/test\
		--detach \
		${HYPERKUBE_IMAGE} \
		/hyperkube apiserver \
			--bind-address=0.0.0.0 \
			--insecure-bind-address=0.0.0.0 \
			--etcd-servers=http://127.0.0.1:2379 \
			--admission-control=NamespaceLifecycle,LimitRanger,DefaultStorageClass,ResourceQuota \
			--authorization-mode=RBAC \
			--service-cluster-ip-range=10.101.0.0/16 \
			--v=10 \
			--anonymous-auth=true \
			--logtostderr=true

	# Wait until we can configure a cluster role binding which allows anonymous auth.
	while ! docker exec st-apiserver kubectl create \
		clusterrolebinding anonymous-admin \
		--clusterrole=cluster-admin \
		--user=system:anonymous; \
		do echo "Trying to create ClusterRoleBinding"; \
		sleep 3; \
		done

	test/setup_k8s_auth.sh

# Stop Kubernetes apiserver
stop-k8s-apiserver:
	@-docker rm -f st-apiserver

# Stop Elasticsearch
stop-elasticsearch:
	@-docker rm -f $$(docker ps -aq --filter ancestor=$(ELASTICSEARCH_IMAGE))

###############################################################################
# Utils
###############################################################################
# this is not a linked target, available for convenience.
.PHONY: tidy
## 'tidy' go modules.
tidy:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod tidy'
