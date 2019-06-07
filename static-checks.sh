#!/usr/bin/env bash

PROJECT_DIR=${1-'.'}
GO_DIRS=$(find "${PROJECT_DIR}" -type f -name "*.go" -not -path "${PROJECT_DIR}/.go-pkg-cache/*" -exec dirname {} \; | sort -u)

enforce() {
  local type=$1
  local output=$2
  if test -n "$output"; then
      echo >&2 "Some files failed $type checks"
      echo >&2 "$output"
      exit 1
  fi
}

enforce "formatting" "$(gofmt -l -s $GO_DIRS)"
enforce "vetting"    "$(go vet "${PROJECT_DIR}/..." 2>&1)"
enforce "linting"    "$(golint "${PROJECT_DIR}/..." 2>&1)"

