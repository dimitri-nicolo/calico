#!/bin/bash

# Copyright 2016 The Kubernetes Authors.
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

SPECROOT="${KUBE_ROOT}/federation/apis/openapi-spec"
TMP_SPECROOT="${KUBE_ROOT}/_tmp_federation/openapi-spec"
_tmp="${KUBE_ROOT}/_tmp_federation"

mkdir -p "${_tmp}"
cp -a "${SPECROOT}" "${TMP_SPECROOT}"
trap "cp -a ${TMP_SPECROOT} ${SPECROOT}/..; rm -rf ${_tmp}" EXIT SIGINT
rm ${SPECROOT}/*

"${KUBE_ROOT}/hack/update-federation-openapi-spec.sh"
echo "diffing ${SPECROOT} against freshly generated federation openapi spec"
ret=0
diff -Naupr -I 'Auto generated by' "${SPECROOT}" "${TMP_SPECROOT}" || ret=$?
if [[ $ret -eq 0 ]]
then
  echo "${SPECROOT} up to date."
else
  echo "${SPECROOT} is out of date. Please run hack/update-federation-openapi-spec.sh"
  exit 1
fi

# ex: ts=2 sw=2 et filetype=sh
