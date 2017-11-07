help:
	@echo "Typha Makefile"
	@echo
	@echo "Dependencies: docker 1.12+; go 1.8+"
	@echo
	@echo "Initial set-up:"
	@echo
	@echo "  make update-tools  Update/install the go build dependencies."
	@echo
	@echo "Builds:"
	@echo
	@echo "  make all           Build all the binary packages."
	@echo "  make calico/typha  Build calico/typha docker image."
	@echo
	@echo "Tests:"
	@echo
	@echo "  make ut                Run UTs."
	@echo "  make go-cover-browser  Display go code coverage in browser."
	@echo
	@echo "Maintenance:"
	@echo
	@echo "  make update-vendor  Update the vendor directory with new "
	@echo "                      versions of upstream packages.  Record results"
	@echo "                      in glide.lock."
	@echo "  make go-fmt        Format our go code."
	@echo "  make clean         Remove binary files."

# Disable make's implicit rules, which are not useful for golang, and slow down the build
# considerably.
.SUFFIXES:

all: bin/calico-typha calico/typha bin/typha-client
test: ut

GO_BUILD_CONTAINER?=calico/go-build:v0.9

# Figure out version information.  To support builds from release tarballs, we default to
# <unknown> if this isn't a git checkout.
GIT_COMMIT:=$(shell git rev-parse HEAD || echo '<unknown>')
BUILD_ID:=$(shell git rev-parse HEAD || uuidgen | sed 's/-//g')
GIT_DESCRIPTION:=$(shell git describe --tags || echo '<unknown>')

# Calculate a timestamp for any build artefacts.
DATE:=$(shell date -u +'%FT%T%z')

# List of Go files that are generated by the build process.  Builds should
# depend on these, clean removes them.
GENERATED_GO_FILES:=

# All Typha go files.
TYPHA_GO_FILES:=$(shell find . $(foreach dir,$(NON_TYPHA_DIRS),-path ./$(dir) -prune -o) -type f -name '*.go' -print) $(GENERATED_GO_FILES)

# Figure out the users UID/GID.  These are needed to run docker containers
# as the current user and ensure that files built inside containers are
# owned by the current user.
MY_UID:=$(shell id -u)
MY_GID:=$(shell id -g)

# Build the calico/typha docker image, which contains only Typha.
.PHONY: calico/typha
calico/typha: bin/calico-typha
	rm -rf docker-image/bin
	mkdir -p docker-image/bin
	cp bin/calico-typha docker-image/bin/
	docker build --pull -t calico/typha docker-image

# Pre-configured docker run command that runs as this user with the repo
# checked out to /code, uses the --rm flag to avoid leaving the container
# around afterwards.
DOCKER_RUN_RM:=docker run --rm --user $(MY_UID):$(MY_GID) -v $${PWD}:/code
DOCKER_RUN_RM_ROOT:=docker run --rm -v $${PWD}:/code

# Allow libcalico-go and the ssh auth sock to be mapped into the build container.
ifdef LIBCALICOGO_PATH
  EXTRA_DOCKER_ARGS += -v $(LIBCALICOGO_PATH):/go/src/github.com/projectcalico/libcalico-go:ro
endif
ifdef SSH_AUTH_SOCK
  EXTRA_DOCKER_ARGS += -v $(SSH_AUTH_SOCK):/ssh-agent --env SSH_AUTH_SOCK=/ssh-agent
endif
DOCKER_GO_BUILD := mkdir -p .go-pkg-cache && \
                   docker run --rm \
                              --net=host \
                              $(EXTRA_DOCKER_ARGS) \
                              -e LOCAL_USER_ID=$(MY_UID) \
                              -v $${PWD}:/go/src/github.com/projectcalico/typha:rw \
                              -v $${PWD}/.go-pkg-cache:/go/pkg:rw \
                              -w /go/src/github.com/projectcalico/typha \
                              $(GO_BUILD_CONTAINER)

# Update the vendored dependencies with the latest upstream versions matching
# our glide.yaml.  If there area any changes, this updates glide.lock
# as a side effect.  Unless you're adding/updating a dependency, you probably
# want to use the vendor target to install the versions from glide.lock.
.PHONY: update-vendor
update-vendor:
	mkdir -p $$HOME/.glide
	$(DOCKER_GO_BUILD) glide up --strip-vendor
	touch vendor/.up-to-date

# vendor is a shortcut for force rebuilding the go vendor directory.
.PHONY: vendor
vendor vendor/.up-to-date: glide.lock
	mkdir -p $$HOME/.glide
	$(DOCKER_GO_BUILD) glide install --strip-vendor
	touch vendor/.up-to-date

