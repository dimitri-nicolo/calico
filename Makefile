PACKAGE_NAME = github.com/projectcalico/calico

include metadata.mk
include lib.Makefile

CALICO_VERSIONS_FILE := calico/_data/versions.yml

CALICO_VERSIONS_CALIENT_VERSION_KEY := .[0].title
CALICO_VERSIONS_OPERATOR_VERSION_KEY := .[0].tigera-operator.version
CALICO_VERSIONS_HELM_RELEASE_KEY := .[0].helmRelease

calico_versions_get_val = $(shell bin/yq "$(1)" $(CALICO_VERSIONS_FILE))

DOCKER_RUN := mkdir -p ./.go-pkg-cache bin $(GOMOD_CACHE) && \
	docker run --rm \
		--net=host \
		--init \
		$(EXTRA_DOCKER_ARGS) \
		-e LOCAL_USER_ID=$(LOCAL_USER_ID) \
		-e GOCACHE=/go-cache \
		$(GOARCH_FLAGS) \
		-e GOPATH=/go \
		-e OS=$(BUILDOS) \
		-e GOOS=$(BUILDOS) \
		-e GOFLAGS=$(GOFLAGS) \
		-v $(CURDIR):/go/src/github.com/projectcalico/calico:rw \
		-v $(CURDIR)/.go-pkg-cache:/go-cache:rw \
		-w /go/src/$(PACKAGE_NAME)

clean:
	$(MAKE) -C api clean
	$(MAKE) -C apiserver clean
	$(MAKE) -C app-policy clean
	$(MAKE) -C calicoctl clean
	$(MAKE) -C cni-plugin clean
	$(MAKE) -C confd clean
	$(MAKE) -C felix clean
	$(MAKE) -C kube-controllers clean
	$(MAKE) -C libcalico-go clean
	$(MAKE) -C node clean
	$(MAKE) -C pod2daemon clean
	$(MAKE) -C typha clean
	rm -rf ./bin
	rm -f $(SUB_CHARTS)
	rm -rf _release_archive
	rm -f manifests/ocp.tgz

ci-preflight-checks:
	$(MAKE) check-dockerfiles
	$(MAKE) check-gotchas
	$(MAKE) check-language || true # Enterprise hasn't been cleaned up yet.
	$(MAKE) check-release-cut-promotions
	$(MAKE) generate
	$(MAKE) check-dirty

check-gotchas:
	@if grep github.com/projectcalico/api go.mod; then \
	  echo; \
	  echo "calico-private go.mod should not reference github.com/projectcalico/api"; \
	  echo "Perhaps an import was merged across from open source without being"; \
	  echo "updated to github.com/tigera/api ?"; \
	  echo; \
	  exit 1; \
	fi

check-dockerfiles:
	./hack/check-dockerfiles.sh

check-release-cut-promotions:
	@docker run --quiet --rm \
		-v .:/source \
		-w /source \
		python:3 \
		bash -c 'pip3 install --quiet --disable-pip-version-check --root-user-action ignore PyYAML \
			&& python3 hack/check_semaphore_cut_releases.py'

check-language:
	./hack/check-language.sh

generate:
	$(MAKE) gen-semaphore-yaml
	$(MAKE) -C api gen-files
	$(MAKE) -C libcalico-go gen-files
	$(MAKE) -C felix gen-files
	$(MAKE) -C calicoctl gen-crds
	$(MAKE) -C app-policy protobuf
	$(MAKE) -C egress-gateway protobuf
	$(MAKE) gen-manifests

gen-manifests: bin/helm bin/yq
	# TODO: Ideally we don't need to do this, but the sub-charts
	# mess up manifest generation if they are present.
	rm -f $(SUB_CHARTS)
	cd ./manifests && ./generate.sh

# The following CRDs are modal, in that for most clusters they are Cluster scoped but for multi-tenant clusters they are namespace scoped.
MULTI_TENANCY_CRDS_FILE_CHANGES = "operator.tigera.io_managers.yaml" \
																	"operator.tigera.io_policyrecommendations.yaml" \
																	"operator.tigera.io_compliances.yaml" \
																	"operator.tigera.io_intrusiondetections.yaml" \
																	"calico/crd.projectcalico.org_managedclusters.yaml"
