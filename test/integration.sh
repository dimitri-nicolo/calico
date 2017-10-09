#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
#
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

set -o errexit
set -o nounset
set -o pipefail

# this script resides in the `test/` folder at the root of the project
KUBE_ROOT=$(realpath $(dirname "${BASH_SOURCE}")/../vendor/k8s.io/kubernetes)
source "${KUBE_ROOT}/hack/lib/init.sh"

runTests() {
  #kube::etcd::start

  ETCD_ENDPOINTS="http://127.0.0.1:2379" DATASTORE_TYPE="etcdv3" go test -v github.com/tigera/calico-k8sapiserver/test/integration/... --args -v 10 -logtostderr
}

# Run cleanup to stop etcd on interrupt or other kill signal.
#trap kube::etcd::cleanup EXIT

runTests