# Linker flags for building Typha.
#
# We use -X to insert the version information into the placeholder variables
# in the buildinfo package.
#
# We use -B to insert a build ID note into the executable, without which, the
# RPM build tools complain.
LDFLAGS:=-ldflags "\
        -X github.com/projectcalico/typha/pkg/buildinfo.GitVersion=$(GIT_DESCRIPTION) \
        -X github.com/projectcalico/typha/pkg/buildinfo.BuildDate=$(DATE) \
        -X github.com/projectcalico/typha/pkg/buildinfo.GitRevision=$(GIT_COMMIT) \
        -B 0x$(BUILD_ID)"

bin/calico-typha: $(TYPHA_GO_FILES) vendor/.up-to-date
	@echo Building typha...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -v -i -o $@ -v $(LDFLAGS) "github.com/projectcalico/typha/cmd/calico-typha" && \
               ( ldd bin/calico-typha 2>&1 | grep -q "Not a valid dynamic program" || \
	             ( echo "Error: bin/calico-typha was not statically linked"; false ) )'

bin/typha-client: $(TYPHA_GO_FILES) vendor/.up-to-date
	@echo Building typha client...
	mkdir -p bin
	$(DOCKER_GO_BUILD) \
	    sh -c 'go build -v -i -o $@ -v $(LDFLAGS) "github.com/projectcalico/typha/cmd/typha-client" && \
               ( ldd bin/typha-client 2>&1 | grep -q "Not a valid dynamic program" || \
	             ( echo "Error: bin/typha-client was not statically linked"; false ) )'

# Install or update the tools used by the build
.PHONY: update-tools
update-tools:
	go get -u github.com/Masterminds/glide
	go get -u github.com/onsi/ginkgo/ginkgo

# Run go fmt on all our go files.
.PHONY: go-fmt goimports
go-fmt goimports:
	$(DOCKER_GO_BUILD) sh -c 'glide nv -x | \
	                          grep -v -e "^\\.$$" | \
	                          xargs goimports -w -local github.com/projectcalico/'

check-licenses/dependency-licenses.txt: vendor/.up-to-date
	$(DOCKER_GO_BUILD) sh -c 'licenses ./cmd/calico-typha > check-licenses/dependency-licenses.tmp && \
	                          mv check-licenses/dependency-licenses.tmp check-licenses/dependency-licenses.txt'

.PHONY: ut
ut combined.coverprofile: vendor/.up-to-date $(TYPHA_GO_FILES)
	@echo Running Go UTs.
	$(DOCKER_GO_BUILD) ./utils/run-coverage

bin/check-licenses: $(TYPHA_GO_FILES)
	$(DOCKER_GO_BUILD) go build -v -i -o $@ "github.com/projectcalico/typha/check-licenses"

.PHONY: check-licenses
check-licenses: check-licenses/dependency-licenses.txt bin/check-licenses
	@echo Checking dependency licenses
	$(DOCKER_GO_BUILD) bin/check-licenses

.PHONY: go-meta-linter
go-meta-linter: vendor/.up-to-date $(GENERATED_GO_FILES)
	# Run staticcheck stand-alone since gometalinter runs concurrent copies, which
	# uses a lot of RAM.
	$(DOCKER_GO_BUILD) sh -c 'glide nv | xargs -n 3 staticcheck'
	$(DOCKER_GO_BUILD) gometalinter --enable-gc \
	                                --deadline=300s \
	                                --disable-all \
	                                --enable=goimports \
	                                --enable=errcheck \
	                                --vendor ./...

.PHONY: static-checks
static-checks:
	$(MAKE) go-meta-linter check-licenses

.PHONY: ut-no-cover
ut-no-cover: vendor/.up-to-date $(TYPHA_GO_FILES)
	@echo Running Go UTs without coverage.
	$(DOCKER_GO_BUILD) ginkgo -r $(GINKGO_OPTIONS)

.PHONY: ut-watch
ut-watch: vendor/.up-to-date $(TYPHA_GO_FILES)
	@echo Watching go UTs for changes...
	$(DOCKER_GO_BUILD) ginkgo watch -r $(GINKGO_OPTIONS)

# Launch a browser with Go coverage stats for the whole project.
.PHONY: cover-browser
cover-browser: combined.coverprofile
	go tool cover -html="combined.coverprofile"

.PHONY: cover-report
cover-report: combined.coverprofile
	# Print the coverage.  We use sed to remove the verbose prefix and trim down
	# the whitespace.
	@echo
	@echo ======== All coverage =========
	@echo
	@$(DOCKER_GO_BUILD) sh -c 'go tool cover -func combined.coverprofile | \
	                           sed 's=github.com/projectcalico/typha/==' | \
	                           column -t'
	@echo
	@echo ======== Missing coverage only =========
	@echo
	@$(DOCKER_GO_BUILD) sh -c "go tool cover -func combined.coverprofile | \
	                           sed 's=github.com/projectcalico/typha/==' | \
	                           column -t | \
	                           grep -v '100\.0%'"

