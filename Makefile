PACKAGE_NAME    ?= github.com/tigera/calicoq
GO_BUILD_VER    ?= v0.32
GOMOD_VENDOR     = true
GIT_USE_SSH      = true
LIBCALICO_REPO   = github.com/tigera/libcalico-go-private
FELIX_REPO       = github.com/tigera/felix-private

build: ut-containerized

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

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

# Allow local libcalico-go to be mapped into the build container.
ifdef LIBCALICOGO_PATH
EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif

include Makefile.common

##############################################################################
# Define some constants
##############################################################################
BUILD_VER?=latest
BUILD_IMAGE:=tigera/calicoq
REGISTRY_PREFIX?=gcr.io/unique-caldron-775/cnx/
BINARY:=bin/calicoq

CALICOQ_VERSION?=$(shell git describe --tags --dirty --always)
CALICOQ_BUILD_DATE?=$(shell date -u +'%FT%T%z')
CALICOQ_GIT_DESCRIPTION?=$(shell git describe --tags)
CALICOQ_GIT_REVISION?=$(shell git rev-parse --short HEAD)

VERSION_FLAGS=-X $(PACKAGE_NAME)/calicoq/commands.VERSION=$(CALICOQ_VERSION) \
	-X $(PACKAGE_NAME)/calicoq/commands.BUILD_DATE=$(CALICOQ_BUILD_DATE) \
	-X $(PACKAGE_NAME)/calicoq/commands.GIT_DESCRIPTION=$(CALICOQ_GIT_DESCRIPTION) \
	-X $(PACKAGE_NAME)/calicoq/commands.GIT_REVISION=$(CALICOQ_GIT_REVISION)
BUILD_LDFLAGS=-ldflags "$(VERSION_FLAGS)"
RELEASE_LDFLAGS=-ldflags "$(VERSION_FLAGS) -s -w"

# Create an extended go-build image with docker binary installed for use with st-containerized target
TOOLING_IMAGE?=calico/go-build-with-docker
TOOLING_IMAGE_VERSION?=v0.24
TOOLING_IMAGE_CREATED=.go-build-with-docker.created

$(TOOLING_IMAGE_CREATED): Dockerfile-testenv.amd64
	docker build --cpuset-cpus 0 --pull -t $(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) -f Dockerfile-testenv.amd64 .
	touch $@

vendor: go.mod mod-download
	$(DOCKER_RUN) $(CALICO_BUILD) \
	    sh -c '$(GIT_CONFIG_SSH) go mod vendor -v'

