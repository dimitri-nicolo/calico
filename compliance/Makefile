include ../metadata.mk

PACKAGE_NAME            ?= github.com/projectcalico/calico/compliance

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

##############################################################################
# Define some constants
##############################################################################
KUBE_BENCH_VERSION		?= b649588f46c54c84cd9c88510680b5a651f12d46

# Override VALIDARCHES inferenced in common Makefile.
#   This repo differs in how ARCHES are determined compared to common logic.
#   overriding the value with the only platform supported ATM.

COMPLIANCE_SERVER_IMAGE      =compliance-server
COMPLIANCE_CONTROLLER_IMAGE  =compliance-controller
COMPLIANCE_SNAPSHOTTER_IMAGE =compliance-snapshotter
COMPLIANCE_REPORTER_IMAGE    =compliance-reporter
COMPLIANCE_SCALELOADER_IMAGE =compliance-scaleloader
COMPLIANCE_BENCHMARKER_IMAGE =compliance-benchmarker

# NOTE COMPLIANCE_SCALELOADER_IMAGE isn't included as it's a special case that shouldn't be pushed to quay when
# releasing images. Pushing this image to gcr is handled explicitly in the cd target.
BUILD_IMAGES ?=$(COMPLIANCE_SERVER_IMAGE)\
 	$(COMPLIANCE_CONTROLLER_IMAGE)\
 	$(COMPLIANCE_SNAPSHOTTER_IMAGE)\
	$(COMPLIANCE_REPORTER_IMAGE)\
	$(COMPLIANCE_BENCHMARKER_IMAGE)


# remove from the list to push to manifest any registries that do not support multi-arch
# EXCLUDE_MANIFEST_REGISTRIES defined in Makefile.comm
PUSH_MANIFEST_IMAGE_PREFIXES=$(PUSH_IMAGE_PREFIXES:$(EXCLUDE_MANIFEST_REGISTRIES)%=)
PUSH_NONMANIFEST_IMAGE_PREFIXES=$(filter-out $(PUSH_MANIFEST_IMAGE_PREFIXES),$(PUSH_IMAGE_PREFIXES))

# Linker flags for building Compliance Server.
#
# We use -X to insert the version information into the placeholder variables
# in the buildinfo package.
#
# We use -B to insert a build ID note into the executable, without which, the
# RPM build tools complain.
VERSION_FLAGS = -X $(PACKAGE_NAME)/pkg/version.VERSION=$(GIT_VERSION) \
		-X $(PACKAGE_NAME)/pkg/version.BUILD_DATE=$(DATE) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_DESCRIPTION=$(GIT_DESCRIPTION) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_REVISION=$(GIT_COMMIT) \
		-B 0x$(BUILD_ID)

NON_SRC_DIRS = test
# All Compliance Server go files.
SRC_FILES:=$(shell find . $(foreach dir,$(NON_SRC_DIRS),-path ./$(dir) -prune -o) -type f -name '*.go' -print)

.PHONY: clean
clean:
	rm -rf bin \
	       docker-image/scaleloader/bin \
	       release-notes-*
	find . -name "*.coverprofile" -type f -delete
	find . -name "coverage.xml" -type f -delete
	find . -name ".coverage" -type f -delete
	find . -name "*.pyc" -type f -delete

###############################################################################
# Building the binary
###############################################################################
build: bin/benchmarker-$(ARCH) bin/controller-$(ARCH) bin/reporter-$(ARCH) bin/report-type-gen-$(ARCH) \
	bin/scaleloader-$(ARCH) bin/server-$(ARCH) bin/snapshotter-$(ARCH) 

build-all: $(addprefix sub-build-,$(VALIDARCHES))
sub-build-%:
	$(MAKE) build ARCH=$*

.PHONY: bin/server-$(ARCH)
bin/server-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/server/server.go && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/server was not statically linked"; false ) )'

.PHONY: bin/controller-$(ARCH)
bin/controller-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/controller/controller.go && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/controller was not statically linked"; false ) )'

.PHONY: bin/snapshotter-$(ARCH)
bin/snapshotter-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/snapshotter/snapshotter.go && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/snapshotter was not statically linked"; false ) )'

.PHONY: bin/reporter-$(ARCH)
bin/reporter-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/reporter/reporter.go && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/reporter was not statically linked"; false ) )'

