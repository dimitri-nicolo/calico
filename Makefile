CALICO_DIR=$(shell git rev-parse --show-toplevel)
GIT_HASH=$(shell git rev-parse --short HEAD)
VERSIONS_FILE?=$(CALICO_DIR)/_data/versions.yml
IMAGES_FILE=
JEKYLL_VERSION=pages
HP_VERSION=v0.2
DEV?=false
CONFIG=--config _config.yml
ifeq ($(DEV),true)
	CONFIG:=$(CONFIG),_config_dev.yml
endif
ifneq ($(IMAGES_FILE),)
	CONFIG:=$(CONFIG),/config_images.yml
endif

GO_BUILD_VER?=v0.20
CALICO_BUILD?=calico/go-build:$(GO_BUILD_VER)
LOCAL_USER_ID?=$(shell id -u $$USER)
PACKAGE_NAME?=github.com/projectcalico/calico

GO_BUILD_VER?=v0.20
CALICO_BUILD?=calico/go-build:$(GO_BUILD_VER)
LOCAL_USER_ID?=$(shell id -u $$USER)
PACKAGE_NAME?=github.com/projectcalico/calico

# Determine whether there's a local yaml installed or use dockerized version.
# Note in order to install local (faster) yaml: "go get github.com/mikefarah/yq.v2"
YAML_CMD:=$(shell which yq.v2 || echo docker run --rm -i calico/yaml)

# Local directories to ignore when running htmlproofer
HP_IGNORE_LOCAL_DIRS="/v2.0/"

##############################################################################
# Version information used for cutting a release.
RELEASE_STREAM?=

CHART?=calico
REGISTRY?=gcr.io/unique-caldron-775/cnx/
DOCS_TEST_CONTAINER?=tigera/docs-test

# Use := so that these V_ variables are computed only once per make run.
CALICO_VER := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '"$(RELEASE_STREAM)".[0].title')
NODE_VER := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '"$(RELEASE_STREAM)".[0].components.cnx-node.version')
CTL_VER := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '"$(RELEASE_STREAM)".[0].components.calicoctl.version')
CNI_VER := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '"$(RELEASE_STREAM)".[0].components.calico/cni.version')
KUBE_CONTROLLERS_VER := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '"$(RELEASE_STREAM)".[0].components.calico/kube-controllers.version')
TYPHA_VER := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '"$(RELEASE_STREAM)".[0].components.typha.version')

##############################################################################

# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks
	
## Serve a local view of your current site on port 4000
serve: bin/helm
	# We have to override JEKYLL_DOCKER_TAG which is usually set to 'pages'.
	# When set to 'pages', jekyll starts in safe mode which means it will not
	# load any plugins. Since we're no longer running in github-pages, but would
	# like to use a docker image that comes preloaded with all the github-pages plugins,
	# its ok to override this variable.
	docker run --rm \
	  -v $$PWD/bin/helm:/usr/local/bin/helm:ro \
	  -v $$PWD:/srv/jekyll \
	  -e JEKYLL_DOCKER_TAG="" \
	  -e JEKYLL_UID=`id -u` \
	  -p 4000:4000 \
	  jekyll/jekyll:$(JEKYLL_VERSION) jekyll serve --incremental $(CONFIG)

.PHONY: build
_site build: bin/helm
	docker run --rm \
	-e JEKYLL_DOCKER_TAG="" \
	-e JEKYLL_UID=`id -u` \
	-v $$PWD/bin/helm:/usr/local/bin/helm:ro \
	-v $$PWD:/srv/jekyll \
	-v $(VERSIONS_FILE):/srv/jekyll/_data/versions.yml \
	-v $(IMAGES_FILE):/config_images.yml \
	jekyll/jekyll:$(JEKYLL_VERSION) jekyll build --incremental $(CONFIG)

## Clean enough that a new release build will be clean
clean:
	rm -rf _output _site .jekyll-metadata stderr.out filtered.out docs_test.created

