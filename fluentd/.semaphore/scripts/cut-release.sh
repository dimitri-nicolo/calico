#!/usr/bin/env bash

DEV_TAG_SUFFIX=${DEV_TAG_SUFFIX:-calient-0.dev}
EXPECTED_RELEASE_TAG=$(git describe --tags --long --always --abbrev=12 --match "*dev*" | grep -P -o "^v\d*.\d*.\d*(-.*)?(?=-${DEV_TAG_SUFFIX})")

if [ -z "$EXPECTED_RELEASE_TAG" ]; then
    echo "Couldn't find expected release tag"
    exit 1
fi

echo "Cutting release for $EXPECTED_RELEASE_TAG."
make cut-release

git checkout "$EXPECTED_RELEASE_TAG"

make cut-release WINDOWS_RELEASE=true IMAGE_ONLY=true EXPECTED_RELEASE_TAG="$EXPECTED_RELEASE_TAG"
