JUNIT_FILE := xunit.xml
JUNIT_PARAMS := --junit-xml=$(JUNIT_FILE)

PARALLEL ?= 0
PARALLEL_PARAMS = --numprocesses=$(PARALLEL)

GIT_TOPLEVEL := $(shell git -c safe.directory='*' rev-parse --show-toplevel)

DOCKER_TEST_IMAGE_NAME = python:3
DOCKER_REQUIRED_ENV_VARS = GITHUB_TOKEN QUAY_TOKEN
DOCKER_OPTIONAL_ENV_VARS ?= $(shell python3 variables.py variable-names)
DOCKER_ENV_VARS = $(DOCKER_REQUIRED_ENV_VARS) $(DOCKER_OPTIONAL_ENV_VARS)

DOCKER_COMMAND = docker run -i --rm \
                 -v $(GIT_TOPLEVEL):/code \
                 -w /code/hack/postrelease/calient \
                 -v $(HOME)/.aws:/root/.aws:ro \
                 $(addprefix -e ,$(DOCKER_ENV_VARS)) \
                 $(DOCKER_TEST_IMAGE_NAME) \
                 make requirements

require_var = $(if \
		$(filter-out undefined,$(origin $(1))),\
		,\
		$(error The variable $(1) is required but is not defined) \
	)

require-var-%:
	$(call require_var,$*)

require-vars-%:
	$(foreach var,$(subst -, ,$*),$(call require_var,$(var)))

install_rpm: /usr/bin/rpmspec

/usr/bin/rpmspec:
	$(info *** Installing apt package `rpm` (for `rpmspec`))
	@apt-get -qq update
	@apt-get -qq install --no-install-recommends rpm > /dev/null

default:
	pytest -v

lint:
	pylint **/*.py
format:
	black **/*.py

show-variables:
	@python3 variables.py variables

docker-%:
	@$(DOCKER_COMMAND) $*

requirements: requirements.txt
	$(info *** Installing python packages from requirements.txt)
	@pip3 install --progress-bar off -q --root-user-action=ignore -U -r requirements.txt

test_all: require-vars-GITHUB_TOKEN-QUAY_TOKEN install_rpm
	pytest -v $(JUNIT_PARAMS) $(PARALLEL_PARAMS)

test_calico_release_cut: require-vars-GITHUB_TOKEN-QUAY_TOKEN
	pytest -v $(JUNIT_PARAMS) $(PARALLEL_PARAMS) -m calico_release_cut

test_operator_release: require-vars-GITHUB_TOKEN-QUAY_TOKEN
	pytest -v $(JUNIT_PARAMS) $(PARALLEL_PARAMS) -m operator_release

test_postrelease_artifacts: install_rpm
	pytest -v $(JUNIT_PARAMS) $(PARALLEL_PARAMS) -m postrelease_artifacts

test_marker-%:
	pytest -v $(JUNIT_PARAMS) $(PARALLEL_PARAMS) -m $*