########################################################################################################################
# Builds locally checked out code using local versions of libcalico, felix, and confd.
#
# Example commands:
#
#       # Make a build of your locally checked out code with custom registry.
#	make dev-clean dev-image REGISTRY=caseydavenport
#
#	# Build a set of manifests using the produced images.
#	make dev-manifests REGISTRY=caseydavenport
#
#	# Push the built images.
#	make dev-push REGISTRY=caseydavenport
#
#	# Make a build using a specific tag, e.g. calico/node:mytag-amd64.
#	make dev-clean dev-image TAG_COMMAND='echo mytag'
#
########################################################################################################################
RELEASE_REPOS=felix typha kube-controllers calicoctl cni-plugin app-policy pod2daemon node
RELEASE_BRANCH_REPOS=$(sort $(RELEASE_REPOS) libcalico-go confd)
TAG_COMMAND=git describe --tags --dirty --always --long
REGISTRY?=calico
LOCAL_BUILD=true
.PHONY: dev-image dev-test dev-vendor dev-clean
## Build a local version of Calico based on the checked out codebase.
dev-image: dev-vendor $(addsuffix -dev-image, $(filter-out calico felix, $(RELEASE_REPOS)))
$(addsuffix -dev-image,$(RELEASE_REPOS)): %-dev-image: ../%
	@cd $< && export TAG=$$($(TAG_COMMAND)); make image tag-images \
		BUILD_IMAGE=$(REGISTRY)/$* \
		PUSH_IMAGES=$(REGISTRY)/$* \
		LOCAL_BUILD=$(LOCAL_BUILD) \
		IMAGETAG=$$TAG

## Push locally built images.
dev-push: $(addsuffix -dev-push, $(filter-out calico felix, $(RELEASE_REPOS)))
$(addsuffix -dev-push,$(RELEASE_REPOS)): %-dev-push: ../%
	@cd $< && export TAG=$$($(TAG_COMMAND)); make push \
		BUILD_IMAGE=$(REGISTRY)/$* \
		PUSH_IMAGES=$(REGISTRY)/$* \
		LOCAL_BUILD=$(LOCAL_BUILD) \
		IMAGETAG=$$TAG

## Run all tests against currently checked out code. WARNING: This takes a LONG time.
dev-test: dev-vendor $(addsuffix -dev-test, $(filter-out calico, $(RELEASE_REPOS)))
$(addsuffix -dev-test,$(RELEASE_REPOS)): %-dev-test: ../%
	@cd $< && make test LOCAL_BUILD=$(LOCAL_BUILD)

dev-vendor: $(addsuffix -dev-vendor, $(filter-out calico, $(RELEASE_BRANCH_REPOS)))
$(addsuffix -dev-vendor,$(RELEASE_BRANCH_REPOS)): %-dev-vendor: ../%
	@cd $< && make vendor

## Run `make clean` across all repos.
dev-clean: $(addsuffix -dev-clean, $(filter-out calico felix, $(RELEASE_REPOS)))
$(addsuffix -dev-clean,$(RELEASE_REPOS)): %-dev-clean: ../%
	@cd $< && export TAG=$$($(TAG_COMMAND)); make clean \
		BUILD_IMAGE=$(REGISTRY)/$* \
		PUSH_IMAGES=$(REGISTRY)/$* \
		LOCAL_BUILD=$(LOCAL_BUILD) \
		IMAGETAG=$$TAG

dev-manifests: dev-versions-yaml dev-images-file
	@make bin/helm
	@make clean _site \
		VERSIONS_FILE="$$PWD/pinned_versions.yml" \
		IMAGES_FILE="$$PWD/pinned_images.yml" \
		DEV=true
	@mkdir -p _output
	@cp -r _site/master/manifests _output/dev-manifests

# Builds an images file for help in building the docs manifests. We need this in order
# to override the default images file with the desired registry and image names as
# produced by the `dev-image` target.
dev-images-file:
	@echo "imageNames:" > pinned_images.yml
	@echo "  node: $(REGISTRY)/node" >> pinned_images.yml
	@echo "  calicoctl: $(REGISTRY)/calicoctl" >> pinned_images.yml
	@echo "  typha: $(REGISTRY)/typha" >> pinned_images.yml
	@echo "  cni: $(REGISTRY)/cni-plugin" >> pinned_images.yml
	@echo "  kubeControllers: $(REGISTRY)/kube-controllers" >> pinned_images.yml
	@echo "  calico-upgrade: $(REGISTRY)/upgrade" >> pinned_images.yml
	@echo "  flannel: quay.io/coreos/flannel" >> pinned_images.yml
	@echo "  dikastes: $(REGISTRY)/app-policy" >> pinned_images.yml
	@echo "  pilot-webhook: $(REGISTRY)/pilot-webhook" >> pinned_images.yml
	@echo "  flexvol: $(REGISTRY)/pod2daemon" >> pinned_images.yml


