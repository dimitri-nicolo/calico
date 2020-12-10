.PHONY: ci cd image

GIT_VERSION?=$(shell git describe --tags --dirty --always --long --abbrev=12)

KIBANA_IMAGE?=gcr.io/unique-caldron-775/cnx/tigera/kibana

GTM_INTEGRATION?=disable

ifdef IMAGE_PREFIX
	LATEST_TAG=$(IMAGE_PREFIX)-latest
else
	LATEST_TAG=latest
endif


image:
	docker build --build-arg GTM_INTEGRATION=$(GTM_INTEGRATION) --pull -t $(KIBANA_IMAGE):$(LATEST_TAG) ./

ci: image

cd: image
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
ifdef IMAGE_PREFIX
	$(eval BRANCH_NAME_TAG := $(IMAGE_PREFIX)-$(BRANCH_NAME))
	$(eval GIT_VERSION_TAG := $(IMAGE_PREFIX)-$(GIT_VERSION))
else
	$(eval BRANCH_NAME_TAG := $(BRANCH_NAME))
	$(eval GIT_VERSION_TAG := $(GIT_VERSION))
endif
	docker tag $(KIBANA_IMAGE):$(LATEST_TAG) \
	           $(KIBANA_IMAGE):$(BRANCH_NAME_TAG)
	docker push $(KIBANA_IMAGE):$(BRANCH_NAME_TAG)

	docker tag $(KIBANA_IMAGE):$(LATEST_TAG) \
	           $(KIBANA_IMAGE):$(GIT_VERSION_TAG)
	docker push $(KIBANA_IMAGE):$(GIT_VERSION_TAG)

dev:
	docker-compose -f docker-compose.dev.yml up --build
