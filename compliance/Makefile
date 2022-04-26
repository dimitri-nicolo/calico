PACKAGE_NAME            ?= github.com/tigera/compliance
GO_BUILD_VER            ?= v0.65
GOMOD_VENDOR             = false
GIT_USE_SSH              = true
API_REPO                 =github.com/tigera/api

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_COMPLIANCE_PROJECT_ID)

# Used so semaphore can trigger the update pin pipelines in projects that have this project as a dependency.
SEMAPHORE_AUTO_PIN_UPDATE_PROJECT_IDS=$(SEMAPHORE_FIREWALL_INTEGRATION_PROJECT_ID) $(SEMAPHORE_ES_PROXY_IMAGE_PROJECT_ID)

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

MOCKERY_FILE_PATHS= \
	pkg/datastore/ClusterCtxK8sClientFactory \
	pkg/datastore/ClientSet \
	pkg/list/Source \
	pkg/syncer/SyncerCallbacks

# Build mounts for running in "local build" mode. Developers will need to make sure they have the correct local version
# otherwise the build will fail.
PHONY:local_build
# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/github.com/tigera/libcalico-go:ro
endif
ifdef LOCAL_BUILD
EXTRA_DOCKER_ARGS += -v $(CURDIR)/../libcalico-go:/go/src/github.com/tigera/libcalico-go:rw
EXTRA_DOCKER_ARGS += -v $(CURDIR)/../lma:/go/src/github.com/tigera/lma:rw
local_build:
	go mod edit -replace=github.com/projectcalico/libcalico-go=../libcalico-go
	go mod edit -replace=github.com/tigera/lma=../lma
else
local_build:
endif
ifdef GOPATH
EXTRA_DOCKER_ARGS += -v $(GOPATH)/pkg/mod:/go/pkg/mod:rw
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*

ARCHES = amd64

##############################################################################
# Define some constants
##############################################################################
ELASTIC_VERSION			?= 7.16.2
K8S_VERSION     		?= v1.11.3
ETCD_VERSION			?= v3.3.7
KUBE_BENCH_VERSION		?= b649588f46c54c84cd9c88510680b5a651f12d46

# Override VALIDARCHES inferenced in common Makefile.
#   This repo differs in how ARCHES are determined compared to common logic.
#   overriding the value with the only platform supported ATM.

COMPLIANCE_SERVER_IMAGE      =tigera/compliance-server
COMPLIANCE_CONTROLLER_IMAGE  =tigera/compliance-controller
COMPLIANCE_SNAPSHOTTER_IMAGE =tigera/compliance-snapshotter
COMPLIANCE_REPORTER_IMAGE    =tigera/compliance-reporter
COMPLIANCE_SCALELOADER_IMAGE =tigera/compliance-scaleloader
COMPLIANCE_BENCHMARKER_IMAGE =tigera/compliance-benchmarker

# NOTE COMPLIANCE_SCALELOADER_IMAGE isn't included as it's a special case that shouldn't be pushed to quay when
# releasing images. Pushing this image to gcr is handled explicitly in the cd target.
BUILD_IMAGES ?=$(COMPLIANCE_SERVER_IMAGE)\
 	$(COMPLIANCE_CONTROLLER_IMAGE)\
 	$(COMPLIANCE_SNAPSHOTTER_IMAGE)\
	$(COMPLIANCE_REPORTER_IMAGE)\
	$(COMPLIANCE_BENCHMARKER_IMAGE)


DEV_REGISTRIES        ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES    ?=quay.io
RELEASE_BRANCH_PREFIX ?=release-calient
DEV_TAG_SUFFIX        ?=calient-0.dev

# remove from the list to push to manifest any registries that do not support multi-arch
# EXCLUDE_MANIFEST_REGISTRIES defined in Makefile.comm
PUSH_MANIFEST_IMAGE_PREFIXES=$(PUSH_IMAGE_PREFIXES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGE_PREFIXES=$(filter-out $(PUSH_MANIFEST_IMAGE_PREFIXES),$(PUSH_IMAGE_PREFIXES))

# Figure out version information.  To support builds from release tarballs, we default to
# <unknown> if this isn't a git checkout.
PKG_VERSION?=$(shell git describe --tags --dirty --always --abbrev=12 || echo '<unknown>')
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

# Common Makefile needs to be included after the build env variables are set.
include Makefile.common

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
	       .go-pkg-cache \
	       vendor \
	       Makefile.common*
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
	ln -sf bin/server-$(ARCH) bin/server

bin/server-$(ARCH): $(SRC_FILES) local_build
	@echo Building compliance-server...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/server" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/server was not statically linked"; false ) )'

bin/controller: bin/controller-$(ARCH)
	ln -sf bin/controller-$(ARCH) bin/controller

bin/controller-$(ARCH): $(SRC_FILES) local_build
	@echo Building compliance controller...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/controller" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/controller was not statically linked"; false ) )'

