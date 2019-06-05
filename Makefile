.PHONY: ci cd

ci:
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	docker build --pull -t gcr.io/unique-caldron-775/cnx/tigera/kibana:$(BRANCH_NAME) ./

cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
ifndef BRANCH_NAME
	$(error BRANCH_NAME is undefined - run using make <target> BRANCH_NAME=var or set an environment variable)
endif
	docker push gcr.io/unique-caldron-775/cnx/tigera/kibana:$(BRANCH_NAME)
