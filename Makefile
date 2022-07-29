PACKAGE_NAME = github.com/projectcalico/calico

include metadata.mk
include lib.Makefile

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

MAKE_DIRS=$(shell ls -d */)

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
	$(MAKE) -C calico clean
	rm -f $(SUB_CHARTS)

generate:
	$(MAKE) gen-semaphore-yaml
	$(MAKE) -C api gen-files
	$(MAKE) -C libcalico-go gen-files
	$(MAKE) -C felix gen-files
	$(MAKE) -C app-policy protobuf
	$(MAKE) gen-manifests

gen-manifests: bin/helm
	# TODO: Ideally we don't need to do this, but the sub-charts
	# mess up manifest generation if they are present.
	rm -f $(SUB_CHARTS)
	cd ./manifests && \
		OPERATOR_VERSION=$(OPERATOR_VERSION) \
		CALICO_VERSION=$(CALICO_VERSION) \
		./generate.sh

# Build the tigera-operator helm chart.
SUB_CHARTS=charts/tigera-operator/charts/tigera-prometheus-operator.tgz
chart: bin/tigera-operator-$(GIT_VERSION).tgz
bin/tigera-operator-$(GIT_VERSION).tgz: bin/helm $(shell find ./charts/tigera-operator -type f) $(SUB_CHARTS)
	bin/helm package ./charts/tigera-operator \
	--destination ./bin/ \
	--version $(GIT_VERSION) \
	--app-version $(GIT_VERSION)

# Build the tigera-prometheus-operator.tgz helm chart.
bin/tigera-prometheus-operator-$(GIT_VERSION).tgz:
	bin/helm package ./charts/tigera-prometheus-operator \
	--destination ./bin/ \
	--version $(GIT_VERSION) \
	--app-version $(GIT_VERSION)

# Include the tigera-prometheus-operator helm chart as a sub-chart.
charts/tigera-operator/charts/tigera-prometheus-operator.tgz: bin/tigera-prometheus-operator-$(GIT_VERSION).tgz
	mkdir -p $(@D)
	cp bin/tigera-prometheus-operator-$(GIT_VERSION).tgz $@

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
e2e-test:
	$(MAKE) -C e2e build
	$(MAKE) -C node kind-k8st-setup
	KUBECONFIG=$(KIND_KUBECONFIG) ./e2e/bin/e2e.test -ginkgo.focus=$(E2E_FOCUS)

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

gen-semaphore-yaml:
	cd .semaphore && ./generate-semaphore-yaml.sh