bin/snapshotter: bin/snapshotter-$(ARCH)
	ln -sf bin/snapshotter-$(ARCH) bin/snapshotter

bin/snapshotter-$(ARCH): $(SRC_FILES) local_build
	@echo Building compliance snapshotter...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/snapshotter" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/snapshotter was not statically linked"; false ) )'

bin/reporter: bin/reporter-$(ARCH)
	ln -sf bin/reporter-$(ARCH) bin/reporter

bin/reporter-$(ARCH): $(SRC_FILES) local_build
	@echo Building compliance reporter...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/reporter" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/reporter was not statically linked"; false ) )'

bin/report-type-gen: bin/report-type-gen-$(ARCH)
	ln -sf bin/report-type-gen-$(ARCH) bin/report-type-gen

bin/report-type-gen-$(ARCH): $(SRC_FILES) local_build
	@echo Building report type generator...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/report-type-gen" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/report-type-gen was not statically linked"; false ) )'

bin/scaleloader: bin/scaleloader-$(ARCH)
	ln -sf bin/scaleloader-$(ARCH) bin/scaleloader

bin/scaleloader-$(ARCH): $(SRC_FILES) local_build
	@echo Building scaleloader...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/mockdata-scaleloader" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/scaleloader was not statically linked"; false ) )'

bin/benchmarker: bin/benchmarker-$(ARCH)
	ln -sf bin/benchmarker-$(ARCH) bin/benchmarker

bin/benchmarker-$(ARCH): $(SRC_FILES) local_build
	@echo Building benchmarker...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/benchmarker" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/benchmarker was not statically linked"; false ) )'

###############################################################################
# Building the report files
###############################################################################

.PHONY: gen-files
## Force rebuild of the report generator tool and the default report manifests
gen-files: bin/report-type-gen
	rm -rf ./output/default
	mkdir -p ./output/default/manifests
	mkdir -p ./output/default/json
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c './bin/report-type-gen generate'

###############################################################################
# Building the images
###############################################################################
.PHONY: $(BUILD_IMAGES) $(-$(ARCH),$(BUILD_IMAGES))
.PHONY: images
.PHONY: image

images image: $(BUILD_IMAGES) $(COMPLIANCE_SCALELOADER_IMAGE)

# Build the images for the target architecture
.PHONY: images-all
images-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) images ARCH=$*