# Get operator CRDs from the operator repo, OPERATOR_BRANCH_NAME must be set
get-operator-crds: var-require-all-OPERATOR_BRANCH_NAME
	cd ./charts/tigera-operator/crds/ && \
	for file in operator.tigera.io_*.yaml; do echo "downloading $$file from operator repo" && curl -fsSL https://raw.githubusercontent.com/tigera/operator/$(OPERATOR_BRANCH_NAME)/pkg/crds/operator/$${file} -o $${file}; done
	cd ./manifests/ocp/ && \
	for file in operator.tigera.io_*.yaml; do echo "downloading $$file from operator repo for ocp" && curl -fsSL https://raw.githubusercontent.com/tigera/operator/$(OPERATOR_BRANCH_NAME)/pkg/crds/operator/$${file} -o $${file}; done
	cp -vLR ./charts/tigera-operator/crds/ ./charts/multi-tenant-crds/. && \
	cd ./charts/multi-tenant-crds/crds && \
	curl -fsSOL https://raw.githubusercontent.com/tigera/operator/$(OPERATOR_BRANCH_NAME)/pkg/crds/operator/operator.tigera.io_tenants.yaml && \
	for file in $(MULTI_TENANCY_CRDS_FILE_CHANGES); do \
		echo "Update CRD $$file to be Namespaced"; \
		sed -i 's/scope: Cluster/scope: Namespaced/g' $$file; \
	done

gen-semaphore-yaml:
	cd .semaphore && ./generate-semaphore-yaml.sh

# Build the tigera-operator helm chart.
ifdef CHART_RELEASE
chartVersion:=$(RELEASE_STREAM)
appVersion:=$(RELEASE_STREAM)
else
chartVersion:=$(GIT_VERSION)
appVersion:=$(GIT_VERSION)
endif

publish: var-require-all-CHART_RELEASE-RELEASE_STREAM-REGISTRY publish-chart-release publish-release-archive
	cd selinux && make publish

chart-release: var-require-all-CHART_RELEASE-RELEASE_STREAM chart
	mv ./bin/tigera-operator-$(RELEASE_STREAM).tgz ./bin/tigera-operator-$(RELEASE_STREAM)-$(CHART_RELEASE).tgz

publish-chart-release: chart-release
	@aws --profile helm s3 cp ./bin/tigera-operator-$(RELEASE_STREAM)-$(CHART_RELEASE).tgz s3://tigera-public/ee/charts/ --acl public-read

publish-release-archive: release-archive
	$(MAKE) -f release-archive.mk publish-release-archive
release-archive: manifests/ocp.tgz
	$(MAKE) -f release-archive.mk release-archive

SUB_CHARTS=charts/tigera-operator/charts/tigera-prometheus-operator.tgz
chart: tigera-operator-release tigera-operator-master multi-tenant-crds-release tigera-prometheus-operator-release

tigera-operator-release: bin/tigera-operator-$(chartVersion).tgz

# Build the multi-tenant-crds helm chart.
multi-tenant-crds-release: bin/multi-tenant-crds-$(chartVersion).tgz
bin/multi-tenant-crds-$(chartVersion).tgz:
	bin/helm package ./charts/multi-tenant-crds \
	--destination ./bin/ \
	--version $(chartVersion) \
	--app-version $(appVersion)

# If we run CD as master from semaphore, we want to also publish bin/tigera-operator-v0.0.tgz for the master docs.
tigera-operator-master:
ifeq ($(SEMAPHORE_GIT_BRANCH), master)
	$(MAKE) bin/tigera-operator-v0.0.tgz
endif

bin/tigera-operator-%.tgz: bin/helm $(shell find ./charts/tigera-operator -type f) $(SUB_CHARTS)
	bin/helm package ./charts/tigera-operator \
	--destination ./bin/ \
	--version $(@:bin/tigera-operator-%.tgz=%) \
	--app-version $(@:bin/tigera-operator-%.tgz=%)

# Build the tigera-prometheus-operator.tgz helm chart.
tigera-prometheus-operator-release: bin/tigera-prometheus-operator-$(chartVersion).tgz
bin/tigera-prometheus-operator-$(chartVersion).tgz:
	bin/helm package ./charts/tigera-prometheus-operator \
	--destination ./bin/ \
	--version $(chartVersion) \
	--app-version $(appVersion)

# Include the tigera-prometheus-operator helm chart as a sub-chart.
charts/tigera-operator/charts/tigera-prometheus-operator.tgz: bin/tigera-prometheus-operator-$(chartVersion).tgz
	mkdir -p $(@D)
	cp bin/tigera-prometheus-operator-$(chartVersion).tgz $@

# Build all Calico images for the current architecture.
image:
	$(MAKE) -C pod2daemon image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)
	$(MAKE) -C calicoctl image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)
	$(MAKE) -C cni-plugin image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)
	$(MAKE) -C apiserver image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)
	$(MAKE) -C kube-controllers image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)
	$(MAKE) -C app-policy image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)
	$(MAKE) -C typha image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)
	$(MAKE) -C node image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)

