include ../metadata.mk

PACKAGE_NAME            ?= github.com/projectcalico/calico/honeypod-controller

##############################################################################
# Define some constants
##############################################################################
HONEYPOD_CONTROLLER_IMAGE ?=honeypod-controller
BUILD_IMAGES              ?=$(HONEYPOD_CONTROLLER_IMAGE)

SNORT_VERSION             ?=2.9.19

# Linker flags for building Honeypod Controller.
#
# We use -X to insert the version information into the placeholder variables
# in the buildinfo package.
#
# We use -B to insert a build ID note into the executable, without which, the
# RPM build tools complain.
LDFLAGS:=-ldflags "\
		-X $(PACKAGE_NAME)/pkg/version.VERSION=$(GIT_VERSION) \
		-X $(PACKAGE_NAME)/pkg/version.BUILD_DATE=$(DATE) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_DESCRIPTION=$(GIT_DESCRIPTION) \
		-X $(PACKAGE_NAME)/pkg/version.GIT_REVISION=$(GIT_COMMIT) \
		-B 0x$(BUILD_ID)"

##############################################################################
# Include ../lib.Makefile before anything else
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
##############################################################################
include ../lib.Makefile

SRC_FILES  = $(shell find pkg cmd -name '*.go') \
		$(shell find ../libcalico-go/lib -name '*.go') \
		$(shell find ../licensing/client -name '*.go') \
		$(shell find ../licensing/monitor -name '*.go') \
		$(shell find ../lma/pkg -name '*.go')

###############################################################################
# Building the binary
###############################################################################
build: bin/controller-$(ARCH) 

.PHONY: bin/controller-$(ARCH)
bin/controller-$(ARCH): $(SRC_FILES)
	$(DOCKER_GO_BUILD) \
		sh -c '$(GIT_CONFIG_SSH) \
			go build -o $@ -v $(BUILD_FLAGS) -ldflags "$(VERSION_FLAGS)" cmd/controller/controller.go'

###############################################################################
# Building the images
###############################################################################
HONEYPOD_CONTROLLER_CONTAINER_CREATED=.honeypod-controller.created-$(ARCH)

# Build the images for the target architecture
.PHONY: image-all
image-all: $(addprefix sub-image-,$(VALIDARCHES))
sub-image-%:
	$(MAKE) image ARCH=$*

.PHONY: image
image: $(HONEYPOD_CONTROLLER_IMAGE)

$(HONEYPOD_CONTROLLER_IMAGE): $(HONEYPOD_CONTROLLER_CONTAINER_CREATED)
$(HONEYPOD_CONTROLLER_CONTAINER_CREATED): docker-image/Dockerfile.$(ARCH) bin/controller-$(ARCH)
	$(DOCKER_BUILD) --build-arg SNORT_VERSION=$(SNORT_VERSION) -t $(HONEYPOD_CONTROLLER_IMAGE):latest-$(ARCH) -f docker-image/Dockerfile.$(ARCH) . --load
	$(MAKE) retag-build-images-with-registries VALIDARCHES=$(ARCH) IMAGETAG=latest
	touch $@

.PHONY: clean
clean:
	rm -rf bin
	find . -name "*.coverprofile" -type f -delete
	find . -name "coverage.xml" -type f -delete
	find . -name ".coverage" -type f -delete
	rm -f $(HONEYPOD_CONTROLLER_CONTAINER_CREATED)
	-docker image rm -f $$(docker images $(HONEYPOD_CONTROLLER_IMAGE) -a -q)

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
.PHONY: cd ci

## Builds the code and runs all tests.
ci: clean static-checks ut

## Deploys images to registry
cd: image-all cd-common

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
