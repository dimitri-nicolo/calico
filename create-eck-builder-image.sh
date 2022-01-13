#!/bin/bash

OPERATOR_IMAGE=$1
UBI_VERSION=$2
GO_VERSION=$3

cd cloud-on-k8s

VERSION=$(cat VERSION)
ECK_GO_LDFLAGS = -X github.com/elastic/cloud-on-k8s/pkg/about.version=$(VERSION) \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildHash=$(git rev-parse --short=8 --verify HEAD) \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
	-X github.com/elastic/cloud-on-k8s/pkg/about.buildSnapshot=false

sed -i "s/ubi-minimal\:[[:digit:]].[[:digit:]]\+/ubi-minimal\:${UBI_VERSION}/g" Dockerfile
sed -i "s/golang\:[[:digit:]].[[:digit:]]\+.[[:digit:]]/golang\:${GO_VERSION}/g" Dockerfile

DOCKER_BUILDKIT=1 docker build . \
  --progress=plain \
  --build-arg GO_LDFLAGS=$ECK_GO_LDFLAGS \
  --build-arg GO_TAGS=$GO_TAGS \
  --build-arg VERSION=$VERSION \
  -t $OPERATOR_IMAGE

git checkout Dockerfile
