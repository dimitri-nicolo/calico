PACKAGE_NAME            ?= github.com/tigera/compliance
GO_BUILD_VER            ?= v0.32
GOMOD_VENDOR             = false
GIT_USE_SSH              = true
LIBCALICO_REPO           = github.com/tigera/libcalico-go-private
FELIX_REPO               = github.com/tigera/felix-private
LIBCALICO_GO_VERSION    ?= v0.0.0-20200110190915-9fa812d46e44

build: ut

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

# Build mounts for running in "local build" mode. Developers will need to make sure they have the correct local version
# otherwise the build will fail.
PHONY:local_build
# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/github.com/tigera/libcalico-go:ro
endif
ifdef LOCAL_BUILD
EXTRA_DOCKER_ARGS += -v $(CURDIR)/../libcalico-go:/go/src/github.com/tigera/libcalico-go:rw
local_build:
	go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go
else
local_build:
	-go mod edit -replace=github.com/projectcalico/libcalico-go=github.com/tigera/libcalico-go-private@$(LIBCALICO_GO_VERSION)
endif
ifdef GOPATH
EXTRA_DOCKER_ARGS += -v $(GOPATH)/pkg/mod:/go/pkg/mod:rw
endif


include Makefile.common

##############################################################################
# Define some constants
##############################################################################
ELASTIC_VERSION			?= 7.3.2
K8S_VERSION     		?= v1.11.3
ETCD_VERSION			?= v3.3.7
KUBE_BENCH_VERSION		?= b649588f46c54c84cd9c88510680b5a651f12d46

# Override VALIDARCHES inferenced in common Makefile.
#   This repo differs in how ARCHES are determined compared to common logic.
#   overriding the value with the only platform supported ATM.
VALIDARCHES = amd64

BUILD_IMAGE_SERVER=tigera/compliance-server
BUILD_IMAGE_CONTROLLER=tigera/compliance-controller
BUILD_IMAGE_SNAPSHOTTER=tigera/compliance-snapshotter
BUILD_IMAGE_REPORTER=tigera/compliance-reporter
BUILD_IMAGE_SCALELOADER=tigera/compliance-scaleloader
BUILD_IMAGE_BENCHMARKER=tigera/compliance-benchmarker
GCR_REPO?=gcr.io/unique-caldron-775/cnx

PUSH_IMAGE_PREFIXES?=$(GCR_REPO)/
RELEASE_IMAGES?=
# If this is a release, also tag and push additional images.
ifeq ($(RELEASE),true)
PUSH_IMAGE_PREFIXES+=$(RELEASE_IMAGES)
endif

# remove from the list to push to manifest any registries that do not support multi-arch
# EXCLUDE_MANIFEST_REGISTRIES defined in Makefile.comm
PUSH_MANIFEST_IMAGE_PREFIXES=$(PUSH_IMAGE_PREFIXES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGE_PREFIXES=$(filter-out $(PUSH_MANIFEST_IMAGE_PREFIXES),$(PUSH_IMAGE_PREFIXES))

# Figure out version information.  To support builds from release tarballs, we default to
# <unknown> if this isn't a git checkout.
PKG_VERSION?=$(shell git describe --tags --dirty --always || echo '<unknown>')
PKG_VERSION_BUILD_DATE?=$(shell date -u +'%FT%T%z' || echo '<unknown>')
PKG_VERSION_GIT_DESCRIPTION?=$(shell git describe --tags 2>/dev/null || echo '<unknown>')
PKG_VERSION_GIT_REVISION?=$(shell git rev-parse --short HEAD || echo '<unknown>')

# Linker flags for building Compliance Server.
#
# We use -X to insert the version information into the placeholder variables
# in the buildinfo package.
#
# We use -B to insert a build ID note into the executable, without which, the
# RPM build tools complain.
LDFLAGS:=-ldflags "\
		-X $(PACKAGE_NAME)/pkg/version.VERSION=$(PKG_VERSION) \
		-X $(PACKAGE_NAME)/pkg/version.BUILD_DATE=$(PKG_VERSION_BUILD_DATE) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_DESCRIPTION=$(PKG_VERSION_GIT_DESCRIPTION) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_REVISION=$(PKG_VERSION_REVISION) \
		-B 0x$(BUILD_ID)"

NON_SRC_DIRS = test
# All Compliance Server go files.
SRC_FILES:=$(shell find . $(foreach dir,$(NON_SRC_DIRS),-path ./$(dir) -prune -o) -type f -name '*.go' -print)

.PHONY: clean
clean:
	rm -rf bin \
	       docker-image/server/bin \
	       docker-image/controller/bin \
	       docker-image/snapshotter/bin \
	       docker-image/reporter/bin \
	       docker-image/scaleloader/bin \
	       docker-image/benchmarker/bin \
	       docker-image/benchmarker/cfg \
	       tmp/kube-bench \
	       release-notes-* \
	       .go-pkg-cache
	find . -name "*.coverprofile" -type f -delete
	find . -name "coverage.xml" -type f -delete
	find . -name ".coverage" -type f -delete
	find . -name "*.pyc" -type f -delete

###############################################################################
# Building the binary
###############################################################################
build: bin/server bin/controller bin/snapshotter bin/reporter bin/report-type-gen bin/benchmarker
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

bin/server: bin/server-$(ARCH)
	ln -f bin/server-$(ARCH) bin/server

bin/server-$(ARCH): $(SRC_FILES) local_build
	@echo Building compliance-server...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/server" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/server was not statically linked"; false ) )'

