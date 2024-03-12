#!/bin/bash -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
BASE_DIR=$(dirname $SCRIPT_DIR)

pushd $BASE_DIR > /dev/null

docker buildx build --pull -t tigera/fluentd .