# Build the tigera/compliance-server docker image, which contains only Compliance server.
$(COMPLIANCE_SERVER_IMAGE): bin/server-$(ARCH) register
	rm -rf docker-image/server/bin
	mkdir -p docker-image/server/bin
	cp bin/server-$(ARCH) docker-image/server/bin/
	docker build --pull -t $(COMPLIANCE_SERVER_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/server/Dockerfile.$(ARCH) docker-image/server
ifeq ($(ARCH),amd64)
	docker tag $(COMPLIANCE_SERVER_IMAGE):latest-$(ARCH) $(COMPLIANCE_SERVER_IMAGE):latest
endif

# Build the tigera/compliance-controller docker image, which contains only Compliance controller.
$(COMPLIANCE_CONTROLLER_IMAGE): bin/controller-$(ARCH) register
	rm -rf docker-image/controller/bin
	mkdir -p docker-image/controller/bin
	cp bin/controller-$(ARCH) docker-image/controller/bin/
	docker build --pull -t $(COMPLIANCE_CONTROLLER_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/controller/Dockerfile.$(ARCH) docker-image/controller
ifeq ($(ARCH),amd64)
	docker tag $(COMPLIANCE_CONTROLLER_IMAGE):latest-$(ARCH) $(COMPLIANCE_CONTROLLER_IMAGE):latest
endif

# Build the tigera/compliance-snapshotter docker image, which contains only Compliance snapshotter.
$(COMPLIANCE_SNAPSHOTTER_IMAGE): bin/snapshotter-$(ARCH) register
	rm -rf docker-image/snapshotter/bin
	mkdir -p docker-image/snapshotter/bin
	cp bin/snapshotter-$(ARCH) docker-image/snapshotter/bin/
	docker build --pull -t $(COMPLIANCE_SNAPSHOTTER_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/snapshotter/Dockerfile.$(ARCH) docker-image/snapshotter
ifeq ($(ARCH),amd64)
	docker tag $(COMPLIANCE_SNAPSHOTTER_IMAGE):latest-$(ARCH) $(COMPLIANCE_SNAPSHOTTER_IMAGE):latest
endif

# Build the tigera/compliance-reporter docker image, which contains only Compliance reporter.
$(COMPLIANCE_REPORTER_IMAGE): bin/reporter-$(ARCH) register
	rm -rf docker-image/reporter/bin
	mkdir -p docker-image/reporter/bin
	cp bin/reporter-$(ARCH) docker-image/reporter/bin/
	docker build --pull -t $(COMPLIANCE_REPORTER_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/reporter/Dockerfile.$(ARCH) docker-image/reporter
ifeq ($(ARCH),amd64)
	docker tag $(COMPLIANCE_REPORTER_IMAGE):latest-$(ARCH) $(COMPLIANCE_REPORTER_IMAGE):latest
endif

# Build the tigera/compliance-scaleloader docker image, which contains only Compliance scaleloader.
$(COMPLIANCE_SCALELOADER_IMAGE): bin/scaleloader-$(ARCH) register
	rm -rf docker-image/scaleloader/bin
	rm -rf docker-image/scaleloader/playbooks
	rm -rf docker-image/scaleloader/scenarios
	rm -rf docker-image/scaleloader/clean.sh
	mkdir -p docker-image/scaleloader/bin
	cp bin/scaleloader-$(ARCH) docker-image/scaleloader/bin/
	cp docker-image/clean.sh docker-image/scaleloader/clean.sh
	cp -r mockdata/scaleloader/playbooks docker-image/scaleloader
	cp -r mockdata/scaleloader/scenarios docker-image/scaleloader
	docker build --pull -t $(COMPLIANCE_SCALELOADER_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/scaleloader/Dockerfile.$(ARCH) docker-image/scaleloader
ifeq ($(ARCH),amd64)
	docker tag $(COMPLIANCE_SCALELOADER_IMAGE):latest-$(ARCH) $(COMPLIANCE_SCALELOADER_IMAGE):latest
endif

# Build the tigera/compliance-benchmarker docker image, which contains only Compliance benchmarker.
$(COMPLIANCE_BENCHMARKER_IMAGE): check-kubebench-update bin/benchmarker-$(ARCH) register
	rm -rf docker-image/benchmarker/bin
	rm -rf tmp/kube-bench
	rm -rf docker-image/benchmarker/clean.sh
	mkdir -p docker-image/benchmarker/bin
	cp bin/benchmarker-$(ARCH) docker-image/benchmarker/bin/
	docker build --pull -t $(COMPLIANCE_BENCHMARKER_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --file ./docker-image/benchmarker/Dockerfile.$(ARCH) docker-image/benchmarker
ifeq ($(ARCH),amd64)
	docker tag $(COMPLIANCE_BENCHMARKER_IMAGE):latest-$(ARCH) $(COMPLIANCE_BENCHMARKER_IMAGE):latest
endif

K8S_CLIENT_VERSION := $(shell grep -E 'k8s.io/apiserver' go.mod | awk '{print$$2;exit}')
check-kubebench-update:
	if [ $(findstring v0.22,$(K8S_CLIENT_VERSION)) = "v0.22" ]; then \
		echo "No need for kubebench update"; \
	else \
		echo "************ \n"; \
		echo "It looks like we have updated k8s client version.\n";\
		echo "Please check if benchmarker needs to be updated with latest kube-bench version.\n"; \
		echo "Instructions to update benchmarker can be found in README.md#Benchmarker \n"; \
		echo "Once updated (or after verifying that it is not needed) update the check on K8S_CLIENT_VERSION and continue. \n"; \
		echo "****** \n"; \
        exit 1; \
	fi;
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

update-calico-pin:
	$(call update_replace_pin,github.com/projectcalico/calico,github.com/tigera/calico-private,$(PIN_BRANCH))
	$(call update_replace_submodule_pin,github.com/tigera/api,github.com/tigera/calico-private/api,$(PIN_BRANCH))

update-lma-pin:
	$(call update_pin,$(LMA_REPO),$(LMA_REPO),$(LMA_BRANCH))

update-pins: guard-ssh-forwarding-bug update-calico-pin update-lma-pin

###############################################################################

###############################################################################
# Static checks
###############################################################################
# See .golangci.yml for golangci-lint config
LINT_ARGS +=

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
	docker run --rm $(COMPLIANCE_SERVER_IMAGE):latest-$(ARCH) --version
	docker run --rm $(COMPLIANCE_CONTROLLER_IMAGE):latest-$(ARCH) --version
	docker run --rm $(COMPLIANCE_SNAPSHOTTER_IMAGE):latest-$(ARCH) --version
	docker run --rm $(COMPLIANCE_REPORTER_IMAGE):latest-$(ARCH) --version
	docker run --rm $(COMPLIANCE_BENCHMARKER_IMAGE):latest-$(ARCH) --version

## Builds the code and runs all tests.
ci: images-all version static-checks ut

## Deploys images to registry
cd: images-all cd-common
#push the scale loader separately because we don't release it and therefore don't want it as part of the build images list.
	$(MAKE) retag-build-images-with-registries push-images-to-registries push-manifests BUILD_IMAGES=$(COMPLIANCE_SCALELOADER_IMAGE) IMAGETAG=$(BRANCH_NAME) EXCLUDEARCH="$(EXCLUDEARCH)"

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

###############################################################################
# Utils
###############################################################################
# this is not a linked target, available for convenience.
.PHONY: tidy
## 'tidy' go modules.
tidy:
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) go mod tidy'
