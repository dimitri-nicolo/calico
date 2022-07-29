include ../metadata.mk

CALICO_DIR=$(shell git rev-parse --show-toplevel)/calico
GIT_HASH=$(shell git rev-parse --short=11 HEAD)
VERSIONS_FILE?=$(CALICO_DIR)/_data/versions.yml
JEKYLL_VERSION=4.0.0
HP_VERSION=v0.2
DEV?=false
CONFIG=--config _config.yml

# Set DEV_NULL=true to enable the Null Converter which renders the docs site as markdown.
# This is useful for comparing changes to templates & includes.
ifeq ($(DEV_NULL),true)
	CONFIG:=$(CONFIG),_config_null.yml
endif

# Append any additional config files for jekyll to use.
# Typically used from hashreleases. The additional configuration
# files need to present in the current directory to get automatically
# mounted into the build container.
# Note that when specifying multiple files, they should be comma separated.
# Example: make build EXTRA_JEKYLL_CONFIG=_config_url.yml,_config_null.yml
ifdef EXTRA_JEKYLL_CONFIG
	CONFIG:=$(CONFIG),$(EXTRA_JEKYLL_CONFIG)
endif

include ../metadata.mk
CALICO_BUILD?=calico/go-build:$(GO_BUILD_VER)
LOCAL_USER_ID?=$(shell id -u $$USER)
PACKAGE_NAME = github.com/projectcalico/calico/calico

# Determine whether there's a local yaml installed or use dockerized version.
# Note in order to install local (faster) yaml: "go get github.com/mikefarah/yq.v2"
YAML_CMD:=$(shell which yq.v2 || echo docker run --rm -i mikefarah/yq:2.4.2 yq)
HTML_CMD:=$(shell which pandoc || echo docker run --rm --volume "`pwd`:/data" pandoc/core:2.9.2)

# Local directories to ignore when running htmlproofer
HP_IGNORE_LOCAL_DIRS="/v2.0/"

##############################################################################
# Version information used for cutting a release.
RELEASE_STREAM := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '[0].title' | grep --only-matching --extended-regexp '(v[0-9]+\.[0-9]+)|master')

CHART?=calico
REGISTRY?=gcr.io/unique-caldron-775/cnx/
DOCS_TEST_CONTAINER?=tigera/docs-test

# Use := so that these V_ variables are computed only once per make run.
CALICO_VER := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '[0].title')

###############################################################################
# Include ../lib.Makefile
#   Additions to EXTRA_DOCKER_ARGS need to happen before the include since
#   that variable is evaluated when we declare DOCKER_RUN and siblings.
###############################################################################
include ../lib.Makefile

##############################################################################
# Always install the git hooks to prevent publishing closed source code to a non-private repo.
hooks_installed:=$(shell ./install-git-hooks)

.PHONY: install-git-hooks
## Install Git hooks
install-git-hooks:
	./install-git-hooks

## Serve a local view of your current site on port 4000
serve:
	# We have to override JEKYLL_DOCKER_TAG which is usually set to 'pages'.
	# When set to 'pages', jekyll starts in safe mode which means it will not
	# load any plugins. Since we're no longer running in github-pages, but would
	# like to use a docker image that comes preloaded with all the github-pages plugins,
	# its ok to override this variable.
	docker run --rm -it \
	  -v $(CURDIR):/srv/jekyll \
	  -v $(CURDIR)/../libcalico-go:/srv/libcalico-go \
	  -e JEKYLL_DOCKER_TAG="" \
	  -e JEKYLL_UID=`id -u` \
	  -p 4000:4000 \
	  jekyll/jekyll:$(JEKYLL_VERSION) /bin/sh -c 'bundle update; jekyll serve --incremental $(CONFIG)'

DOCS_SOURCE=$(find . -name '*.md') $(find ../manifests -type f)

.PHONY: build
_site build: bin/ocp.tgz $(DOCS_SOURCE)
	docker run --rm \
	-e JEKYLL_DOCKER_TAG="" \
	-e JEKYLL_UID=`id -u` \
	-v $(CURDIR)/:/srv/jekyll \
	-v $(CURDIR)/../libcalico-go:/srv/libcalico-go \
	-v $(VERSIONS_FILE):/srv/jekyll/_data/versions.yml \
	jekyll/jekyll:$(JEKYLL_VERSION) /bin/sh -c 'bundle update; jekyll build --incremental $(CONFIG)'
	# Move the manifests into the the generated site, and also update
	# the repos manifests that the site generated, removing the templates dir.
	cp -r ../manifests/ _site/manifests
	# Move ocp.tgz into place in the generate site.
	cp bin/ocp.tgz _site/manifests

