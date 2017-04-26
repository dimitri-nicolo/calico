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

# Contains swagger related util functions.
#
set -o errexit
set -o nounset
set -o pipefail

# The root of the build/dist directory
KUBE_ROOT="$(cd "$(dirname "${BASH_SOURCE}")/../.." && pwd -P)"

# Generates types_swagger_doc_generated file for the given group version.
# $1: Name of the group version
# $2: Path to the directory where types.go for that group version exists. This
# is the directory where the file will be generated.
kube::swagger::gen_types_swagger_doc() {
  local group_version=$1
  local gv_dir=$2
  local TMPFILE="${TMPDIR:-/tmp}/types_swagger_doc_generated.$(date +%s).go"

  echo "Generating swagger type docs for ${group_version} at ${gv_dir}"

  sed 's/YEAR/2016/' hack/boilerplate/boilerplate.go.txt > "$TMPFILE"
  echo "package ${group_version##*/}" >> "$TMPFILE"
  cat >> "$TMPFILE" <<EOF

// This file contains a collection of methods that can be used from go-restful to
// generate Swagger API documentation for its models. Please read this PR for more
// information on the implementation: https://github.com/emicklei/go-restful/pull/215
//
// TODOs are ignored from the parser (e.g. TODO(andronat):... || TODO:...) if and only if
// they are on one line! For multiple line or blocks that you want to ignore use ---.
// Any context after a --- is ignored.
//
// Those methods can be generated by using hack/update-generated-swagger-docs.sh

// AUTO-GENERATED FUNCTIONS START HERE
EOF

  go run cmd/genswaggertypedocs/swagger_type_docs.go -s \
    "${gv_dir}/types.go" \
    -f - \
    >>  "$TMPFILE"

  echo "// AUTO-GENERATED FUNCTIONS END HERE" >> "$TMPFILE"

  gofmt -w -s "$TMPFILE"
  mv "$TMPFILE" ""${gv_dir}"/types_swagger_doc_generated.go"
}

# Generates API reference docs for the given API group versions.
# Required env vars:
#   GROUP_VERSIONS: Array of group versions to be included in the reference
#   docs.
#   GV_DIRS: Array of root directories for those group versions.
# Input vars:
#   $1: Root directory path for swagger spec
#   $2: Root directory path where the reference docs should be generated.
kube::swagger::gen_api_ref_docs() {
    : "${GROUP_VERSIONS?Must set GROUP_VERSIONS env var}"
    : "${GV_DIRS?Must set GV_DIRS env var}"

  echo "Generating API reference docs for group versions: ${GROUP_VERSIONS[@]}, at dirs: ${GV_DIRS[@]}"
  GROUP_VERSIONS=(${GROUP_VERSIONS[@]})
  GV_DIRS=(${GV_DIRS[@]})
  local swagger_spec_path=${1}
  local output_dir=${2}
  echo "Reading swagger spec from: ${swagger_spec_path}"
  echo "Generating the docs at: ${output_dir}"

  # Use REPO_DIR if provided so we can set it to the host-resolvable path
  # to the repo root if we are running this script from a container with
  # docker mounted in as a volume.
  # We pass the host output dir as the source dir to `docker run -v`, but use
  # the regular one to compute diff (they will be the same if running this
  # test on the host, potentially different if running in a container).
  local repo_dir=${REPO_DIR:-"${KUBE_ROOT}"}
  local tmp_subpath="_output/generated_html"
  local output_tmp_in_host="${repo_dir}/${tmp_subpath}"
  local output_tmp="${KUBE_ROOT}/${tmp_subpath}"

  echo "Generating api reference docs at ${output_tmp}"

  for ver in "${GROUP_VERSIONS[@]}"; do
    mkdir -p "${output_tmp}/${ver}"
  done

  user_flags="-u $(id -u)"
  if [[ $(uname) == "Darwin" ]]; then
    # mapping in a uid from OS X doesn't make any sense
    user_flags=""
  fi

  for i in "${!GROUP_VERSIONS[@]}"; do
    local ver=${GROUP_VERSIONS[i]}
    local dir=${GV_DIRS[i]}
    local tmp_in_host="${output_tmp_in_host}/${ver}"
    local register_file="${dir}/register.go"
    local swagger_json_name="$(kube::util::gv-to-swagger-name "${ver}")"

    docker run ${user_flags} \
      --rm -v "${tmp_in_host}":/output:z \
      -v "${swagger_spec_path}":/swagger-source:z \
      -v "${register_file}":/register.go:z \
      --net=host -e "https_proxy=${KUBERNETES_HTTPS_PROXY:-}" \
      gcr.io/google_containers/gen-swagger-docs:v8 \
      "${swagger_json_name}"
  done

  # Check if we actually changed anything
  pushd "${output_tmp}" > /dev/null
  touch .generated_html
  find . -type f | cut -sd / -f 2- | LC_ALL=C sort > .generated_html
  popd > /dev/null

  while read file; do
    if [[ -e "${output_dir}/${file}" && -e "${output_tmp}/${file}" ]]; then
      echo "comparing ${output_dir}/${file} with ${output_tmp}/${file}"

      # By now, the contents should be normalized and stripped of any
      # auto-managed content.
      if diff -NauprB -I 'Last update' "${output_dir}/${file}" "${output_tmp}/${file}" >/dev/null; then
        # actual contents same, overwrite generated with original.
        cp "${output_dir}/${file}" "${output_tmp}/${file}"
      fi
    fi
  done <"${output_tmp}/.generated_html"

  echo "Moving api reference docs from ${output_tmp} to ${output_dir}"

  cp -af "${output_tmp}"/* "${output_dir}"
  rm -r "${output_tmp}"
}