.PHONY: upload-to-coveralls
upload-to-coveralls: combined.coverprofile
ifndef COVERALLS_REPO_TOKEN
	$(error COVERALLS_REPO_TOKEN is undefined - run using make upload-to-coveralls COVERALLS_REPO_TOKEN=abcd)
endif
	$(DOCKER_GO_BUILD) goveralls -repotoken=$(COVERALLS_REPO_TOKEN) -coverprofile=combined.coverprofile

bin/calico-typha.transfer-url: bin/calico-typha
	$(DOCKER_GO_BUILD) sh -c 'curl --upload-file bin/calico-typha https://transfer.sh/calico-typha > $@'

.PHONY: clean
clean:
	rm -rf bin \
	       docker-image/bin \
	       build \
	       $(GENERATED_GO_FILES) \
	       .glide \
	       vendor \
	       .go-pkg-cache \
	       check-licenses/dependency-licenses.txt \
	       release-notes-*
	find . -name "*.coverprofile" -type f -delete
	find . -name "coverage.xml" -type f -delete
	find . -name ".coverage" -type f -delete
	find . -name "*.pyc" -type f -delete

.PHONY: release release-once-tagged
release: clean
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=X.Y.Z)
endif
ifeq ($(GIT_COMMIT),<unknown>)
	$(error git commit ID couldn't be determined, releases must be done from a git working copy)
endif
	$(DOCKER_GO_BUILD) utils/tag-release.sh $(VERSION)

.PHONY: continue-release
continue-release:
	@echo "Edited release notes are:"
	@echo
	@cat ./release-notes-$(VERSION)
	@echo
	@echo "Hit Return to go ahead and create the tag, or Ctrl-C to cancel."
	@bash -c read
	# Create annotated release tag.
	git tag $(VERSION) -F ./release-notes-$(VERSION)
	rm ./release-notes-$(VERSION)

	# Now decouple onto another make invocation, as we want some variables
	# (GIT_DESCRIPTION and BUNDLE_FILENAME) to be recalculated based on the
	# new tag.
	$(MAKE) release-once-tagged

release-once-tagged:
	@echo
	@echo "Will now build release artifacts..."
	@echo
	$(MAKE) bin/calico-typha calico/typha
	docker tag calico/typha calico/typha:$(VERSION)
	docker tag calico/typha gcr.io/tigera-dev/calico/typha-essentials:latest
	docker tag calico/typha:$(VERSION) gcr.io/tigera-dev/calico/typha-essentials:$(VERSION)
	@echo
	@echo "Checking built typha has correct version..."
	@if docker run gcr.io/tigera-dev/calico/typha-essentials:$(VERSION) calico-typha --version | grep -q '$(VERSION)$$'; \
	then \
	  echo "Check successful."; \
	else \
	  echo "Incorrect version in docker image!"; \
	  false; \
	fi
	@echo
	@echo "Typha release artifacts have been built:"
	@echo
	@echo "- Binary:                 bin/calico-typha"
	@echo "- Docker container image: calico/typha:$(VERSION)"
	@echo "- Same, tagged for GCR private registry:  gcr.io/tigera-dev/calico/typha-essentials:$(VERSION)"
	@echo
	@echo "Now to publish this release to Github:"
	@echo
	@echo "- Push the new tag ($(VERSION)) to https://github.com/tigera/typha-private"
	@echo "- Go to https://github.com/tigera/typha-private/releases/tag/$(VERSION)"
	@echo "- Copy the tag content (release notes) shown on that page"
	@echo "- Go to https://github.com/tigera/typha-private/releases/new?tag=$(VERSION)"
	@echo "- Name the GitHub release:"
	@echo "  - For a stable release: 'Typha $(VERSION)'"
	@echo "  - For a test release:   'Typha $(VERSION) pre-release for testing'"
	@echo "- Paste the copied tag content into the large textbox"
	@echo "- Add an introduction message and, for a significant release,"
	@echo "  append information about where to get the release.  (See the 2.2.0"
	@echo "  release for an example.)"
	@echo "- Attach the binary"
	@echo "- Click the 'This is a pre-release' checkbox, if appropriate"
	@echo "- Click 'Publish release'"
	@echo
	@echo "Then, push the versioned docker images to GCR only:"
	@echo
	@echo "- gcloud docker -- docker push gcr.io/tigera-dev/calico/typha-essentials:$(VERSION)"
	@echo
	@echo "If this is the latest release from the most recent stable"
	@echo "release series, also push the 'latest' tag:"
	@echo
	@echo "- gcloud docker -- docker push gcr.io/tigera-dev/calico/typha-essentials:latest"
