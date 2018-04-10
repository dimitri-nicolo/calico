###############################################################################
# Determine whether there's a local yaml installed or use dockerized version.
# Note, to install yaml: "go get github.com/mikefarah/yaml"
GO_BUILD_VER?=v0.12
CALICO_BUILD?=calico/go-build:$(GO_BUILD_VER)
YAML_CMD:=$(shell which yaml || echo docker run --rm -i $(CALICO_BUILD) yaml)

###############################################################################
# Versions
CALICO_DIR=$(shell git rev-parse --show-toplevel)
VERSIONS_FILE?=$(CALICO_DIR)/_data/versions.yml

###############################################################################
JEKYLL_VERSION=pages
JEKYLL_UID?=`id -u`
DEV?=false

CONFIG=--config _config.yml
ifeq ($(DEV),true)
	CONFIG:=$(CONFIG),_config_dev.yml
endif

serve:
	docker run --rm -e JEKYLL_UID=$(JEKYLL_UID) -p 4000:4000 -v $$PWD:/srv/jekyll jekyll/jekyll:$(JEKYLL_VERSION) jekyll serve --incremental $(CONFIG)

.PHONY: build
_site build:
	docker run --rm -e JEKYLL_UID=$(JEKYLL_UID) -v $$PWD:/srv/jekyll jekyll/jekyll:$(JEKYLL_VERSION) jekyll build --incremental $(CONFIG)

clean:
	docker run --rm -e JEKYLL_UID=$(JEKYLL_UID) -v $$PWD:/srv/jekyll jekyll/jekyll:$(JEKYLL_VERSION) jekyll clean
	@rm -f publish-cnx-docs.yaml

htmlproofer: clean _site
	# Run htmlproofer, failing if we hit any errors. 
	./htmlproofer.sh

	# Run kubeval to check master manifests are valid Kubernetes resources.
	docker run -v $$PWD:/calico --entrypoint /bin/sh -ti garethr/kubeval:0.1.1 -c 'find /calico/_site/master -name "*.yaml" |grep -v config.yaml | grep -v cnx-policy.yaml | xargs /kubeval'

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

.PHONY: publish-cnx-docs.yaml
publish-cnx-docs.yaml: build website/generate_site.sh
	cp website/robots.txt _site/	# robots.txt is not generated by jekyll, copy manually
	website/generate_site.sh > publish-cnx-docs.yaml

.PHONY: publish-cnx-docs
publish-cnx-docs: publish-cnx-docs.yaml
	@echo "In order to publish the site, run the following command:"
	@echo "  gcloud app deploy --project=tigera-docs publish-cnx-docs.yaml --stop-previous-version --promote"
	@echo
	@echo "Then visit: https://docs.tigera.io."
	@echo
	@echo "If you're on the gcloud console and wish to test the site"
	@echo "on the staging server, run the following command:"
	@echo "  dev_appserver.py publish-cnx-docs.yaml"
	@echo
	@echo "Then click on the \"Preview on port 8080\" icon."

# publish-cnx-docs-staging is a developer make target which lets you preview
# the CNX docs in a location that is not visibile to customers.
# Note that you'll need appropriate permissions in the tigera-docs-staging GCP
# project.
#
.PHONY: publish-cnx-docs-staging
publish-cnx-docs-staging: publish-cnx-docs.yaml
	gcloud app deploy --project=tigera-docs-staging publish-cnx-docs.yaml --stop-previous-version --promote
	@echo "Visit https://tigera-docs-staging.appspot.com"
	@echo
	@echo "Note you'll need to authenticate with your tigera.io google account."

update_canonical_urls:
	# You must pass two version numbers into this command, e.g., make update_canonical_urls OLD=v3.0 NEW=v3.1
	# Looks through all directories and replaces previous latest release version numbers in canonical URLs with new
	find . \( -name '*.md' -o -name '*.html' \) -exec sed -i '/canonical_url:/s/$(OLD)/$(NEW)/g' {} \;
