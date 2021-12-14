#!/usr/bin/env bash

if [ ! -z "$CONFIRM" ]
then
  RELEASE_VERSION=$(echo $(git describe --tags --exact-match --exclude "*dev*"))
else
  RELEASE_VERSION=$(echo $(git describe --tags --long --always --abbrev=12 --match "*dev*") | grep -P -o "^v\d*.\d*.\d*")
fi

if [ -z "$RELEASE_VERSION" ]
then
    echo "Current commit has no release version tagged"
    exit 1
fi

make release-publish-binaries VERSION=$RELEASE_VERSION