.PHONY: bin/report-type-gen-$(ARCH)
bin/report-type-gen-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/report-type-gen/*.go && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/report-type-gen was not statically linked"; false ) )'

.PHONY: bin/scaleloader-$(ARCH)
bin/scaleloader-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/mockdata-scaleloader/scaleloader.go && \
		( ldd $@ 2>&1 | grep -q -e "Not a valid dynamic program" \
		-e "not a dynamic executable" || \
		( echo "Error: bin/scaleloader was not statically linked"; false ) )'

.PHONY: bin/benchmarker-$(ARCH)
bin/benchmarker-$(ARCH): $(SRC_FILES)
	$(DOCKER_RUN) $(CALICO_BUILD) sh -c '$(GIT_CONFIG_SSH) \
	    go build -v -i -o $@ -v -ldflags "$(VERSION_FLAGS)" cmd/benchmarker/benchmarker.go && \
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
.PHONY: images $(BUILD_IMAGES) $(-$(ARCH),$(BUILD_IMAGES))

images: $(BUILD_IMAGES) $(COMPLIANCE_SCALELOADER_IMAGE)
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest

# Build the images for the target architecture
.PHONY: images-all
images-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) images ARCH=$*

# Build the tigera/compliance-server docker image, which contains only Compliance server.
$(COMPLIANCE_SERVER_IMAGE): bin/server-$(ARCH) register
	$(DOCKER_BUILD) -t $(COMPLIANCE_SERVER_IMAGE):latest-$(ARCH) -f docker-image/server/Dockerfile.$(ARCH) . --load

# Build the tigera/compliance-controller docker image, which contains only Compliance controller.
$(COMPLIANCE_CONTROLLER_IMAGE): bin/controller-$(ARCH) register
	$(DOCKER_BUILD) -t $(COMPLIANCE_CONTROLLER_IMAGE):latest-$(ARCH) -f docker-image/controller/Dockerfile.$(ARCH) . --load

# Build the tigera/compliance-snapshotter docker image, which contains only Compliance snapshotter.
$(COMPLIANCE_SNAPSHOTTER_IMAGE): bin/snapshotter-$(ARCH) register
	$(DOCKER_BUILD) -t $(COMPLIANCE_SNAPSHOTTER_IMAGE):latest-$(ARCH) -f docker-image/snapshotter/Dockerfile.$(ARCH) . --load

# Build the tigera/compliance-reporter docker image, which contains only Compliance reporter.
$(COMPLIANCE_REPORTER_IMAGE): bin/reporter-$(ARCH) register
	$(DOCKER_BUILD) -t $(COMPLIANCE_REPORTER_IMAGE):latest-$(ARCH) -f docker-image/reporter/Dockerfile.$(ARCH) . --load

# Build the tigera/compliance-scaleloader docker image, which contains only Compliance scaleloader.
$(COMPLIANCE_SCALELOADER_IMAGE): bin/scaleloader-$(ARCH) register
	rm -rf docker-image/scaleloader/bin
	rm -rf docker-image/scaleloader/playbooks
	rm -rf docker-image/scaleloader/scenarios
	mkdir -p docker-image/scaleloader/bin
	cp bin/scaleloader-$(ARCH) docker-image/scaleloader/bin/
	cp -r mockdata/scaleloader/playbooks docker-image/scaleloader
	cp -r mockdata/scaleloader/scenarios docker-image/scaleloader
	$(DOCKER_BUILD) -t $(COMPLIANCE_SCALELOADER_IMAGE):latest-$(ARCH) -f docker-image/scaleloader/Dockerfile.$(ARCH) docker-image/scaleloader --load

# Build the tigera/compliance-benchmarker docker image, which contains only Compliance benchmarker.
$(COMPLIANCE_BENCHMARKER_IMAGE): check-kubebench-update bin/benchmarker-$(ARCH) register
	$(DOCKER_BUILD) -t $(COMPLIANCE_BENCHMARKER_IMAGE):latest-$(ARCH) -f docker-image/benchmarker/Dockerfile.$(ARCH) . --load

K8S_CLIENT_VERSION := $(shell grep -E 'k8s.io/apiserver' ../go.mod | awk '{print$$2;exit}')
check-kubebench-update:
	if [ "$(K8S_CLIENT_VERSION)" = "v0.23.3" ]; then \
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
# Tests
###############################################################################
.PHONY: ut
ut combined.coverprofile: run-elastic
	@echo Running Go UTs.
	$(DOCKER_RUN) -e ELASTIC_HOST=localhost $(CALICO_BUILD) ./utils/run-coverage sh -c '$(GIT_CONFIG_SSH)'

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
ci: version static-checks ut

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
