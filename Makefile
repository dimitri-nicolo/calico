.PHONY: ci cd

ci:
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	docker build --pull -t gcr.io/unique-caldron-775/cnx/tigera/kibana:$(BRANCH_NAME) ./

GIT_VERSION?=$(shell git describe --tags --dirty --always)

# The image name for GCR
PUSH_IMAGE?=gcr.io/unique-caldron-775/cnx/tigera/kibana

cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	docker push $(PUSH_IMAGE):$(BRANCH_NAME)
	docker tag $(PUSH_IMAGE):$(BRANCH_NAME) \
	           $(PUSH_IMAGE):$(GIT_VERSION)
	docker push $(PUSH_IMAGE):$(GIT_VERSION)