# Builds a versions.yaml file that corresponds to the versions produced by the `dev-image` target.
dev-versions-yaml:
	@export TYPHA_VER=`cd ../typha && $(TAG_COMMAND)`-amd64; \
	export CTL_VER=`cd ../calicoctl && $(TAG_COMMAND)`-amd64; \
	export NODE_VER=`cd ../node && $(TAG_COMMAND)`-amd64; \
	export CNI_VER=`cd ../cni-plugin && $(TAG_COMMAND)`-amd64; \
	export KUBE_CONTROLLERS_VER=`cd ../kube-controllers && $(TAG_COMMAND)`-amd64; \
	export APP_POLICY_VER=`cd ../app-policy && $(TAG_COMMAND)`-amd64; \
	export POD2DAEMON_VER=`cd ../pod2daemon && $(TAG_COMMAND)`-amd64; \
	/bin/echo -e \
"master:\\n"\
"- title: \"dev-build\"\\n"\
"  note: \"Developer build\"\\n"\
"  components:\\n"\
"     typha:\\n"\
"      version: $$TYPHA_VER\\n"\
"     calicoctl:\\n"\
"      version:  $$CTL_VER\\n"\
"     calico/node:\\n"\
"      version:  $$NODE_VER\\n"\
"     calico/cni:\\n"\
"      version:  $$CNI_VER\\n"\
"     calico/kube-controllers:\\n"\
"      version: $$KUBE_CONTROLLERS_VER\\n"\
"     networking-calico:\\n"\
"      version: master\\n"\
"     flannel:\\n"\
"      version: v0.11.1\\n"\
"     calico/dikastes:\\n"\
"      version: $$APP_POLICY_VER\\n"\
"     flexvol:\\n"\
"      version: $$POD2DAEMON_VER\\n" > pinned_versions.yml;

###############################################################################
# CI / test targets
###############################################################################
docs_test.created:
	docker build -t $(DOCS_TEST_CONTAINER) -f docs_test/Dockerfile.python .
	touch docs_test.created

test: docs_test.created
	docker run --rm \
        -v $(PWD):/code \
        -e RELEASE_STREAM=$(RELEASE_STREAM) \
	-e QUAY_API_TOKEN=$(QUAY_API_TOKEN) \
	-e GIT_HASH=$(GIT_HASH) \
	$(DOCS_TEST_CONTAINER) sh -c \
	"nosetests . $(EXCLUDE_PARAMS) -v --nocapture --with-xunit \
	--xunit-file='/code/tests/report/nosetests.xml' \
	--with-timer"

ci: clean htmlproofer kubeval helm-tests

htmlproofer: _site
	# Run htmlproofer, failing if we hit any errors.
	./htmlproofer.sh

kubeval: _site
	# Run kubeval to check master manifests are valid Kubernetes resources.
	-docker run -v $$PWD:/calico --entrypoint /bin/sh garethr/kubeval:0.7.3 -c 'ok=true; for f in `find /calico/_site/master -name "*.yaml" |grep -v "\(patch-cnx-manager-configmap\|kube-controllers-patch\|config\|allow-istio-pilot\|30-policy\|cnx-policy\|crds-only\|istio-app-layer-policy\|patch-flow-logs\|upgrade-calico\|upgrade-calico-3.10\|-cf\).yaml"`; do echo Running kubeval on $$f; /kubeval $$f || ok=false; done; $$ok' 1>stderr.out 2>&1

	# Filter out error loading schema for non-standard resources.
	-grep -v "Could not read schema from HTTP, response status is 404 Not Found" stderr.out > filtered.out

	# Filter out error reading empty secrets (which we use for e.g. etcd secrets and seem to work).
	-grep -v "invalid Secret" filtered.out > filtered.out

	# Filter out error reading calico networkpolicy since kubeval thinks they're kubernetes networkpolicies and
	# complains when it doesn't have a podSelector. Unfortunately, this also filters out networkpolicy failures.
	# TODO: don't filter out k8s networkpolicy errors
	-grep -v "invalid NetworkPolicy" filtered.out > filtered.out

	# Display the errors with context and fail if there were any.
	-rm stderr.out
	! grep -C3 -P "invalid|\t\*" filtered.out
	rm filtered.out