bin/controller: bin/controller-$(ARCH)
	ln -f bin/controller-$(ARCH) bin/controller

bin/controller-$(ARCH): $(SRC_FILES) local_build
	@echo Building compliance controller...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/controller" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/controller was not statically linked"; false ) )'

bin/snapshotter: bin/snapshotter-$(ARCH)
	ln -f bin/snapshotter-$(ARCH) bin/snapshotter

bin/snapshotter-$(ARCH): $(SRC_FILES) local_build
	@echo Building compliance snapshotter...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/snapshotter" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/snapshotter was not statically linked"; false ) )'

bin/reporter: bin/reporter-$(ARCH)
	ln -f bin/reporter-$(ARCH) bin/reporter

bin/reporter-$(ARCH): $(SRC_FILES) local_build
	@echo Building compliance reporter...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/reporter" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/reporter was not statically linked"; false ) )'

bin/report-type-gen: bin/report-type-gen-$(ARCH)
	ln -f bin/report-type-gen-$(ARCH) bin/report-type-gen

bin/report-type-gen-$(ARCH): $(SRC_FILES) local_build
	@echo Building report type generator...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/report-type-gen" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/report-type-gen was not statically linked"; false ) )'

bin/scaleloader: bin/scaleloader-$(ARCH)
	ln -f bin/scaleloader-$(ARCH) bin/scaleloader

bin/scaleloader-$(ARCH): $(SRC_FILES) local_build
	@echo Building scaleloader...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/mockdata-scaleloader" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/scaleloader was not statically linked"; false ) )'

bin/benchmarker: bin/benchmarker-$(ARCH)
	ln -f bin/benchmarker-$(ARCH) bin/benchmarker

bin/benchmarker-$(ARCH): $(SRC_FILES) local_build
	@echo Building benchmarker...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/benchmarker" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/benchmarker was not statically linked"; false ) )'

###############################################################################
# Building the images
###############################################################################
.PHONY: $(BUILD_IMAGE_SERVER) $(BUILD_IMAGE_SERVER)-$(ARCH)
.PHONY: $(BUILD_IMAGE_CONTROLLER) $(BUILD_IMAGE_CONTROLLER)-$(ARCH)
.PHONY: $(BUILD_IMAGE_SNAPSHOTTER) $(BUILD_IMAGE_SNAPSHOTTER)-$(ARCH)
.PHONY: $(BUILD_IMAGE_REPORTER) $(BUILD_IMAGE_REPORTER)-$(ARCH)
.PHONY: $(BUILD_IMAGE_SCALELOADER) $(BUILD_IMAGE_SCALELOADER)-$(ARCH)
.PHONY: $(BUILD_IMAGE_BENCHMARKER) $(BUILD_IMAGE_BENCHMARKER)-$(ARCH)
.PHONY: images
.PHONY: image

images image: $(BUILD_IMAGE_SERVER) $(BUILD_IMAGE_CONTROLLER) $(BUILD_IMAGE_SNAPSHOTTER) $(BUILD_IMAGE_REPORTER) $(BUILD_IMAGE_SCALELOADER) $(BUILD_IMAGE_BENCHMARKER)

# Build the images for the target architecture
.PHONY: images-all
images-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) images ARCH=$*

# Build the tigera/compliance-server docker image, which contains only Compliance server.
$(BUILD_IMAGE_SERVER): bin/server-$(ARCH) register
	rm -rf docker-image/server/bin
	mkdir -p docker-image/server/bin
	cp bin/server-$(ARCH) docker-image/server/bin/
	docker build --pull -t $(BUILD_IMAGE_SERVER):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/server/Dockerfile.$(ARCH) docker-image/server
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE_SERVER):latest-$(ARCH) $(BUILD_IMAGE_SERVER):latest
endif

