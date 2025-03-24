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
	$(MAKE) -C fluent-bit clean
	$(MAKE) -C kube-controllers clean
	$(MAKE) -C libcalico-go clean
	$(MAKE) -C node clean
	$(MAKE) -C pod2daemon clean
	$(MAKE) -C key-cert-provisioner clean
	$(MAKE) -C typha clean
	$(MAKE) -C release clean
	$(MAKE) -C selinux clean
	rm -rf ./bin
	rm -f $(SUB_CHARTS)
	rm -rf _release_archive
	rm -f manifests/ocp.tgz

ci-preflight-checks:
	$(MAKE) check-go-mod
	$(MAKE) check-dockerfiles
	$(MAKE) check-gotchas
	$(MAKE) check-language || true # Enterprise hasn't been cleaned up yet.
	$(MAKE) check-release-cut-promotions
	$(MAKE) generate
	$(MAKE) fix-all
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
	@if [ -e manifests/ocp/operator.tigera.io_tenants.yaml ]; then \
		echo; \
		echo "The operator Tenants CRD has been added to manifests/ocp/ but should only be included for multi-tenant purposes";\
		echo; \
		exit 1; \
	fi
	@if grep tenants\.operator\.tigera\.io manifests/tigera-operator.yaml > /dev/null ; then \
		echo; \
		echo "The operator manifest includes the Tenants CRD but should not, please investigate and remove whatever caused its inclusion";\
		echo; \
		exit 1; \
	fi

check-go-mod:
	$(DOCKER_GO_BUILD) sh -c '$(GIT_CONFIG_SSH) ./hack/check-go-mod.sh'

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

protobuf:
	$(MAKE) -C app-policy protobuf
	$(MAKE) -C cni-plugin protobuf
	$(MAKE) -C egress-gateway protobuf
	$(MAKE) -C felix protobuf
	$(MAKE) -C pod2daemon protobuf
	$(MAKE) -C goldmane protobuf

generate:
	$(MAKE) gen-semaphore-yaml
	$(MAKE) protobuf
	$(MAKE) -C lib gen-files
	$(MAKE) -C api gen-files
	$(MAKE) -C libcalico-go gen-files
	$(MAKE) -C felix gen-files
	$(MAKE) -C goldmane gen-files
	$(MAKE) gen-manifests
	$(MAKE) fix-changed

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
# Get operator CRDs from the operator repo, OPERATOR_BRANCH must be set
get-operator-crds: var-require-all-OPERATOR_BRANCH
	cd ./charts/tigera-operator/crds/ && \
	for file in operator.tigera.io_*.yaml; do echo "downloading $$file from operator repo" && curl -fsSL https://raw.githubusercontent.com/tigera/operator/$(OPERATOR_BRANCH)/pkg/crds/operator/$${file} -o $${file}; done
	cp -vLR ./charts/tigera-operator/crds/ ./charts/multi-tenant-crds/. && \
	cd ./charts/multi-tenant-crds/crds && \
	curl -fsSOL https://raw.githubusercontent.com/tigera/operator/$(OPERATOR_BRANCH)/pkg/crds/operator/operator.tigera.io_tenants.yaml && \
	for file in $(MULTI_TENANCY_CRDS_FILE_CHANGES); do \
		echo "Update CRD $$file to be Namespaced"; \
		sed -i 's/scope: Cluster/scope: Namespaced/g' $$file; \
	done
	$(MAKE) fix-changed

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

PUBLISH_TARGETS := chart-release release-archive multi-tenant-crds selinux non-cluster-host-rpms

publish: var-require-all-CHART_RELEASE-RELEASE_STREAM-REGISTRY $(addprefix publish-,$(PUBLISH_TARGETS))

# TODO: We're moving selinux RPMs into the same repository
# as non-cluster host RPMs. We may want to remove this
# location in the future, but we should keep it here in case
# users actually use it.
publish-selinux:
	$(MAKE) -C selinux publish

chart-release: var-require-all-CHART_RELEASE-RELEASE_STREAM chart
	mv ./bin/tigera-operator-$(RELEASE_STREAM).tgz ./bin/tigera-operator-$(RELEASE_STREAM)-$(CHART_RELEASE).tgz

publish-chart-release: chart-release
	@aws --profile helm s3 cp ./bin/tigera-operator-$(RELEASE_STREAM)-$(CHART_RELEASE).tgz s3://tigera-public/ee/charts/ --acl public-read

publish-release-archive: release-archive
	$(MAKE) -f release-archive.mk publish-release-archive
release-archive: manifests/ocp.tgz
	$(MAKE) -f release-archive.mk release-archive

.PHONY: build-non-cluster-host-rpms publish-non-cluster-host-rpms

# Build the non-cluster host RPMs for a given sub-project
# TODO: find a concise way to check if things are built already and skip if they are
build-non-cluster-host-rpms-%:
	@$(MAKE) -C $* package

# Ensure that all of our non-cluster host RPMs are built before we try to publish them
build-non-cluster-host-rpms: $(addprefix build-non-cluster-host-rpms-,$(NON_CLUSTER_HOST_SUBDIRS))

publish-non-cluster-host-rpms: var-require-all-VERSION build-non-cluster-host-rpms
	VERSION=$(RELEASE_STREAM) hack/publish_rpms_to_repo.sh

SUB_CHARTS=charts/tigera-operator/charts/tigera-prometheus-operator.tgz
chart: tigera-operator-release tigera-operator-master multi-tenant-crds-release tigera-prometheus-operator-release