###############################################################################
# Run local e2e smoke test against the checked-out code
# using a local kind cluster.
###############################################################################
E2E_FOCUS ?= "sig-network.*Conformance"
ADMINPOLICY_UNSUPPORTED_FEATURES ?= "BaselineAdminNetworkPolicy"
e2e-test:
	$(MAKE) -C e2e build
	$(MAKE) -C node kind-k8st-setup
	KUBECONFIG=$(KIND_KUBECONFIG) ./e2e/bin/k8s/e2e.test -ginkgo.focus=$(E2E_FOCUS)
	KUBECONFIG=$(KIND_KUBECONFIG) ./e2e/bin/adminpolicy/e2e.test -exempt-features=$(ADMINPOLICY_UNSUPPORTED_FEATURES)

# Merge OSS branch.
# Expects the following arguments:
# - OSS_REMOTE: Git remote to use for OSS.
# - OSS_BRANCH: OSS branch to merge.
OSS_REMOTE?=open-source
PRIVATE_REMOTE?=origin
OSS_BRANCH?=master
PRIVATE_BRANCH?=master
merge-open:
	git fetch $(OSS_REMOTE)
	git branch -D $(USER)-merge-oss; git checkout -B $(USER)-merge-oss-$(OSS_BRANCH)
	git merge $(OSS_REMOTE)/$(OSS_BRANCH)
	@echo "==========================================================="
	@echo "Resolve any conflicts, push to private, and submit a PR"
	@echo "==========================================================="

os-merge-status:
	@git fetch $(OSS_REMOTE)
	@echo "==============================================================================================================="
	@echo "Showing unmerged commits from calico/$(OSS_BRANCH) that are not in calico-private/$(PRIVATE_BRANCH):"
	@echo ""
	@git --no-pager log --pretty='format:%C(auto)%h %aD: %an: %s' --first-parent  $(PRIVATE_REMOTE)/$(PRIVATE_BRANCH)..$(OSS_REMOTE)/$(OSS_BRANCH)
	@echo ""
	@echo "==============================================================================================================="

# Currently our openstack builds either build *or* build and publish,
# hence why we have two separate jobs here that do almost the same thing.
build-openstack: bin/yq
	$(eval VERSION=$(shell bin/yq '.version' charts/calico/values.yaml))
	$(info Building openstack packages for version $(VERSION))
	$(MAKE) -C release/packaging release VERSION=$(VERSION)

publish-openstack: bin/yq
	$(eval VERSION=$(shell bin/yq '.version' charts/calico/values.yaml))
	$(info Publishing openstack packages for version $(VERSION))
	$(MAKE) -C release/packaging release-publish VERSION=$(VERSION)

## Kicks semaphore job which syncs github released helm charts with helm index file
.PHONY: helm-index
helm-index:
	@echo "Triggering semaphore workflow to update helm index."
	SEMAPHORE_PROJECT_ID=30f84ab3-1ea9-4fb0-8459-e877491f3dea \
			     SEMAPHORE_WORKFLOW_BRANCH=master \
			     SEMAPHORE_WORKFLOW_FILE=../releases/calico/helmindex/update_helm.yml \
			     $(MAKE) semaphore-run-workflow

# Creates the tar file used for installing Calico on OpenShift.
# Excludes manifests that should be applied after cluster creation.
manifests/ocp.tgz: bin/yq
	rm -f $@
	mkdir -p ocp-tmp
	cp -r manifests/ocp ocp-tmp/
	$(DOCKER_RUN) $(CALICO_BUILD) /bin/bash -c " \
		for file in ocp-tmp/ocp/* ; \
        	do bin/yq -i 'del(.. | select(select(has(\"description\")).description|type == \"!!str\").description)' \$$file ; \
        done"
	tar czvf $@ -C ocp-tmp \
		--exclude=tigera-enterprise-resources.yaml \
		--exclude=tigera-prometheus-operator.yaml \
		ocp
	rm -rf ocp-tmp

## Generates release notes for the given version.
.PHONY: release-notes
release-notes:
	@$(MAKE) -C release release-notes

