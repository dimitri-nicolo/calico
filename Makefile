.SUFFIXES:

VERSION?=master
IMAGETAG:=$(shell git describe --tags --dirty --always --long)

ES_PROXY_IMAGE?=gcr.io/unique-caldron-775/cnx/tigera/es-proxy
ES_PROXY_CREATED?=.es-proxy.created



$(ES_PROXY_CREATED): Dockerfile haproxy.cfg rsyslog.conf
	docker build -f Dockerfile -t tigera/es-proxy:latest .
	touch $@

.PHONY: release
release: clean
ifndef VERSION
	$(error VERSION is undefined - run using make release VERSION=v.X.Y.Z) 
endif
	git tag $(VERSION)

	# Check to make sure the tag isn't "dirty"
	if git describe --tags --dirty | grep dirty; \
	then echo current git working tree is "dirty". Make sure you do not have any uncommitted changes ;false; fi

	$(MAKE) image

	# Retag images with correct version and registry prefix
	docker tag tigera/es-proxy:latest $(ES_PROXY_IMAGE):$(VERSION)

	# Check that image were created recently and that the IDs of the versioned and latest image match
	@docker images --format "{{.CreatedAt}}\tID:{{.ID}}\t{{.Repository}}:{{.Tag}}" tigera/es-proxy:latest
	@docker images --format "{{.CreatedAt}}\tID:{{.ID}}\t{{.Repository}}:{{.Tag}}" $(ES_PROXY_IMAGE):$(VERSION)

	@echo "\nNow push the tag and image."
	@echo "git push origin $(VERSION)"
	@echo "gcloud auth configure-docker"
	@echo "docker push $(ES_PROXY_IMAGE):$(VERSION)"
	@echo "\nIf this release version is the newest stable release, also tag and push the"
	@echo "image with the 'latest' tag"
	@echo "docker tag $(ES_PROXY_IMAGE):$(VERSION) $(ES_PROXY_IMAGE):latest"
	@echo "docker push $(ES_PROXY_IMAGE):latest"


.PHONY: image
image: $(ES_PROXY_CREATED)


ci: image


cd:
	@echo pushing $(IMAGETAG)
	docker tag tigera/es-proxy:latest $(ES_PROXY_IMAGE):$(IMAGETAG)
	docker push $(ES_PROXY_IMAGE):$(IMAGETAG)


.PHONY: clean
clean:
	-rm -rf *.tar
	-rm -f $(ES_PROXY_CREATED)
