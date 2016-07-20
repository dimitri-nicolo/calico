#!/usr/bin/env bash
# Copyright (c) 2016 Tigera, Inc. All rights reserved.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e
set -x

GOPATH=/go
PATH=$PATH:$GOPATH/bin



mkdir -p $GOPATH/src/github.com/tigera/
cp -r /calicoq $GOPATH/src/github.com/tigera/

cd $GOPATH/src/github.com/tigera/calicoq
#rm -rf vendor
#glide install

REVISION=$(git rev-parse HEAD)
SHORT_REVISION=$(git rev-parse --short HEAD)
if git describe --tags; then
  DESCRIPTION=$(git describe --tags)
else
  DESCRIPTION="$SHORT_REVISION"
fi
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

sed -i "s/__BUILD_DATE__/${DATE}/" calicoq/commands/build_info.go
sed -i "s/__GIT_REVISION__/${REVISION}/" calicoq/commands/build_info.go
sed -i "s/__GIT_DESCRIPTION__/${DESCRIPTION}/" calicoq/commands/build_info.go

go build -o /calicoq/release/calicoq-$DESCRIPTION calicoq/*.go
cd /calicoq/release
ln -f calicoq-$DESCRIPTION calicoq

set +x

echo
echo "Binaries release/(calicoq|calicoq-$DESCRIPTION) created"