# Create updates for pre-release
release-prep: var-require-all-RELEASE_VERSION-HELM_RELEASE-OPERATOR_VERSION-CALICO_VERSION-REGISTRY var-require-one-of-CONFIRM-DRYRUN
	@cd calico && \
		$(YQ_V4) ".[0].title = \"$(RELEASE_VERSION)\" | .[0].helmRelease = $(HELM_RELEASE)" -i _data/versions.yml && \
		$(YQ_V4) ".[0].tigera-operator.version = \"$(OPERATOR_VERSION)\" | .[0].calico.minor_version = \"$(shell echo "$(CALICO_VERSION)" | awk -F  "." '{print $$1"."$$2}')\"" -i _data/versions.yml && \
		$(YQ_V4) ".[0].components |= with_entries(select(.key | test(\"^(eck-|coreos-).*\") | not)) |= with(.[]; .version = \"$(RELEASE_VERSION)\")" -i _data/versions.yml
	@cd charts && \
		$(YQ_V4) ".tigeraOperator.version = \"$(OPERATOR_VERSION)\" | .calicoctl.tag = \"$(RELEASE_VERSION)\"" -i tigera-operator/values.yaml && \
		$(YQ_V4) ". |= with_entries(select(.key | test(\"^prometheus.*\"))) |= with(.[]; .tag = \"$(RELEASE_VERSION)\")" -i tigera-prometheus-operator/values.yaml && \
		sed -i "s/gcr.io.*\/tigera/quay.io\/tigera/g" tigera*-operator/values.yaml
	$(MAKE) generate CALICO_VERSION=$(RELEASE_VERSION)
	$(eval RELEASE_UPDATE_BRANCH = $(if $(SEMAPHORE),semaphore-,)auto-build-updates-$(RELEASE_VERSION))
	GIT_PR_BRANCH_BASE=$(if $(SEMAPHORE),$(SEMAPHORE_GIT_BRANCH),) RELEASE_UPDATE_BRANCH=$(RELEASE_UPDATE_BRANCH) \
	GIT_PR_BRANCH_HEAD=$(if $(GIT_FORK_USER),$(GIT_FORK_USER):$(RELEASE_UPDATE_BRANCH),$(RELEASE_UPDATE_BRANCH)) GIT_REPO_SLUG=$(if $(SEMAPHORE),$(SEMAPHORE_GIT_REPO_SLUG),) \
	$(MAKE) release-prep/create-and-push-branch release-prep/create-pr release-prep/set-pr-labels


ifneq ($(if $(GIT_REPO_SLUG),$(shell dirname $(GIT_REPO_SLUG)),), $(shell dirname `git config remote.$(GIT_REMOTE).url | cut -d: -f2`))
GIT_FORK_USER:=$(shell dirname `git config remote.$(GIT_REMOTE).url | cut -d: -f2`)
endif
release-prep/create-and-push-branch:
ifeq ($(shell git rev-parse --abbrev-ref HEAD),$(RELEASE_UPDATE_BRANCH))
	$(error Current branch is pull request head, cannot set it up.)
endif
	-git branch -D $(RELEASE_UPDATE_BRANCH)
	-$(GIT) push $(GIT_REMOTE) --delete $(RELEASE_UPDATE_BRANCH)
	git checkout -b $(RELEASE_UPDATE_BRANCH)
	$(GIT) add calico/_data/versions.yml charts/**/values.yaml manifests/*
	$(GIT) commit -m "Automatic version updates for $(RELEASE_VERSION) release"
	$(GIT) push $(GIT_REMOTE) $(RELEASE_UPDATE_BRANCH)

release-prep/create-pr:
	$(call github_pr_create,$(GIT_REPO_SLUG),[$(GIT_PR_BRANCH_BASE)] $(if $(SEMAPHORE),Semaphore ,)Auto Release Update for $(RELEASE_VERSION),$(GIT_PR_BRANCH_HEAD),$(GIT_PR_BRANCH_BASE))
	echo 'Created release update pull request for $(RELEASE_VERSION): $(PR_NUMBER)'

release-prep/set-pr-labels:
	$(call github_pr_add_comment,$(GIT_REPO_SLUG),$(PR_NUMBER),/merge-when-ready delete-branch release-note-not-required docs-not-required)
	echo "Added labels to pull request $(PR_NUMBER): merge-when-ready, release-note-not-required, docs-not-required & delete-branch"

## Update the AUTHORS.md file.
update-authors:
ifndef GITHUB_TOKEN
	$(error GITHUB_TOKEN must be set)
endif
	@echo "# Calico authors" > AUTHORS.md
	@echo "" >> AUTHORS.md
	@echo "This file is auto-generated based on commit records reported" >> AUTHORS.md
	@echo "by git for the projectcalico/calico repository. It is ordered alphabetically." >> AUTHORS.md
	@echo "" >> AUTHORS.md
	@docker run -ti --rm --net=host \
		-v $(REPO_ROOT):/code \
		-w /code \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		python:3 \
		bash -c '/usr/local/bin/python release/get-contributors.py >> /code/AUTHORS.md'

release/release:
	$(MAKE) -C release

bin/metadata.yaml: release/release
	mkdir -p bin
	./release/release release metadata --dir bin/

###############################################################################
# Post-release validation
###############################################################################
postrelease-checks: bin/yq
	$(MAKE) -C hack/postrelease/calient docker-test_all \
		CALICO_VERSION=$(call calico_versions_get_val,$(CALICO_VERSIONS_CALIENT_VERSION_KEY)) \
		CHART_RELEASE=$(call calico_versions_get_val,$(CALICO_VERSIONS_HELM_RELEASE_KEY)) \
		OPERATOR_VERSION=$(call calico_versions_get_val,$(CALICO_VERSIONS_OPERATOR_VERSION_KEY))