tigera-operator-release: bin/tigera-operator-$(chartVersion).tgz

# Build the multi-tenant-crds helm chart.
multi-tenant-crds-release: bin/multi-tenant-crds-$(chartVersion)-$(CHART_RELEASE).tgz
bin/multi-tenant-crds-$(chartVersion)-$(CHART_RELEASE).tgz: bin/helm
	bin/helm package ./charts/multi-tenant-crds \
	--destination ./bin/ \
	--version $(chartVersion) \
	--app-version $(appVersion)

publish-multi-tenant-crds: multi-tenant-crds-release
	aws --profile helm \
		s3 cp \
		bin/multi-tenant-crds-$(chartVersion)-$(CHART_RELEASE).tgz \
		s3://tigera-public/ee/charts/ \
		--acl public-read


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
bin/tigera-prometheus-operator-$(chartVersion).tgz: bin/helm
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
	$(MAKE) -C key-cert-provisioner image IMAGETAG=$(GIT_VERSION) VALIDARCHES=$(ARCH)
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
ADMINPOLICY_SUPPORTED_FEATURES ?= "AdminNetworkPolicy,BaselineAdminNetworkPolicy"
ADMINPOLICY_UNSUPPORTED_FEATURES ?= ""
e2e-test:
	$(MAKE) -C e2e build
	$(MAKE) -C node kind-k8st-setup
	KUBECONFIG=$(KIND_KUBECONFIG) ./e2e/bin/k8s/e2e.test -ginkgo.focus=$(E2E_FOCUS)
	# Disable AdminNetworkPolicy Conformance tests since it's flaky. The issue is being tracked at https://tigera.atlassian.net/browse/CORE-10742
	# KUBECONFIG=$(KIND_KUBECONFIG) ./e2e/bin/adminpolicy/e2e.test \
	#  -exempt-features=$(ADMINPOLICY_UNSUPPORTED_FEATURES) \
	#  -supported-features=$(ADMINPOLICY_SUPPORTED_FEATURES)

e2e-test-adminpolicy:
	$(MAKE) -C e2e build
	$(MAKE) -C node kind-k8st-setup
	KUBECONFIG=$(KIND_KUBECONFIG) ./e2e/bin/adminpolicy/e2e.test \
	  -exempt-features=$(ADMINPOLICY_UNSUPPORTED_FEATURES) \
	  -supported-features=$(ADMINPOLICY_SUPPORTED_FEATURES)

###############################################################################
# Release logic below
###############################################################################
.PHONY: release release-publish create-release-branch release-test build-openstack publish-openstack release-notes
# Build the release tool.
release/bin/release: $(shell find ./release -type f -name '*.go')
	$(MAKE) -C release

# Install gh for interacting with GitHub.
bin/gh:
	curl -sSL -o bin/gh.tgz https://github.com/cli/cli/releases/download/v$(GITHUB_CLI_VERSION)/gh_$(GITHUB_CLI_VERSION)_linux_amd64.tar.gz
	tar -zxvf bin/gh.tgz -C bin/ gh_$(GITHUB_CLI_VERSION)_linux_amd64/bin/gh --strip-components=2
	chmod +x $@
	rm bin/gh.tgz

# Create updates for pre-release
release-prep: release/bin/release bin/gh var-require-all-HASHRELEASE-RELEASE_VERSION-HELM_RELEASE-OPERATOR_VERSION-REGISTRY var-require-one-of-CONFIRM-DRYRUN
	@REGISTRIES=$(REGISTRY) release/bin/release release prep

# Install ghr for publishing to github.
bin/ghr:
	$(DOCKER_RUN) -e GOBIN=/go/src/$(PACKAGE_NAME)/bin/ $(CALICO_BUILD) go install github.com/tcnksm/ghr@$(GHR_VERSION)

# Build a release.
release: release/bin/release
	@release/bin/release release build

# Publish an already built release.
release-publish: release/bin/release bin/gh
	@release/bin/release release publish

# Create a release branch.
create-release-branch: release/bin/release
	@release/bin/release branch cut -git-publish

# Test the release code
release-test:
	$(DOCKER_RUN) $(CALICO_BUILD) ginkgo -cover -r release/pkg

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
bin/ocp.tgz manifests/ocp.tgz: manifests/ocp/
	tar czvf $@ -C manifests/ \
		--exclude=tigera-enterprise-resources.yaml \
		--exclude=tigera-prometheus-operator.yaml \
		--exclude=00-namespace-calico-apiserver.yaml \
		--exclude=00-namespace-calico-system.yaml \
		ocp

## Generates release notes for the given version.
.PHONY: release-notes
release-notes:
	@$(MAKE) -C release release-notes

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

release/bin/release: $(shell find ./release -type f -name '*.go')
	$(MAKE) -C release

release/metadata: release/bin/release var-require-all-METADATA_DIR
	@release/bin/release release metadata

update-pins: update-go-build-pin update-calico-base-pin

###############################################################################
# Post-release validation
###############################################################################
postrelease-checks: bin/yq
	$(MAKE) -C hack/postrelease/calient docker-test_all \
		CALICO_VERSION=$(call calico_versions_get_val,$(CALICO_VERSIONS_CALIENT_VERSION_KEY)) \
		CHART_RELEASE=$(call calico_versions_get_val,$(CALICO_VERSIONS_HELM_RELEASE_KEY)) \
		OPERATOR_VERSION=$(call calico_versions_get_val,$(CALICO_VERSIONS_OPERATOR_VERSION_KEY))