# Build the tigera/compliance-controller docker image, which contains only Compliance controller.
$(BUILD_IMAGE_CONTROLLER): bin/controller-$(ARCH) register
	rm -rf docker-image/controller/bin
	mkdir -p docker-image/controller/bin
	cp bin/controller-$(ARCH) docker-image/controller/bin/
	docker build --pull -t $(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/controller/Dockerfile.$(ARCH) docker-image/controller
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) $(BUILD_IMAGE_CONTROLLER):latest
endif

# Build the tigera/compliance-snapshotter docker image, which contains only Compliance snapshotter.
$(BUILD_IMAGE_SNAPSHOTTER): bin/snapshotter-$(ARCH) register
	rm -rf docker-image/snapshotter/bin
	mkdir -p docker-image/snapshotter/bin
	cp bin/snapshotter-$(ARCH) docker-image/snapshotter/bin/
	docker build --pull -t $(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/snapshotter/Dockerfile.$(ARCH) docker-image/snapshotter
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) $(BUILD_IMAGE_SNAPSHOTTER):latest
endif

# Build the tigera/compliance-reporter docker image, which contains only Compliance reporter.
$(BUILD_IMAGE_REPORTER): bin/reporter-$(ARCH) register
	rm -rf docker-image/reporter/bin
	mkdir -p docker-image/reporter/bin
	cp bin/reporter-$(ARCH) docker-image/reporter/bin/
	docker build --pull -t $(BUILD_IMAGE_REPORTER):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/reporter/Dockerfile.$(ARCH) docker-image/reporter
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE_REPORTER):latest-$(ARCH) $(BUILD_IMAGE_REPORTER):latest
endif

# Build the tigera/compliance-scaleloader docker image, which contains only Compliance scaleloader.
$(BUILD_IMAGE_SCALELOADER): bin/scaleloader-$(ARCH) register
	rm -rf docker-image/scaleloader/bin
	rm -rf docker-image/scaleloader/playbooks
	rm -rf docker-image/scaleloader/scenarios
	mkdir -p docker-image/scaleloader/bin
	cp bin/scaleloader-$(ARCH) docker-image/scaleloader/bin/
	cp -r mockdata/scaleloader/playbooks docker-image/scaleloader
	cp -r mockdata/scaleloader/scenarios docker-image/scaleloader
	docker build --pull -t $(BUILD_IMAGE_SCALELOADER):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/scaleloader/Dockerfile.$(ARCH) docker-image/scaleloader
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE_SCALELOADER):latest-$(ARCH) $(BUILD_IMAGE_SCALELOADER):latest
endif

# Build the tigera/compliance-benchmarker docker image, which contains only Compliance benchmarker.
$(BUILD_IMAGE_BENCHMARKER): bin/benchmarker-$(ARCH) register
	rm -rf docker-image/benchmarker/bin
	rm -rf tmp/kube-bench
	mkdir -p docker-image/benchmarker/bin
	cp bin/benchmarker-$(ARCH) docker-image/benchmarker/bin/
	git clone https://github.com/aquasecurity/kube-bench tmp/kube-bench/
	cd tmp/kube-bench/ && git checkout $(KUBE_BENCH_VERSION)
	cp -r tmp/kube-bench/cfg docker-image/benchmarker
	rm -rf tmp/kube-bench
	docker build --pull -t $(BUILD_IMAGE_BENCHMARKER):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/benchmarker/Dockerfile.$(ARCH) docker-image/benchmarker
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE_BENCHMARKER):latest-$(ARCH) $(BUILD_IMAGE_BENCHMARKER):latest
endif

# ensure we have a real imagetag
imagetag:
ifndef IMAGETAG
	$(error IMAGETAG is undefined - run using make <target> IMAGETAG=X.Y.Z)
endif

# we want to be able to run the same recipe on multiple targets keyed on the image name
# to do that, we would use the entire image name, e.g. calico/node:abcdefg, as the stem, or '%', in the target
# however, make does **not** allow the usage of invalid filename characters - like / and : - in a stem, and thus errors out
# to get around that, we "escape" those characters by converting all : to --- and all / to ___ , so that we can use them
# in the target, we then unescape them back
escapefs = $(subst :,---,$(subst /,___,$(1)))
unescapefs = $(subst ---,:,$(subst ___,/,$(1)))