helm-tests: vendor bin/helm values.yaml
ifndef RELEASE_STREAM
	# Default the version to master if not set
	$(eval RELEASE_STREAM = master)
endif
	mkdir -p .go-pkg-cache && \
		docker run --rm \
		--net=host \
		-v $$(pwd):/go/src/$(PACKAGE_NAME):rw \
		-v $$(pwd)/.go-pkg-cache:/go/pkg:rw \
		-v $$(pwd)/bin/helm:/usr/local/bin/helm \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-w /go/src/$(PACKAGE_NAME) \
		$(CALICO_BUILD) ginkgo -cover -r -skipPackage vendor ./helm-tests -chart-path=./_includes/$(RELEASE_STREAM)/charts/calico,./_includes/$(RELEASE_STREAM)/charts/tigera-secure-ee $(GINKGO_ARGS)

###############################################################################
# Docs automation
###############################################################################

# URLs to ignore when checking external links.
HP_IGNORE_URLS=/docs.openshift.org/

check_external_links: _site
	docker run -ti -e JEKYLL_UID=`id -u` --rm -v $(PWD)/_site:/_site/ quay.io/calico/htmlproofer:$(HP_VERSION) /_site --external_only --file-ignore $(HP_IGNORE_LOCAL_DIRS) --assume-extension --url-ignore $(HP_IGNORE_URLS) --internal_domains "docs.tigera.io"

strip_redirects:
	find \( -name '*.md' -o -name '*.html' \) -exec sed -i'' '/redirect_from:/d' '{}' \;

add_redirects_for_latest: strip_redirects
ifndef VERSION
	$(error VERSION is undefined - run using make add_redirects_for_latest VERSION=vX.Y)
endif
	# Check that the VERSION directory already exists
	@test -d $(VERSION)

	# Add the redirect line - look at .md files only and add "redirect_from: XYZ" on a new line after each "title:"
	find $(VERSION) \( -name '*.md' -o -name '*.html' \) -exec sed -i 's#^title:.*#&\nredirect_from: {}#' '{}' \;

	# Check the redirect_from lines and update the version to be "latest"
	find $(VERSION) \( -name '*.md' -o -name '*.html' \) -exec sed -i 's#^\(redirect_from: \)$(VERSION)#\1latest#' '{}' \;

	# Check the redirect_from lines and strip the .md from the URL
	find $(VERSION) \( -name '*.md' -o -name '*.html' \) -exec sed -i 's#^\(redirect_from:.*\)\.md#\1#' '{}' \;

update_canonical_urls:
	# Looks through all directories and replaces previous latest release version numbers in canonical URLs with new
	python release-scripts/update-canonical-urls.py

# Copy a docs change from ORIG_VERSION (default master) to a specified version.
# The docs change copied is all modifications from the master branch.
backport_docs_change:
ifndef VERSION
	$(error VERSION is undefined - run using make backport_docs_change VERSION=vX.Y)
endif  
ifndef ORIG_VERSION
	# Backporting changes from master.
	$(eval ORIG_VERSION = master)
endif
	# (Note that ... indicates the diff from the merge-base.)
	git diff master...HEAD -- $(ORIG_VERSION) > backport_main.patch
	git diff master...HEAD -- _includes/$(ORIG_VERSION) > backport_includes.patch
	git diff master...HEAD -- _data/`echo $(ORIG_VERSION) | tr . _` > backport_data.patch

	-git apply --3way -p2 --directory=$(VERSION) backport_main.patch
	-git apply --3way -p3 --directory=_includes/$(VERSION) backport_includes.patch
	-git apply --3way -p3 --directory=_data/`echo $(VERSION) | tr . _` backport_data.patch
	# "error: unrecognized input" can be ignored if you didn't modify those directories.
	# "error: patch failed" means you will need to manually patch certain directories.

