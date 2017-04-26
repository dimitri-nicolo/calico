#!/bin/bash

# Copyright 2015 The Kubernetes Authors.
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

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..
source "${KUBE_ROOT}/hack/lib/init.sh"

kube::golang::setup_env

make -C "${KUBE_ROOT}" WHAT=cmd/genswaggertypedocs

# Find binary
genswaggertypedocs=$(kube::util::find-binary "genswaggertypedocs")

if [[ ! -x "$genswaggertypedocs" ]]; then
  {
    echo "It looks as if you don't have a compiled genswaggertypedocs binary"
    echo
    echo "If you are running from a clone of the git repo, please run"
    echo "'make WHAT=cmd/genswaggertypedocs'."
  } >&2
  exit 1
fi

DIFFROOT="${KUBE_ROOT}/pkg"
TMP_DIFFROOT="${KUBE_ROOT}/_tmp/pkg"
_tmp="${KUBE_ROOT}/_tmp"

cleanup() {
  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_DIFFROOT}"
cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}/"

"${KUBE_ROOT}/hack/update-generated-swagger-docs.sh"
echo "diffing ${DIFFROOT} against freshly generated swagger type documentation"
ret=0
diff -Naupr -I 'Auto generated by' "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?
cp -a "${TMP_DIFFROOT}"/* "${DIFFROOT}/"
if [[ $ret -eq 0 ]]
then
  echo "${DIFFROOT} up to date."
else
  echo "${DIFFROOT} is out of date. Please run hack/update-generated-swagger-docs.sh"
  exit 1
fi