# Creates the tar file used for installing Calico on OpenShift.
bin/ocp.tgz:
	mkdir -p bin
	tar czvf $@ -C ../manifests/ ocp

## Clean enough that a new release build will be clean
clean:
	# Clean .created files which indicate images / releases have been built.
	find . -name '.*.created*' -type f -delete
	find . -name '.*.published*' -type f -delete
	rm -rf _output _site .jekyll-metadata pinned_versions.yaml _includes/charts/*/values.yaml
	rm -rf stderr.out filtered.out docs_test.created bin
	rm -f _includes/charts/tigera-operator/charts/tigera-secure-ee-core.tgz
	rm -f _includes/charts/tigera-operator/charts/tigera-prometheus-operator.tgz

###############################################################################
# CI / test targets
###############################################################################
.PHONY: docs_test.created
docs_test.created:
	docker build -t $(DOCS_TEST_CONTAINER) -f docs_test/Dockerfile.python .

.PHONY: test
test: docs_test.created
	docker run --rm \
		-v $(CURDIR):/code \
		-e RELEASE_STREAM=$(RELEASE_STREAM) \
		-e QUAY_API_TOKEN=$(QUAY_API_TOKEN) \
		-e GITHUB_API_TOKEN=$(GITHUB_ACCESS_TOKEN) \
		-e DOCS_URL=$(DOCS_URL) \
		-e GIT_HASH=$(GIT_HASH) \
		$(DOCS_TEST_CONTAINER) sh -c \
		"nosetests . -e "$(EXCLUDE_REGEX)" \
		-v -s --with-xunit \
		--xunit-file='/code/tests/report/nosetests.xml' \
		--with-timer $(EXTRA_NOSE_ARGS)"

ci: htmlproofer kubeval

htmlproofer: _site
	# Run htmlproofer, failing if we hit any errors.
	./htmlproofer.sh

kubeval: _site
	# Run kubeval to check master manifests are valid Kubernetes resources.
	-docker run -v $(CURDIR):/calico --entrypoint /bin/sh garethr/kubeval:0.7.3 -c 'ok=true; for f in `find /calico/_site/master -name "*.yaml" |grep -v "\(patch-cnx-manager-configmap\|kube-controllers-patch\|config\|allow-istio-pilot\|30-policy\|cnx-policy\|crds-only\|istio-app-layer-policy\|patch-flow-logs\|upgrade-calico\|upgrade-calico-3.10\|-cf\).yaml"`; do echo Running kubeval on $$f; /kubeval $$f || ok=false; done; $$ok' 1>stderr.out 2>&1

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

###############################################################################
# Docs automation
###############################################################################

# URLs to ignore when checking external links.
HP_IGNORE_URLS="/docs.openshift.org/,/localhost/"

check_external_links: _site
	docker run -ti -e JEKYLL_UID=`id -u` --rm -v $(CURDIR)/_site:/_site/ quay.io/calico/htmlproofer:$(HP_VERSION) /_site --external_only --file-ignore $(HP_IGNORE_LOCAL_DIRS) --assume-extension --url-ignore $(HP_IGNORE_URLS) --internal_domains "docs.tigera.io"

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

DOCS_TEST_CONTAINER=projectcalico/release-test
.PHONY: release-test-image
release-test-image:
	cd release-scripts/tests && docker build -t $(DOCS_TEST_CONTAINER) . && cd -

.PHONY: release-test
release-test: release-test-image
	docker run --rm \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v $(CURDIR):/docs \
	-e RELEASE_STREAM=$(RELEASE_STREAM) \
	$(DOCS_TEST_CONTAINER) sh -c \
	"nosetests . -e "$(EXCLUDE_REGEX)" \
	-s -v --with-xunit \
	--xunit-file='/docs/nosetests.xml' \
	--with-timer $(EXTRA_NOSE_ARGS)"

API_GEN_REPO?=tmjd/gen-crd-api-reference-docs
API_GEN_BRANCH?=kb_v2
OPERATOR_VERSION?=master
OPERATOR_REPO?=tigera/operator
build-operator-reference:
	mkdir -p .go-pkg-cache && \
	   docker run --rm \
	   --net=host \
	   -v $(CURDIR):/go/src/$(PACKAGE_NAME):rw \
	   -v $(CURDIR)/.go-pkg-cache:/go/pkg:rw \
	   -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
	   -w /go/src/$(PACKAGE_NAME) \
	   $(CALICO_BUILD) /bin/bash -c 'rm -rf builder && mkdir builder && cd builder && \
	           git clone --depth=1 -b $(API_GEN_BRANCH) https://github.com/$(API_GEN_REPO) api-gen && cd api-gen && \
	           go mod edit -replace github.com/tigera/operator=github.com/$(OPERATOR_REPO)@$(OPERATOR_VERSION) && \
	           go mod download all && go build && \
	           ./gen-crd-api-reference-docs -config /go/src/$(PACKAGE_NAME)/reference/installation/config.json \
	                   -api-dir github.com/tigera/operator/api -out-file /go/src/$(PACKAGE_NAME)/reference/installation/_api.html'

## START builds the release archives for the version
## Creates archive of all the manifests
OUTPUT_DIR?=_output
OPERATOR_VER := $(shell cat $(VERSIONS_FILE) | $(YAML_CMD) read - '[0].tigera-operator.version')
RELEASE_DIR_NAME?=release-$(CALICO_VER)-$(OPERATOR_VER)
RELEASE_DIR?=$(OUTPUT_DIR)/$(RELEASE_DIR_NAME)
RELEASE_DIR_K8S_MANIFESTS?=$(RELEASE_DIR)/manifests
IGNORED_MANIFESTS= 02-tigera-operator-no-resource-loading.yaml
# Determine where the manifests live. For older versions we used
# a different location, but we still need to package them up for patch
# releases.
DEFAULT_MANIFEST_SRC=./_site/manifests
OLD_VERSIONS := v2.0 v2.1 v2.2 v2.3 v2.4
ifneq ($(filter $(RELEASE_STREAM),$(OLD_VERSIONS)),)
DEFAULT_MANIFEST_SRC=./_site/$(RELEASE_STREAM)/getting-started/kubernetes/installation
endif
MANIFEST_SRC?=$(DEFAULT_MANIFEST_SRC)

release-archive: $(RELEASE_DIR).tgz

$(RELEASE_DIR).tgz: $(RELEASE_DIR) $(RELEASE_DIR_K8S_MANIFESTS) $(RELEASE_DIR)/README.md
	$(MAKE) bin/ocp.tgz

	# find ignored manifests in the archive and delete them
	$(foreach var,$(IGNORED_MANIFESTS), find $(RELEASE_DIR) -name $(var) -delete;)

	# converting the generated html file to markdown format for manifest archive.
	$(HTML_CMD) -f html -t markdown_github-raw_html _site/getting-started/private-registry/private-registry-archive.html -o $(RELEASE_DIR)/private-registry.md
	tar -czvf $(RELEASE_DIR).tgz -C $(OUTPUT_DIR) $(RELEASE_DIR_NAME)

$(RELEASE_DIR)/README.md:
	@echo "This directory contains an archive of all the manifests for release of Calico Enterprise $(CALICO_VER)" >> $@
	@echo "Documentation for this release can be found at https://docs.tigera.io/$(RELEASE_STREAM)" >> $@
	@echo "" >> $@
	@echo "To install Calico Enterprise from this archive, please follow the docs at https://docs.tigera.io/$(RELEASE_STREAM)/maintenance/manifest-archive" >> $@
	@echo "and use the appropriate manifest from the archive where ever you are prompted to download a manifest" >> $@
	@echo "" >> $@
	@echo "Example:" >> $@
	@echo "" >> $@
	@echo "From the docs for OpenShift installation, we have the following command" >> $@
	@echo "" >> $@
	@echo "curl https://docs.tigera.io/manifests/ocp/01-cr-installation.yaml -o manifests/01-cr-installation.yaml" >> $@
	@echo "" >> $@
	@echo "For this example, instead of download the manifest using curl, you need to navigate the archive (after extracting) " >> $@
	@echo "and copy the relevant file at manifests/ocp/01-cr-installation.yaml and paste it into your local manifests folder " >> $@
	@echo "" >> $@

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
