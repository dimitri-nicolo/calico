#!/bin/bash
# Copyright (c) 2021 Tigera, Inc. All rights reserved.
set -xe

FV_PROMETHEUS_SERVICE_TEST_IMAGE=${FV_PROMETHEUS_SERVICE_TEST_IMAGE:-"tigera/prometheus-service:latest"}
TEST_CONTAINER_NAME="fv-prometheus-proxy-test"
PACKAGE_ROOT=${PACKAGE_ROOT:-$(pwd)/..}
GO_BUILD_IMAGE=${GO_BUILD_IMAGE:-"calico/go-build:v0.53"}

function run_fvs()
{
	# Run test - if this fails output the logs from the proxy container. Running ginkgo with the--failFast flag
	# is useful if you are debugging issues since we do not correlate tests with output from the proxy container.
	docker run \
		--rm \
		--net=host \
		-v ${GOMOD_CACHE}/..:/go/pkg:rw \
		-v ${PACKAGE_ROOT}/.go-pkg-cache:/home/user/.cache/go-build:rw \
		-v ${PACKAGE_ROOT}:/${PACKAGE_NAME}:rw \
		-v ${PACKAGE_ROOT}/report:/report:rw \
		-e LOCAL_USER_ID=$(id -u) \
		-w /${PACKAGE_NAME} \
		${GO_BUILD_IMAGE} \
		sh -c "ginkgo ./fv/" || (docker logs ${TEST_CONTAINER_NAME} && false)
}

function run_proxy()
{

	docker rm -f ${TEST_CONTAINER_NAME} || true

	# Start test image
	docker run \
		--net=host \
		--detach \
		-v ${PACKAGE_ROOT}/test:/test:ro \
    -e LISTEN_ADDR="localhost:8090" \
		-e LOG_LEVEL=debug \
		--name ${TEST_CONTAINER_NAME} \
		${FV_PROMETHEUS_SERVICE_TEST_IMAGE}
}

run_proxy
run_fvs
