.PHONY: ci cd

KIBANA_VERSION?=6.4.3

ci:
	docker build --pull -t gcr.io/unique-caldron-775/cnx/tigera/kibana:$(KIBANA_VERSION) ./

cd:
ifndef CONFIRM
	$(error CONFIRM is undefined - run using make <target> CONFIRM=true)
endif
	docker push gcr.io/unique-caldron-775/cnx/tigera/kibana:$(KIBANA_VERSION)