###############################################################################
# Release targets
###############################################################################

## Tags and builds a release from start to finish.
release: release-prereqs
	$(MAKE) release-tag
	$(MAKE) release-build
	$(MAKE) release-verify

	@echo ""
	@echo "Release build complete. Next, push the release."
	@echo ""
	@echo "  make RELEASE_STREAM=$(RELEASE_STREAM) release-publish"
	@echo ""

## Produces a git tag for the release.
release-tag: release-prereqs
	git tag $(CALICO_VER)

## Produces a clean build of release artifacts at the specified version.
release-build: release-prereqs clean
	# Create the release archive.
	$(MAKE) release-archive

## Verifies the release artifacts produces by `make release-build` are correct.
release-verify: release-prereqs
	@echo "TODO: Implement release tar verification"

ifneq (,$(findstring $(RELEASE_STREAM),v3.5 v3.4 v3.3 v3.2 v3.1 v3.0 v2.6))
    # Found: this is an older release.
    REL_NOTES_PATH:=releases
else
    # Not found: this is a newer release.
    REL_NOTES_PATH:=release-notes
endif

## Pushes a github release and release artifacts produced by `make release-build`.
release-publish: release-prereqs
	# Push the git tag.
	git push origin $(CALICO_VER)

	# Push binaries to GitHub release.
	# Requires ghr: https://github.com/tcnksm/ghr
	# Requires GITHUB_TOKEN environment variable set.
	ghr -u projectcalico -r calico \
		-b 'Release notes can be found at https://docs.projectcalico.org/$(RELEASE_STREAM)/$(REL_NOTES_PATH)/' \
		-n $(CALICO_VER) \
		$(CALICO_VER) $(RELEASE_DIR).tgz

	@echo "Verify the GitHub release based on the pushed tag."
	@echo ""
	@echo "  https://github.com/projectcalico/calico/releases/tag/$(CALICO_VER)"
	@echo ""

## Generates release notes for the given version.
release-notes: #release-prereqs
	VERSION=$(CALICO_VER) GITHUB_TOKEN=$(GITHUB_TOKEN) python2 ./release-scripts/generate-release-notes.py

update-authors:
ifndef GITHUB_TOKEN
	$(error GITHUB_TOKEN must be set)
endif
	@echo "# Calico authors" > AUTHORS.md
	@echo "" >> AUTHORS.md
	@echo "This file is auto-generated based on contribution records reported" >> AUTHORS.md
	@echo "by GitHub for the core repositories within the projectcalico/ organization. It is ordered alphabetically." >> AUTHORS.md
	@echo "" >> AUTHORS.md
	@docker run -ti --rm -v $(PWD):/code -e GITHUB_TOKEN=$(GITHUB_TOKEN) python:3 \
		bash -c 'pip install pygithub && /usr/local/bin/python /code/release-scripts/get-contributors.py >> /code/AUTHORS.md'

# release-prereqs checks that the environment is configured properly to create a release.
release-prereqs:
ifndef RELEASE_STREAM
	$(error RELEASE_STREAM is undefined - run using make release RELEASE_STREAM=vX.Y)
endif
	@if [ $(CALICO_VER) != $(NODE_VER) ]; then \
		echo "Expected CALICO_VER $(CALICO_VER) to equal NODE_VER $(NODE_VER)"; \
		exit 1; fi
ifeq (, $(shell which ghr))
	$(error Unable to find `ghr` in PATH, run this: go get -u github.com/tcnksm/ghr)
endif

OUTPUT_DIR?=_output
RELEASE_DIR_NAME?=release-$(CALICO_VER)
RELEASE_DIR?=$(OUTPUT_DIR)/$(RELEASE_DIR_NAME)
RELEASE_DIR_K8S_MANIFESTS?=$(RELEASE_DIR)/k8s-manifests
RELEASE_DIR_IMAGES?=$(RELEASE_DIR)/images
RELEASE_DIR_BIN?=$(RELEASE_DIR)/bin
MANIFEST_SRC ?= ./_site/$(RELEASE_STREAM)/manifests

## Create an archive that contains a complete "Calico" release
release-archive: release-prereqs $(RELEASE_DIR).tgz