## push one arch
push: imagetag $(addprefix sub-single-push-,$(call escapefs,$(PUSH_IMAGE_PREFIXES)))

sub-single-push-%:
	docker push $(call unescapefs,$*$(BUILD_IMAGE_SERVER):$(IMAGETAG)-$(ARCH))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_CONTROLLER):$(IMAGETAG)-$(ARCH))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_SNAPSHOTTER):$(IMAGETAG)-$(ARCH))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_REPORTER):$(IMAGETAG)-$(ARCH))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_BENCHMARKER):$(IMAGETAG)-$(ARCH))
ifneq ("",$(findstring $(GCR_REPO),$(call unescapefs,$*)))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_SCALELOADER):$(IMAGETAG)-$(ARCH))
endif

## push all archs
push-all: imagetag $(addprefix sub-push-,$(VALIDARCHES))
sub-push-%:
	$(MAKE) push ARCH=$* IMAGETAG=$(IMAGETAG)

push-manifests: imagetag  $(addprefix sub-manifest-,$(call escapefs,$(PUSH_MANIFEST_IMAGE_PREFIXES)))
sub-manifest-%:
	# Docker login to hub.docker.com required before running this target as we are using $(DOCKER_CONFIG) holds the docker login credentials
	# path to credentials based on manifest-tool's requirements here https://github.com/estesp/manifest-tool#sample-usage
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*$(BUILD_IMAGE_SERVER):$(IMAGETAG))-ARCH --target $(call unescapefs,$*$(BUILD_IMAGE_SERVER):$(IMAGETAG))"
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*$(BUILD_IMAGE_CONTROLLER):$(IMAGETAG))-ARCH --target $(call unescapefs,$*$(BUILD_IMAGE_CONTROLLER):$(IMAGETAG))"
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*$(BUILD_IMAGE_SNAPSHOTTER):$(IMAGETAG))-ARCH --target $(call unescapefs,$*$(BUILD_IMAGE_SNAPSHOTTER):$(IMAGETAG))"
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*$(BUILD_IMAGE_REPORTER):$(IMAGETAG))-ARCH --target $(call unescapefs,$*$(BUILD_IMAGE_REPORTER):$(IMAGETAG))"
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*$(BUILD_IMAGE_BENCHMARKER):$(IMAGETAG))-ARCH --target $(call unescapefs,$*$(BUILD_IMAGE_BENCHMARKER):$(IMAGETAG))"
ifneq ("",$(findstring $(GCR_REPO),$(call unescapefs,$*)))
	docker run -t --entrypoint /bin/sh -v $(DOCKER_CONFIG):/root/.docker/config.json $(CALICO_BUILD) -c "/usr/bin/manifest-tool push from-args --platforms $(call join_platforms,$(VALIDARCHES)) --template $(call unescapefs,$*$(BUILD_IMAGE_SCALELOADER):$(IMAGETAG))-ARCH --target $(call unescapefs,$*$(BUILD_IMAGE_SCALELOADER):$(IMAGETAG))"
endif

## push default amd64 arch where multi-arch manifest is not supported
push-non-manifests: imagetag $(addprefix sub-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGE_PREFIXES)))
sub-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker push $(call unescapefs,$*$(BUILD_IMAGE_SERVER):$(IMAGETAG))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_CONTROLLER):$(IMAGETAG))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_SHAPSHOTTER):$(IMAGETAG))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_REPORTER):$(IMAGETAG))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_BENCHMARKER):$(IMAGETAG))
ifneq ("",$(findstring $(GCR_REPO),$(call unescapefs,$*)))
	docker push $(call unescapefs,$*$(BUILD_IMAGE_SCALELOADER):$(IMAGETAG))
endif
else
	$(NOECHO) $(NOOP)
endif

## tag images of one arch
tag-images: imagetag $(addprefix sub-single-tag-images-arch-,$(call escapefs,$(PUSH_IMAGE_PREFIXES))) $(addprefix sub-single-tag-images-non-manifest-,$(call escapefs,$(PUSH_NONMANIFEST_IMAGE_PREFIXES)))
sub-single-tag-images-arch-%:
	docker tag $(BUILD_IMAGE_SERVER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_SERVER):$(IMAGETAG)-$(ARCH))
	docker tag $(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_CONTROLLER):$(IMAGETAG)-$(ARCH))
	docker tag $(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_SNAPSHOTTER):$(IMAGETAG)-$(ARCH))
	docker tag $(BUILD_IMAGE_REPORTER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_REPORTER):$(IMAGETAG)-$(ARCH))
	docker tag $(BUILD_IMAGE_BENCHMARKER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_BENCHMARKER):$(IMAGETAG)-$(ARCH))
