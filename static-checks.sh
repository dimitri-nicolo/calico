#!/usr/bin/env bash

PROJECT_DIR=${1-'.'}
GO_DIRS=$(find "${PROJECT_DIR}" -type f -name "*.go" -not -path "${PROJECT_DIR}/.go-pkg-cache/*" -exec dirname {} \; | sort -u)

enforce() {
  local type=$1
  local output=$2
  if test -n "$output"; then
      echo >&2 "failed: $type"
      echo >&2 "$output"
      exit 1
  fi
}

# preemptively download modules
# to prevent STDERR from being filled with "go mod" output later
go mod download

# various linters
enforce "formatting" "$(gofmt -l -s $GO_DIRS)"
enforce "imports"    "$(goimports -l $GO_DIRS)"
enforce "vetting"    "$(go vet "${PROJECT_DIR}/..." 2>&1)"
enforce "linting"    "$(golint "${PROJECT_DIR}/..." 2>&1)"

# go modules
enforce "mod tidy"   "$(go mod tidy && git ls-files -m | awk '/^go.mod$/ {print $0, "was modified"}')"
enforce "mod verify" "$(go mod verify | grep -v "all modules verified")"

