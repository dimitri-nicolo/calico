PACKAGE_NAME            ?= github.com/tigera/honeypod-controller
GO_BUILD_VER            ?= v0.65
GOMOD_VENDOR             = false
GIT_USE_SSH              = true

ORGANIZATION=tigera
SEMAPHORE_PROJECT_ID?=$(SEMAPHORE_HONEYPOD_CONTROLLER_PROJECT_ID)

# Build mounts for running in "local build" mode. Developers will need to make sure they have the correct local version
# otherwise the build will fail.
PHONY:local_build
ifdef LOCAL_BUILD
EXTRA_DOCKER_ARGS += -v $(CURDIR)/../calico-private:/go/src/github.com/tigera/calico-private:rw
local_build:
	go mod edit -replace=github.com/projectcalico/calico=../calico-private
	go mod edit -replace=github.com/tigera/api=../calico-private/api
else
local_build:
endif
ifdef GOPATH
EXTRA_DOCKER_ARGS += -v $(GOPATH)/pkg/mod:/go/pkg/mod:rw
endif

EXTRA_DOCKER_ARGS += -e GOPRIVATE=github.com/tigera/*


##############################################################################
# Define some constants
##############################################################################
HONEYPOD_CONTROLLER_IMAGE ?=tigera/honeypod-controller
SNORT_IMAGE               ?=tigera/snort2
SNORT_VERSION             ?=2.9.19
BUILD_IMAGES              ?=$(HONEYPOD_CONTROLLER_IMAGE)
ARCHES                    ?=amd64
DEV_REGISTRIES            ?=gcr.io/unique-caldron-775/cnx
RELEASE_REGISTRIES        ?=quay.io
RELEASE_BRANCH_PREFIX     ?=release-calient
DEV_TAG_SUFFIX            ?=calient-0.dev

ELASTIC_VERSION			?= 7.16.2
K8S_VERSION     		?= v1.18.6
ETCD_VERSION			?= v3.5.1
KUBE_BENCH_VERSION		?= b649588f46c54c84cd9c88510680b5a651f12d46

# Figure out version information.  To support builds from release tarballs, we default to
# <unknown> if this isn't a git checkout.
PKG_VERSION?=$(shell git describe --tags --dirty --always || echo '<unknown>')
PKG_VERSION_BUILD_DATE?=$(shell date -u +'%FT%T%z' || echo '<unknown>')
PKG_VERSION_GIT_DESCRIPTION?=$(shell git describe --tags 2>/dev/null || echo '<unknown>')
PKG_VERSION_GIT_REVISION?=$(shell git rev-parse --short HEAD || echo '<unknown>')

# Linker flags for building Honeypod Controller.
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

NON_SRC_DIRS = test
# All Honeypod Controller go files.
SRC_FILES:=$(shell find . $(foreach dir,$(NON_SRC_DIRS),-path ./$(dir) -prune -o) -type f -name '*.go' -print)

.PHONY: clean
clean:
	rm -rf bin \
	       docker-image/controller/bin \
	       release-notes-* \
	       .go-pkg-cache \
	       vendor
	find . -name "*.coverprofile" -type f -delete
	find . -name "coverage.xml" -type f -delete
	find . -name ".coverage" -type f -delete
	find . -name "*.pyc" -type f -delete

###############################################################################
# Building the binary
###############################################################################
build: bin/controller 
build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

bin/controller: bin/controller-$(ARCH)
	ln -sf bin/controller-$(ARCH) bin/controller

bin/controller-$(ARCH): $(SRC_FILES) local_build
	@echo Building honeypod controller...
	mkdir -p bin
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v $(BUILD_FLAGS) $(LDFLAGS) "$(PACKAGE_NAME)/cmd/controller" && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/controller was not statically linked"; false ) )'


###############################################################################
# Building the images
###############################################################################
.PHONY: $(HONEYPOD_CONTROLLER_IMAGE) $(HONEYPOD_CONTROLLER_IMAGE)-$(ARCH)
.PHONY: images
.PHONY: image

images image: $(HONEYPOD_CONTROLLER_IMAGE)

# Build the images for the target architecture
.PHONY: images-all
images-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) images ARCH=$*

.PHONY: $(SNORT_IMAGE)

$(SNORT_IMAGE): $(SNORT_IMAGE)-$(ARCH)
$(SNORT_IMAGE)-$(ARCH):
	sh "docker manifest inspect $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH)"; \
	EXIT_CODE=$$?;\
	if [ "$$EXIT_CODE" = 0 ]; then \
  		echo "Using existing snort image $(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH)"; \
  		docker pull $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) ;\
  		docker tag $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) $(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) ;\
  	else \
  	  	echo "Snort image  $(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH)) doesn't exist in $(DEV_REGISTRIES), building it" ; \
  	  	rm -rf docker-image/bin; \
  	  	mkdir -p docker-image/bin; \
  	  	docker build -t $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) -t $(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) --build-arg SNORT_VERSION=$(SNORT_VERSION) --file ./docker-image/snort/Dockerfile docker-image/snort; \
  	  	docker tag $(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) ; \
  	fi
ifeq ($(ARCH),amd64)
	docker tag $(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) $(SNORT_IMAGE):$(SNORT_VERSION)
	docker tag $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH) $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT_VERSION)
endif

# Build the tigera/honeypod-controller docker image.
$(HONEYPOD_CONTROLLER_IMAGE): $(SNORT_IMAGE) bin/controller-$(ARCH) register
	rm -rf docker-image/controller/bin
	mkdir -p docker-image/controller/bin
	cp bin/controller-$(ARCH) docker-image/controller/bin/
	docker build --pull -t $(HONEYPOD_CONTROLLER_IMAGE):latest-$(ARCH) --build-arg QEMU_IMAGE=$(CALICO_BUILD) --build-arg SNORT_VERSION=$(SNORT_VERSION) --build-arg SNORT_IMAGE=$(SNORT_IMAGE) --file ./docker-image/controller/Dockerfile.$(ARCH) docker-image/controller --pull=false
ifeq ($(ARCH),amd64)
	docker tag $(HONEYPOD_CONTROLLER_IMAGE):latest-$(ARCH) $(HONEYPOD_CONTROLLER_IMAGE):latest
endif

.PHONY: push-snort-image
push-snort-image:
	docker push $(DEV_REGISTRIES)/$(SNORT_IMAGE):$(SNORT_VERSION)-$(ARCH)


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

update-calico-pin:
	$(call update_replace_pin,github.com/projectcalico/calico,github.com/tigera/calico-private,$(PIN_BRANCH))
	$(call update_replace_submodule_pin,github.com/tigera/api,github.com/tigera/calico-private/api,$(PIN_BRANCH))

update-pins: guard-ssh-forwarding-bug update-calico-pin

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

###########################
# Better Test
###########################
.PHONY: bt
bt: 
	-kubectl delete -f install/controller.yaml
	-kubectl apply -f install/controller.yaml

POD_NAME=$(shell kubectl get pods -n tigera-intrusion-detection|grep Running| grep honeypod | cut - -c 1-25 | head -n 1)
#bt-terminal: bt
bt-terminal: 
	-kubectl cp bin/controller-amd64 $(POD_NAME):/controller  -n tigera-intrusion-detection
	-urxvt -e bash -c "kubectl exec -it $(POD_NAME)  -n tigera-intrusion-detection /bin/bash"


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
	docker run --rm $(HONEYPOD_CONTROLLER_IMAGE):latest-$(ARCH) --version

## Builds the code and runs all tests.
#ci: images-all version static-checks ut
ci: images-all static-checks ut

## Deploys images to registry
cd: push-snort-image cd-common

###############################################################################
# Developer helper scripts (not used by build or test)
###############################################################################
.PHONY: ut-no-cover
ut-no-cover: $(SRC_FILES)
	@echo Running Go UTs without coverage.
	export ELASTIC_URI=http://127.0.0.1:9200
	$(DOCKER_RUN) $(CALICO_BUILD) ginkgo -r $(GINKGO_OPTIONS)

.PHONY: ut-watch
ut-watch: $(SRC_FILES)
	@echo Watching go UTs for changes...
	export ELASTIC_URI=http://127.0.0.1:9200
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

bin/controller.transfer-url: bin/controller-$(ARCH)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c 'curl --upload-file bin/controller-$(ARCH) https://transfer.sh/tigera-honeypod-controller > $@'

# Install or update the tools used by the build
.PHONY: update-tools
update-tools:
	go get -u github.com/onsi/ginkgo/ginkgo