ifneq ("",$(findstring $(GCR_REPO),$(call unescapefs,$*)))
	docker tag $(BUILD_IMAGE_SCALELOADER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_SCALELOADER):$(IMAGETAG)-$(ARCH))
endif

# because some still do not support multi-arch manifest
sub-single-tag-images-non-manifest-%:
ifeq ($(ARCH),amd64)
	docker tag $(BUILD_IMAGE_SERVER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_SERVER):$(IMAGETAG))
	docker tag $(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_CONTROLLER):$(IMAGETAG))
	docker tag $(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_SNAPSHOTTER):$(IMAGETAG))
	docker tag $(BUILD_IMAGE_REPORTER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_REPORTER):$(IMAGETAG))
	docker tag $(BUILD_IMAGE_BENCHMARKER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_BENCHMARKER):$(IMAGETAG))
ifneq ("",$(findstring $(GCR_REPO),$(call unescapefs,$*)))
	docker tag $(BUILD_IMAGE_SCALELOADER):latest-$(ARCH) $(call unescapefs,$*$(BUILD_IMAGE_SCALELOADER):$(IMAGETAG))
endif
else
	$(NOECHO) $(NOOP)
endif

## tag images of all archs
tag-images-all: imagetag $(addprefix sub-tag-images-,$(VALIDARCHES))
sub-tag-images-%:
	$(MAKE) tag-images ARCH=$* IMAGETAG=$(IMAGETAG)

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


LMA_REPO=github.com/tigera/lma
LMA_BRANCH=$(PIN_BRANCH)

update-lma-pin:
	$(call update_pin,$(LMA_REPO),$(LMA_REPO),$(LMA_BRANCH))

update-pins: guard-ssh-forwarding-bug replace-libcalico-pin replace-felix-pin update-lma-pin
###############################################################################

###############################################################################
# Static checks
###############################################################################
#TODO: enable linters
LINT_ARGS := --disable gosimple,govet,structcheck,errcheck,goimports,unused,ineffassign,staticcheck,deadcode,typecheck
LINT_ARGS += --timeout 5m

#TODO: Shoud gometalinter be deleted in favor of golangci-lint?
.PHONY: go-meta-linter
go-meta-linter: vendor/.up-to-date $(GENERATED_GO_FILES)
	# Run staticcheck stand-alone since gometalinter runs concurrent copies, which
	# uses a lot of RAM.
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'glide nv | xargs -n 3 staticcheck'
	$(DOCKER_RUN) $(CALICO_BUILD) gometalinter --enable-gc \
		--deadline=300s \
		--disable-all \
		--enable=goimports \
		--enable=errcheck \
		--vendor ./...

###############################################################################
# Tests
###############################################################################
.PHONY: ut
ut combined.coverprofile: run-elastic
	@echo Running Go UTs.
	$(DOCKER_RUN) -e ELASTIC_HOST=localhost $(CALICO_BUILD) ./utils/run-coverage sh -c '$(GIT_CONFIG_SSH)'

## Run elasticsearch as a container (tigera-elastic)
run-elastic: stop-elastic
	# Run ES on Docker.
	docker run --detach \
	--net=host \
	--name=tigera-elastic \
	-e "discovery.type=single-node" \
	docker.elastic.co/elasticsearch/elasticsearch:$(ELASTIC_VERSION)

	# Wait until ES is accepting requests.
	@while ! docker exec tigera-elastic curl localhost:9200 2> /dev/null; do echo "Waiting for Elasticsearch to come up..."; sleep 2; done

## Stop elasticsearch with name tigera-elastic
stop-elastic:
	-docker rm -f tigera-elastic

## Run etcd as a container (calico-etcd)
run-etcd: stop-etcd
	docker run --detach \
	--net=host \
	--entrypoint=/usr/local/bin/etcd \
	--name calico-etcd quay.io/coreos/etcd:$(ETCD_VERSION) \
	--advertise-client-urls "http://$(LOCAL_IP_ENV):2379,http://127.0.0.1:2379,http://$(LOCAL_IP_ENV):4001,http://127.0.0.1:4001" \
	--listen-client-urls "http://0.0.0.0:2379,http://0.0.0.0:4001"

## Stop the etcd container (calico-etcd)
stop-etcd:
	-docker rm -f calico-etcd


