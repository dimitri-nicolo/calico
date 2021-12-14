#!/usr/bin/env bash

EXPECTED_RELEASE_TAG=$(echo $(git describe --tags --long --always --abbrev=12 --match "*dev*") | grep -P -o "^v\d*.\d*.\d*")

if [ -z "$EXPECTED_RELEASE_TAG" ]
then
    echo "Couldn't find expected release tag"
    exit 1
fi

echo "Cutting release for $EXPECTED_RELEASE_TAG."
make cut-release

git checkout $EXPECTED_RELEASE_TAG

make cut-release TESLA=true IMAGE_ONLY=true EXPECTED_RELEASE_TAG=$EXPECTED_RELEASE_TAG
