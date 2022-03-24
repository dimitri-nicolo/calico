include ../metadata.mk

CALICO_DIR=$(shell git rev-parse --show-toplevel)/calico
GIT_HASH=$(shell git rev-parse --short=10 HEAD)
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
serve: bin/helm3
	# We have to override JEKYLL_DOCKER_TAG which is usually set to 'pages'.
	# When set to 'pages', jekyll starts in safe mode which means it will not
	# load any plugins. Since we're no longer running in github-pages, but would
	# like to use a docker image that comes preloaded with all the github-pages plugins,
	# its ok to override this variable.
	docker run --rm -it \
	  -v $(CURDIR)/bin/helm3:/usr/local/bin/helm:ro \
	  -v $(CURDIR):/srv/jekyll \
	  -v $(CURDIR)/../libcalico-go:/srv/libcalico-go \
	  -e JEKYLL_DOCKER_TAG="" \
	  -e JEKYLL_UID=`id -u` \
	  -p 4000:4000 \
	  jekyll/jekyll:$(JEKYLL_VERSION) /bin/sh -c 'bundle update; jekyll serve --incremental $(CONFIG)'

.PHONY: build
_site build: bin/helm3
	docker run --rm -it \
	-e JEKYLL_DOCKER_TAG="" \
	-e JEKYLL_UID=`id -u` \
	-v $(CURDIR)/bin/helm3:/usr/local/bin/helm:ro \
	-v $(CURDIR):/srv/jekyll \
	-v $(CURDIR)/../libcalico-go:/srv/libcalico-go \
	-v $(VERSIONS_FILE):/srv/jekyll/_data/versions.yml \
	jekyll/jekyll:$(JEKYLL_VERSION) /bin/sh -c 'bundle update; jekyll build --incremental $(CONFIG)'

## Clean enough that a new release build will be clean
clean:
	# Clean .created files which indicate images / releases have been built.
	find . -name '.*.created*' -type f -delete
	find . -name '.*.published*' -type f -delete
	rm -rf _output _site .jekyll-metadata pinned_versions.yaml _includes/charts/*/values.yaml
	rm -rf stderr.out filtered.out docs_test.created bin

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

ci: clean htmlproofer kubeval

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

# TODO(doublek): These Helm tests are not valid for Enterprise since helm v3 isn't supported yet.
helm-tests: bin/helm3 values.yaml
	$(DOCKER_RUN) \
		-v $(CURDIR)/bin/helm3:/usr/local/bin/helm \
		$(CALICO_BUILD) \
		sh -c "$(GIT_CONFIG_SSH) ginkgo -cover -r ./helm-tests -chart-path=./_includes/$(RELEASE_STREAM)/charts/calico $(GINKGO_ARGS)"

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
	git diff master...HEAD -- _plugins/$(ORIG_VERSION) > backport_helm_values.patch

	-git apply --3way -p2 --directory=$(VERSION) backport_main.patch
	-git apply --3way -p3 --directory=_includes/$(VERSION) backport_includes.patch
	-git apply --3way -p3 --directory=_data/`echo $(VERSION) | tr . _` backport_data.patch
	-git apply --3way -p3 --directory=_plugins/$(VERSION) backport_helm_values.patch
	# "error: unrecognized input" can be ignored if you didn't modify those directories.
	# "error: patch failed" means you will need to manually patch certain directories.

###############################################################################
# Utilities
###############################################################################
HELM3_RELEASE=helm-v3.3.1-linux-amd64.tar.gz 

helm-deps: bin/helm3
bin/helm3:
	mkdir -p bin
	$(eval TMP := $(shell mktemp -d))
	wget -q https://get.helm.sh/$(HELM3_RELEASE) -O $(TMP)/helm3.tar.gz
	tar -zxvf $(TMP)/helm3.tar.gz -C $(TMP)
	mv $(TMP)/linux-amd64/helm bin/helm3

###############################################################################
# Helm
###############################################################################
# Build values.yaml for all charts
.PHONY: values.yaml
values.yaml: _includes/charts/tigera-prometheus-operator/values.yaml _includes/charts/tigera-operator/values.yaml
_includes/charts/%/values.yaml: _plugins/values.rb _plugins/helm.rb _data/versions.yml
	docker run --rm \
	  -v $(CURDIR):/calico \
	  -w /calico \
	  ruby:2.5 ruby ./hack/gen_values_yml.rb --registry $(REGISTRY) --chart $* > $@

# The following chunk of conditionals sets the Version of the helm chart.
# Note that helm requires strict semantic versioning, so we use v0.0 to represent 'master'.
ifdef CHART_RELEASE
# the presence of CHART_RELEASE indicates we're trying to cut an official chart release.
chartVersion:=$(CALICO_VER)-$(CHART_RELEASE)
appVersion:=$(CALICO_VER)
else
# otherwise, it's a nightly build.
ifeq ($(RELEASE_STREAM), master)
# For master, helm requires semantic versioning, so use v0.0
chartVersion:=v0.0
appVersion:=$(CALICO_VER)-$(GIT_HASH)
else
chartVersion:=$(RELEASE_STREAM)
appVersion:=$(CALICO_VER)-$(GIT_HASH)
endif
endif

charts: bin/helm3 chart/tigera-prometheus-operator chart/tigera-operator
chart/tigera-operator: _includes/charts/tigera-operator/charts/tigera-prometheus-operator.tgz
chart/%: _includes/charts/%/values.yaml
	mkdir -p bin
	bin/helm3 package ./_includes/charts/$(@F) \
	--destination ./bin/ \
	--version $(chartVersion) \
	--app-version $(appVersion)

helm-install-gcs-plugin:
	bin/helm3 plugin install https://github.com/viglesiasce/helm-gcs.git
# Upload to Google tigera-helm-charts storage bucket.
publish-charts:
	bin/helm3 repo add tigera gs://tigera-helm-charts
	for chart in ./bin/*.tgz; do \
		bin/helm3 gcs push $$chart gs://tigera-helm-charts; \
	done

# Copy the prometheus-operator chart package as a sub-chart for tigera-operator for helm packaging.
# After the changes in PR #3580, Tigera Operator doesn't depend on Prometheus/Alertmanager resources.
# The symbolic link of tigera-prometheus-operator folder is also removed from the tigera-operator/charts
# directory to keep the folder strucure clean. However, this causes a helm packaging issue that
# prometheus-operator chart is not packaged as a sub-chart. Since we will always build the
# prometheus-operator chart together with the tigera-operator chart, copying the prometheus-operator
# chart package to the tigera-operator/charts directory fixes helm packaging.
_includes/charts/tigera-operator/charts/tigera-prometheus-operator.tgz:
	mkdir -p $(@D)
	cp $$(pwd)/bin/tigera-prometheus-operator-$(chartVersion).tgz $@

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