## Run a local kubernetes master with API via hyperkube
run-kubernetes-master: stop-kubernetes-master
	# Run a Kubernetes apiserver using Docker.
	docker run \
		--net=host --name st-apiserver \
		-v  $(CURDIR)/test:/test\
		--detach \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
		/hyperkube apiserver \
			--bind-address=0.0.0.0 \
			--insecure-bind-address=0.0.0.0 \
			--etcd-servers=http://127.0.0.1:2379 \
			--admission-control=NamespaceLifecycle,LimitRanger,DefaultStorageClass,ResourceQuota \
			--authorization-mode=RBAC \
			--service-cluster-ip-range=10.101.0.0/16 \
			--v=10 \
			--token-auth-file=/test/rbac/token_auth.csv \
			--basic-auth-file=/test/rbac/basic_auth.csv \
			--anonymous-auth=true \
			--logtostderr=true

	# Wait until the apiserver is accepting requests.
	while ! docker exec st-apiserver kubectl get nodes; do echo "Waiting for apiserver to come up..."; sleep 2; done

	# And run the controller manager.
	docker run \
		--net=host --name st-controller-manager \
		--detach \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
		/hyperkube controller-manager \
                        --master=127.0.0.1:8080 \
                        --min-resync-period=3m \
                        --allocate-node-cidrs=true \
                        --cluster-cidr=10.10.0.0/16 \
                        --v=5

	# Create CustomResourceDefinition (CRD) for Calico resources
	# from the manifest crds.yaml
	docker run \
	    --net=host \
	    --rm \
		-v  $(CURDIR):/manifests \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
		/hyperkube kubectl \
		--server=http://127.0.0.1:8080 \
		apply -f /manifests/test/crds.yaml

	# Create a Node in the API for the tests to use.
	docker run \
	    --net=host \
	    --rm \
		-v  $(CURDIR):/manifests \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
		/hyperkube kubectl \
		--server=http://127.0.0.1:8080 \
		apply -f /manifests/test/mock-node.yaml

	# Create Namespaces required by namespaced Calico `NetworkPolicy`
	# tests from the manifests namespaces.yaml.
	docker run \
	    --net=host \
	    --rm \
		-v  $(CURDIR):/manifests \
		gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
		/hyperkube kubectl \
		--server=http://localhost:8080 \
		apply -f /manifests/test/namespaces.yaml


## Stop the local kubernetes master
stop-kubernetes-master:
	# Delete the cluster role binding.
	-docker exec st-apiserver kubectl delete clusterrolebinding anonymous-admin

	# Stop master components.
	-docker rm -f st-apiserver st-controller-manager

###############################################################################
# CI/CD
###############################################################################

.PHONY: cd ci version

## checks that we can get the version
version: images
	docker run --rm $(BUILD_IMAGE_SERVER):latest-$(ARCH) --version
	docker run --rm $(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) --version
	docker run --rm $(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) --version
	docker run --rm $(BUILD_IMAGE_REPORTER):latest-$(ARCH) --version
	docker run --rm $(BUILD_IMAGE_BENCHMARKER):latest-$(ARCH) --version

## Builds the code and runs all tests.
ci: images-all version static-checks ut

## Deploys images to registry
cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(BRANCH_NAME) EXCLUDEARCH="$(EXCLUDEARCH)"
	$(MAKE) tag-images-all push-all push-manifests push-non-manifests IMAGETAG=$(shell git describe --tags --dirty --always --long) EXCLUDEARCH="$(EXCLUDEARCH)"

###############################################################################
# Release
###############################################################################
PREVIOUS_RELEASE=$(shell git describe --tags --abbrev=0 )
GIT_VERSION?=$(shell git describe --tags --dirty  2>/dev/null  )

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) VERSION=$(VERSION) release-tag
	$(MAKE) VERSION=$(VERSION) release-build
	$(MAKE) VERSION=$(VERSION) release-verify

	@echo ""
	@echo "Release build complete. Next, push the produced images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish"
	@echo ""

## Produces a git tag for the release.
release-tag: release-prereqs release-notes
	git tag $(VERSION) -F release-notes-$(VERSION)
	@echo ""
	@echo "Now you can build the release:"
	@echo ""
	@echo "  make VERSION=$(VERSION) release-build"
	@echo ""

## Produces a clean build of release artifacts at the specified version.
release-build: release-prereqs clean
# Check that the correct code is checked out.
ifneq ($(VERSION), $(GIT_VERSION))
	$(error Attempt to build $(VERSION) from $(GIT_VERSION))