$(RELEASE_DIR).tgz: $(RELEASE_DIR) $(RELEASE_DIR_K8S_MANIFESTS) $(RELEASE_DIR_IMAGES) $(RELEASE_DIR_BIN) $(RELEASE_DIR)/README
	tar -czvf $(RELEASE_DIR).tgz -C $(OUTPUT_DIR) $(RELEASE_DIR_NAME)

$(RELEASE_DIR_IMAGES): $(RELEASE_DIR_IMAGES)/calico-node.tar $(RELEASE_DIR_IMAGES)/calico-typha.tar $(RELEASE_DIR_IMAGES)/calico-cni.tar $(RELEASE_DIR_IMAGES)/calico-kube-controllers.tar
$(RELEASE_DIR_BIN): $(RELEASE_DIR_BIN)/calicoctl $(RELEASE_DIR_BIN)/calicoctl-windows-amd64.exe $(RELEASE_DIR_BIN)/calicoctl-darwin-amd64

$(RELEASE_DIR)/README:
	@echo "This directory contains a complete release of Calico $(CALICO_VER)" >> $@
	@echo "Documentation for this release can be found at http://docs.projectcalico.org/$(RELEASE_STREAM)" >> $@
	@echo "" >> $@
	@echo "Docker images (under 'images'). Load them with 'docker load'" >> $@
	@echo "* The calico/node docker image  (version $(NODE_VERS))" >> $@
	@echo "* The calico/typha docker image  (version $(TYPHA_VER))" >> $@
	@echo "* The calico/cni docker image  (version $(CNI_VERS))" >> $@
	@echo "* The calico/kube-controllers docker image (version $(KUBE_CONTROLLERS_VER))" >> $@
	@echo "" >> $@
	@echo "Binaries (for amd64) (under 'bin')" >> $@
	@echo "* The calicoctl binary (for Linux) (version $(CTL_VER))" >> $@
	@echo "* The calicoctl-windows-amd64.exe binary (for Windows) (version $(CTL_VER))" >> $@
	@echo "* The calicoctl-darwin-amd64 binary (for Mac) (version $(CTL_VER))" >> $@
	@echo "" >> $@
	@echo "Kubernetes manifests (under 'k8s-manifests directory')" >> $@

$(RELEASE_DIR):
	mkdir -p $(RELEASE_DIR)

$(RELEASE_DIR_K8S_MANIFESTS):
	# Ensure that the docs site is generated
	rm -rf ../_site
	$(MAKE) _site

	# Find all the hosted manifests and copy them into the release dir. Use xargs to mkdir the destination directory structure before copying them.
	# -printf "%P\n" prints the file name and directory structure with the search dir stripped off
	find $(MANIFEST_SRC) -name  '*.yaml' -printf "%P\n" | \
	  xargs -I FILE sh -c \
	    'mkdir -p $(RELEASE_DIR_K8S_MANIFESTS)/`dirname FILE`;\
	    cp $(MANIFEST_SRC)/FILE $(RELEASE_DIR_K8S_MANIFESTS)/`dirname FILE`;'

$(RELEASE_DIR_IMAGES)/calico-node.tar:
	mkdir -p $(RELEASE_DIR_IMAGES)
	docker pull calico/node:$(NODE_VER)
	docker save --output $@ calico/node:$(NODE_VER)

$(RELEASE_DIR_IMAGES)/calico-typha.tar:
	mkdir -p $(RELEASE_DIR_IMAGES)
	docker pull calico/typha:$(TYPHA_VER)
	docker save --output $@ calico/typha:$(TYPHA_VER)

$(RELEASE_DIR_IMAGES)/calico-cni.tar:
	mkdir -p $(RELEASE_DIR_IMAGES)
	docker pull calico/cni:$(CNI_VER)
	docker save --output $@ calico/cni:$(CNI_VER)

$(RELEASE_DIR_IMAGES)/calico-kube-controllers.tar:
	mkdir -p $(RELEASE_DIR_IMAGES)
	docker pull calico/kube-controllers:$(KUBE_CONTROLLERS_VER)
	docker save --output $@ calico/kube-controllers:$(KUBE_CONTROLLERS_VER)

