JUNIT_FILE := xunit.xml
JUNIT_PARAMS := --junit-xml=$(JUNIT_FILE)

PARALLEL ?= 0
PARALLEL_PARAMS = --numprocesses=$(PARALLEL)

GIT_TOPLEVEL := $(shell git -c safe.directory='*' rev-parse --show-toplevel)

UID = $(shell id -u)

ifdef CI # If we're running in a CI environment ...
DOCKER_BUILDX_PROGRESS = tty
GIT_BRANCH_NAME = $(SEMAPHORE_GIT_WORKING_BRANCH)
DOCKER_TTY =
PARALLEL = 12
else # Otherwise, if we're not in CI...
DOCKER_BUILDX_PROGRESS = auto
GIT_BRANCH_NAME := $(shell git branch --show-current)
junk := $(shell tty)

ifeq ($(.SHELLSTATUS), 0) # If we're not in CI but we're using a TTY
DOCKER_TTY = --tty
else # Not in CI, not using a TTY
DOCKER_TTY =
endif # ifeq .SHELLSTATUS

endif # ifdef CI

DOCKER_FROM_IMAGE_NAME = python:3
DOCKER_TEST_IMAGE_NAME = calico_enterprise_postrelease:$(GIT_BRANCH_NAME)
DOCKER_REQUIRED_ENV_VARS = GOOGLE_APPLICATION_CREDENTIALS QUAY_TOKEN PARALLEL_PARAMS PARALLEL
DOCKER_OPTIONAL_ENV_VARS ?= $(shell python3 variables.py variable-names)
DOCKER_ENV_VARS = $(DOCKER_REQUIRED_ENV_VARS) $(DOCKER_OPTIONAL_ENV_VARS)

ifdef NO_DOCKER

DOCKER_COMMAND =

else

DOCKER_COMMAND = docker run --quiet -i $(DOCKER_TTY) --rm \
                 -v $(GIT_TOPLEVEL):/code \
                 -w /code/hack/postrelease/calient \
                 -v $(HOME)/.aws:/root/.aws \
                 -v $(GOOGLE_APPLICATION_CREDENTIALS):$(GOOGLE_APPLICATION_CREDENTIALS) \
                 $(addprefix -e ,$(DOCKER_ENV_VARS)) \
                 $(DOCKER_TEST_IMAGE_NAME)

endif # NO_DOCKER

build_docker_container: .docker_container.stamp

.docker_container.stamp:
	docker buildx build --load --pull --build-arg UID=$(UID) --progress $(DOCKER_BUILDX_PROGRESS) -t $(DOCKER_TEST_IMAGE_NAME) .

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

test_all: require-vars-GOOGLE_APPLICATION_CREDENTIALS-QUAY_TOKEN build_docker_container
	@$(DOCKER_COMMAND) pytest $(JUNIT_PARAMS) $(PARALLEL_PARAMS)

test_calico_release_cut: require-vars-QUAY_TOKEN build_docker_container
	@$(DOCKER_COMMAND) pytest $(JUNIT_PARAMS) $(PARALLEL_PARAMS) -m calico_release_cut

test_operator_release: require-vars-QUAY_TOKEN build_docker_container
	@$(DOCKER_COMMAND) pytest $(JUNIT_PARAMS) $(PARALLEL_PARAMS) -m operator_release

test_postrelease_artifacts: build_docker_container
	@$(DOCKER_COMMAND) pytest $(JUNIT_PARAMS) $(PARALLEL_PARAMS) -m postrelease_artifacts

test_marker-%: build_docker_container
	@$(DOCKER_COMMAND) pytest $(JUNIT_PARAMS) $(PARALLEL_PARAMS) -m $*