.PHONY: ut ut-containerized
ut: bin/calicoq
	ginkgo -cover -r --skipPackage vendor calicoq/*

	@echo
	@echo '+==============+'
	@echo '| All coverage |'
	@echo '+==============+'
	@echo
	@find ./calicoq/ -iname '*.coverprofile' | xargs -I _ go tool cover -func=_

	@echo
	@echo '+==================+'
	@echo '| Missing coverage |'
	@echo '+==================+'
	@echo
	@find ./calicoq/ -iname '*.coverprofile' | xargs -I _ go tool cover -func=_ | grep -v '100.0%'

ut-containerized: bin/calicoq
	docker run --rm -t \
		-v $(CURDIR):/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		$(CALICO_BUILD) \
		sh -c 'make ut'

.PHONY: fv fv-containerized
fv: bin/calicoq
	CALICOQ=`pwd`/$^ fv/run-test

fv-containerized: build-image run-etcd
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(CALICO_BUILD) \
		-c 'CALICOQ=`pwd`/$(BINARY) fv/run-test'

.PHONY: st st-containerized
st: bin/calicoq
	CALICOQ=`pwd`/$^ st/run-test

st-containerized: build-image $(TOOLING_IMAGE_CREATED)
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) \
		-c 'CALICOQ=`pwd`/$(BINARY) st/run-test'

.PHONY: scale-test scale-test-containerized
scale-test: bin/calicoq
	CALICOQ=`pwd`/$^ scale-test/run-test

scale-test-containerized: build-image
	docker run --net=host --privileged \
		--rm -t \
		--entrypoint '/bin/sh' \
		-v $(CURDIR):/code/$(PACKAGE_NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /code/$(PACKAGE_NAME) \
		$(CALICO_BUILD) \
		-c 'CALICOQ=`pwd`/$(BINARY) scale-test/run-test'

# Build image for containerized testing
.PHONY: build-image
build-image: bin/calicoq
	docker build -t $(BUILD_IMAGE):$(BUILD_VER) `pwd`

# Clean up image from containerized testing
.PHONY: clean-image
clean-image:
	docker rmi -f $(shell docker images -a | grep $(BUILD_IMAGE) | awk '{print $$3}' | awk '!a[$$0]++')

# All calicoq Go source files.
CALICOQ_GO_FILES:=$(shell find calicoq -type f -name '*.go' -print)

bin/calicoq:
	$(MAKE) binary-containerized

.PHONY: binary-containerized
binary-containerized: $(CALICOQ_GO_FILES)
ifndef RELEASE_BUILD
	$(eval LDFLAGS:=$(RELEASE_LDFLAGS))
else
	$(eval LDFLAGS:=$(BUILD_LDFLAGS))
endif
	mkdir -p .go-pkg-cache bin $(GOMOD_CACHE)
	$(MAKE) vendor
	# Generate the protobuf bindings for Felix
	# Cannot do this together with vendoring since docker permissions in go-build are not perfect?
	$(MAKE) felixbackend
	# Create the binary
	$(DOCKER_RUN) $(CALICO_BUILD) \
	   sh -c '$(GIT_CONFIG_SSH) go build -v $(LDFLAGS) -o "$(BINARY)" "./calicoq/calicoq.go"'

imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

tag-image: imagetag build-image
	docker tag $(BUILD_IMAGE):latest $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(IMAGETAG)

push-image: imagetag tag-image
	docker push $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(IMAGETAG)

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


LICENSING_REPO=github.com/tigera/licensing
LICENSING_BRANCH=$(PIN_BRANCH)
LOGRUS_REPO_ORIG=github.com/sirupsen/logrus
LOGRUS_REPO=github.com/projectcalico/logrus
LOGRUS_BRANCH=$(PIN_BRANCH)

update-licensing-pin:
	$(call update_pin,$(LICENSING_REPO),$(LICENSING_REPO),$(LICENSING_BRANCH))

replace-logrus-pin:
	$(call update_replace_pin,$(LOGRUS_REPO_ORIG),$(LOGRUS_REPO),$(LOGRUS_BRANCH))

update-pins: guard-ssh-forwarding-bug replace-libcalico-pin replace-felix-pin update-licensing-pin replace-logrus-pin

###############################################################################
# TODO: re-enable these linters!
LINT_ARGS := --disable gosimple,govet,structcheck,errcheck,goimports,unused,ineffassign,staticcheck,deadcode,typecheck
LINT_ARGS += --timeout 5m

###############################################################################
# CI/CD
###############################################################################
.PHONY: ci
## Run what CI runs
ci: clean static-checks fv-containerized ut-containerized st-containerized

## Deploys images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) push-image IMAGETAG=${BRANCH_NAME}


.PHONY: release
release: clean clean-release release/calicoq

release/calicoq: $(CALICOQ_GO_FILES) clean
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=v.X.Y.Z)
endif
	git tag $(VERSION)

	# Check to make sure the tag isn't "dirty"
	if git describe --tags --dirty | grep dirty; \
	then echo current git working tree is "dirty". Make sure you do not have any uncommitted changes ;false; fi

	# Build the calicoq binaries and image
	$(MAKE) binary-containerized RELEASE_BUILD=1
	$(MAKE) build-image

	# Make the release directory and move over the relevant files
	mkdir -p release
	mv $(BINARY) release/calicoq-$(CALICOQ_GIT_DESCRIPTION)
	ln -f release/calicoq-$(CALICOQ_GIT_DESCRIPTION) release/calicoq

	# Check that the version output includes the version specified.
	# Tests that the "git tag" makes it into the binaries. Main point is to catch "-dirty" builds
	# Release is currently supported on darwin / linux only.
	if ! docker run $(BUILD_IMAGE) version | grep 'Version:\s*$(VERSION)$$'; then \
	  echo "Reported version:" `docker run $(BUILD_IMAGE) version` "\nExpected version: $(VERSION)"; \
	  false; \
	else \
	  echo "Version check passed\n"; \
	fi

	# Retag images with correct version and registry prefix
	docker tag $(BUILD_IMAGE) $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(VERSION)

	# Check that images were created recently and that the IDs of the versioned and latest images match
	@docker images --format "{{.CreatedAt}}\tID:{{.ID}}\t{{.Repository}}:{{.Tag}}" $(BUILD_IMAGE)
	@docker images --format "{{.CreatedAt}}\tID:{{.ID}}\t{{.Repository}}:{{.Tag}}" $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(VERSION)

	@echo "\nNow push the tag and images."
	@echo "git push origin $(VERSION)"
	@echo "gcloud auth configure-docker"
	@echo "docker push $(REGISTRY_PREFIX)$(BUILD_IMAGE):$(VERSION)"
	@echo "\nIf this release version is the newest stable release, also tag and push the"
	@echo "images with the 'latest' tag"
	@echo "docker tag $(BUILD_IMAGE) $(REGISTRY_PREFIX)$(BUILD_IMAGE):latest"
	@echo "docker push $(REGISTRY_PREFIX)$(BUILD_IMAGE):latest"

.PHONY: compress-release
compressed-release: release/calicoq
	# Requires "upx" to be in your PATH.
	# Compress the executable with upx.  We get 4:1 compression with '-8'; the
	# more agressive --best gives a 0.5% improvement but takes several minutes.
	upx -8 release/calicoq-$(CALICOQ_GIT_DESCRIPTION)
	ln -f release/calicoq-$(CALICOQ_GIT_DESCRIPTION) release/calicoq

# Generate the protobuf bindings for Felix.
.PHONY: felixbackend
felixbackend: vendor/github.com/projectcalico/felix/proto/felixbackend.proto
	docker run --rm -v `pwd`/vendor/github.com/projectcalico/felix/proto:/src:rw \
	              calico/protoc \
	              --gogofaster_out=. \
	              felixbackend.proto

## Run etcd as a container (calico-etcd)
run-etcd: stop-etcd
	docker run --detach \
	--net=host \
	--entrypoint=/usr/local/bin/etcd \
	--name calico-etcd quay.io/coreos/etcd:v3.1.7 \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379,http://$(LOCAL_IP_ENV):4001,http://127.0.0.1:4001" \
	--listen-client-urls "http://0.0.0.0:2379,http://0.0.0.0:4001"

## Stop the etcd container (calico-etcd)
stop-etcd:
	-docker rm -f calico-etcd

.PHONY: clean-release
clean-release:
	-rm -rf release

.PHONY: clean
clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build bin release vendor
	-docker rmi calico/build
	-docker rmi $(BUILD_IMAGE) -f
	-docker rmi $(CALICO_BUILD) -f
	-docker rmi $(TOOLING_IMAGE):$(TOOLING_IMAGE_VERSION) -f
	rm -f $(TOOLING_IMAGE_CREATED)