$(RELEASE_DIR_BIN)/%:
	mkdir -p $(RELEASE_DIR_BIN)
	wget https://github.com/projectcalico/calicoctl/releases/download/$(CTL_VER)/$(@F) -O $@
	chmod +x $@

###############################################################################
# Utilities
###############################################################################
HELM_RELEASE=helm-v2.11.0-linux-amd64.tar.gz
bin/helm:
	mkdir -p bin
	$(eval TMP := $(shell mktemp -d))
	wget https://storage.googleapis.com/kubernetes-helm/$(HELM_RELEASE) -O $(TMP)/$(HELM_RELEASE)
	tar -zxvf $(TMP)/$(HELM_RELEASE) -C $(TMP)
	mv $(TMP)/linux-amd64/helm bin/helm

###############################################################################
# Helm
###############################################################################
# Build values.yaml for all charts
.PHONY: values.yaml
values.yaml: values.yaml/tigera-secure-ee-core values.yaml/tigera-secure-ee
values.yaml/%:
ifndef RELEASE_STREAM
	# Default the version to master if not set
	$(eval RELEASE_STREAM = master)
endif
	docker run --rm \
	  -v $$PWD:/calico \
	  -w /calico \
	  ruby:2.5 ruby ./hack/gen_values_yml.rb --registry $(REGISTRY) --chart $(@F) $(RELEASE_STREAM) > _includes/$(RELEASE_STREAM)/charts/$(@F)/values.yaml

# The following chunk of conditionals sets the Version of the helm chart. 
# Helm requires strict semantic versioning.
# There are several use cases this code seeks to accomodate:
# - For master directory, we hardcode the version to v0.0.0
# - For master branch of any other directory, we should append '-pre' to the
#   latest revision release (CALICO_VER) if it's not already appended,
#   e.g. 'v2.4.0-pre'
# - When building release artifacts, allow the ability to override '-pre' with an
#   integer, e.g. 'v2.4.0-1'
ifeq ($(RELEASE_STREAM), master)
# For master, helm requires semantic versioning, so use v0.0.0
chartVersion:=v0.0.0
else ifdef CHART_RELEASE
# When cutting a final package, use CHART_RELEASE to append a increasing integer.
# eg. 'make charts.yaml RELEASE_STREAM=v2.4 CHART_RELEASE=0' would produce v2.4.0-0
chartVersion=$(CALICO_VER)-$(CHART_RELEASE)
else
# Lastly, for builds of the master branch, we want to append a '-pre',
# but not to any release name that already has one.
chartVersion:=$(subst -pre,,$(CALICO_VER))-pre
endif

charts: values.yaml chart/tigera-secure-ee-core chart/tigera-secure-ee
chart/%:
ifndef RELEASE_STREAM
	$(error Must set RELEASE_STREAM to build charts)
endif
	mkdir -p bin
	helm package ./_includes/$(RELEASE_STREAM)/charts/$(@F) \
	--save=false \
	--destination ./bin/ \
	--version $(chartVersion) \
	--app-version $(CALICO_VER)

 ## Create the vendor directory
vendor: glide.yaml
	# Ensure that the glide cache directory exists.
	mkdir -p $(HOME)/.glide

	docker run --rm -i \
	  -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	  -v $(HOME)/.glide:/home/user/.glide:rw \
	  -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	  -w /go/src/$(PACKAGE_NAME) \
	  $(CALICO_BUILD) glide install -strip-vendor

.PHONY: help
## Display this help text
help: # Some kind of magic from https://gist.github.com/rcmachado/af3db315e31383502660
	$(info Available targets)
	@awk '/^[a-zA-Z\-\_0-9\/]+:/ {                                      \
		nb = sub( /^## /, "", helpMsg );                                \
		if(nb == 0) {                                                   \
			helpMsg = $$0;                                              \
			nb = sub( /^[^:]*:.* ## /, "", helpMsg );                   \
		}                                                               \
		if (nb)                                                         \
			printf "\033[1;31m%-" width "s\033[0m %s\n", $$1, helpMsg;  \
	}                                                                   \
	{ helpMsg = $$0 }'                                                  \
	width=20                                                            \
	$(MAKEFILE_LIST)