endif
	$(MAKE) images-all
	$(MAKE) tag-images-all RELEASE=true IMAGETAG=$(VERSION)
	$(MAKE) tag-images-all RELEASE=true IMAGETAG=latest

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	# Check the reported version is correct for each release artifact.
	docker run --rm $(BUILD_IMAGE_SERVER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm $(BUILD_IMAGE_SERVER):$(VERSION)-$(ARCH) --version` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm quay.io/$(BUILD_IMAGE_SERVER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm quay.io/$(BUILD_IMAGE_SERVER):$(VERSION)-$(ARCH) --version | grep -x $(VERSION)` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm $(BUILD_IMAGE_CONTROLLER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm $(BUILD_IMAGE_CONTROLLER):$(VERSION)-$(ARCH) --version` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm quay.io/$(BUILD_IMAGE_CONTROLLER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm quay.io/$(BUILD_IMAGE_CONTROLLER):$(VERSION)-$(ARCH) --version | grep -x $(VERSION)` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm $(BUILD_IMAGE_SNAPSHOTTER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm $(BUILD_IMAGE_SNAPSHOTTER):$(VERSION)-$(ARCH) --version` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm quay.io/$(BUILD_IMAGE_SNAPSHOTTER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm quay.io/$(BUILD_IMAGE_SNAPSHOTTER):$(VERSION)-$(ARCH) --version | grep -x $(VERSION)` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm $(BUILD_IMAGE_REPORTER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm $(BUILD_IMAGE_REPORTER):$(VERSION)-$(ARCH) --version` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm quay.io/$(BUILD_IMAGE_REPORTER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm quay.io/$(BUILD_IMAGE_REPORTER):$(VERSION)-$(ARCH) --version | grep -x $(VERSION)` "\nExpected version: $(VERSION)" && exit 1 )
	docker run --rm quay.io/$(BUILD_IMAGE_BENCHMARKER):$(VERSION)-$(ARCH) --version | grep $(VERSION) || ( echo "Reported version:" `docker run --rm quay.io/$(BUILD_IMAGE_BENCHMARKER):$(VERSION)-$(ARCH) --version | grep -x $(VERSION)` "\nExpected version: $(VERSION)" && exit 1 )

	# TODO: Some sort of quick validation of the produced binaries.

## Generates release notes based on commits in this version.
release-notes: release-prereqs
	mkdir -p dist
	echo "# Changelog" > release-notes-$(VERSION)
	echo "" >> release-notes-$(VERSION)
	sh -c "git cherry -v $(PREVIOUS_RELEASE) | cut '-d ' -f 2- | sed 's/^/- /' >> release-notes-$(VERSION)"

## Pushes a github release and release artifacts produced by `make release-build`.
release-publish: release-prereqs
	# Push the git tag.
	git push origin $(VERSION)

	# Push images.
	$(MAKE) push-all push-manifests push-non-manifests RELEASE=true IMAGETAG=$(VERSION)

	@echo "Finalize the GitHub release based on the pushed tag."
	@echo ""
	@echo "  https://$(PACKAGE_NAME)/releases/tag/$(VERSION)"
	@echo ""
	@echo "If this is the latest stable release, then run the following to push 'latest' images."
	@echo ""
	@echo "  make VERSION=$(VERSION) release-publish-latest"
	@echo ""

# WARNING: Only run this target if this release is the latest stable release. Do NOT
# run this target for alpha / beta / release candidate builds, or patches to earlier Calico versions.
## Pushes `latest` release images. WARNING: Only run this for latest stable releases.
release-publish-latest: release-prereqs
	# Check latest versions match.
	if ! docker run $(BUILD_IMAGE_SERVER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run $(BUILD_IMAGE_SERVER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi
	if ! docker run quay.io/$(BUILD_IMAGE_SERVER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run quay.io/$(BUILD_IMAGE_SERVER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi
	if ! docker run $(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run $(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi
	if ! docker run quay.io/$(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run quay.io/$(BUILD_IMAGE_CONTROLLER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi
	if ! docker run $(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run $(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi
	if ! docker run quay.io/$(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run quay.io/$(BUILD_IMAGE_SNAPSHOTTER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi
	if ! docker run $(BUILD_IMAGE_REPORTER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run $(BUILD_IMAGE_REPORTER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi
	if ! docker run quay.io/$(BUILD_IMAGE_REPORTER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run quay.io/$(BUILD_IMAGE_REPORTER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi
	if ! docker run quay.io/$(BUILD_IMAGE_BENCHMARKER):latest-$(ARCH) --version | grep '$(VERSION)'; then echo "Reported version:" `docker run quay.io/$(BUILD_IMAGE_BENCHMARKER):latest-$(ARCH) --version` "\nExpected version: $(VERSION)"; false; else echo "\nVersion check passed\n"; fi

	$(MAKE) push-all push-manifests push-non-manifests RELEASE=true IMAGETAG=latest

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=vX.Y.Z)
endif
ifeq ($(GIT_COMMIT),<unknown>)
	$(error git commit ID could not be determined, releases must be done from a git working copy)
endif
ifdef LOCAL_BUILD
	$(error LOCAL_BUILD must not be set for a release)
endif

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
.PHONY: ut-no-cover
ut-no-cover: $(SRC_FILES)
	@echo Running Go UTs without coverage.
	$(DOCKER_RUN) $(CALICO_BUILD) ginkgo -r $(GINKGO_OPTIONS)

.PHONY: ut-watch
ut-watch: $(SRC_FILES)
	@echo Watching go UTs for changes...
	$(DOCKER_RUN) $(CALICO_BUILD) ginkgo watch -r $(GINKGO_OPTIONS)

# Launch a browser with Go coverage stats for the whole project.
.PHONY: cover-browser
cover-browser: combined.coverprofile
	go tool cover -html="combined.coverprofile"

.PHONY: cover-report
cover-report: combined.coverprofile
	# Print the coverage.  We use sed to remove the verbose prefix and trim down
	# the whitespace.
	@echo
	@echo -------- All coverage ---------
	@echo
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'go tool cover -func combined.coverprofile | \
	                           sed 's=$(PACKAGE_NAME)/==' | \
	                           column -t'
	@echo
	@echo -------- Missing coverage only ---------
	@echo
	@$(DOCKER_RUN) $(CALICO_BUILD) sh -c "go tool cover -func combined.coverprofile | \
	                           sed 's=$(PACKAGE_NAME)/==' | \
	                           column -t | \
	                           grep -v '100\.0%'"

bin/server.transfer-url: bin/server-$(ARCH)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/server-$(ARCH) https://transfer.sh/tigera-compliance-server > $@'

bin/controller.transfer-url: bin/controller-$(ARCH)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/controller-$(ARCH) https://transfer.sh/tigera-compliance-controller > $@'

bin/snapshotter.transfer-url: bin/snapshotter-$(ARCH)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/snapshotter-$(ARCH) https://transfer.sh/tigera-compliance-snapshotter > $@'

bin/reporter.transfer-url: bin/reporter-$(ARCH)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/reporter-$(ARCH) https://transfer.sh/tigera-compliance-reporter > $@'

bin/scaleloader.transfer-url: bin/scaleloader-$(ARCH)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/scaleloader-$(ARCH) https://transfer.sh/tigera-compliance-scaleloader > $@'

bin/benchmarker.transfer-url: bin/benchmarker-$(ARCH)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/benchmarker-$(ARCH) https://transfer.sh/tigera-compliance-benchmarker > $@'

# Install or update the tools used by the build
.PHONY: update-tools
update-tools:
	go get -u github.com/onsi/ginkgo/ginkgo

help:
	@echo "Compliance Components Makefile"
	@echo
	@echo "Dependencies: docker 1.12+; go 1.8+"
	@echo
	@echo "For any target, set ARCH=<target> to build for a given target."
	@echo "For example, to build for arm64:"
	@echo
	@echo "  make build ARCH=arm64"
	@echo
	@echo "Initial set-up:"
	@echo
	@echo "  make update-tools  Update/install the go build dependencies."
	@echo
	@echo "Builds:"
	@echo
	@echo "  make all           Build all the binary packages."
	@echo "  make images        Build $(BUILD_IMAGE_SERVER), $(BUILD_IMAGE_CONTROLLER),"
	@echo "                     $(BUILD_IMAGE_SNAPSHOTTER) and $(BUILD_IMAGE_REPORTER) docker images."
	@echo
	@echo "Tests:"
	@echo
	@echo "  make ut                Run UTs."
	@echo
	@echo "Maintenance:"
	@echo
	@echo "  make go-fmt        Format our go code."
	@echo "  make clean         Remove binary files."
	@echo "-----------------------------------------"
	@echo "ARCH (target):          $(ARCH)"
	@echo "BUILDARCH (host):       $(BUILDARCH)"
	@echo "CALICO_BUILD:     $(CALICO_BUILD)"
	@echo "-----------------------------------------"
