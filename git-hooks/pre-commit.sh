#!/bin/bash

set -e

# Make sure we get some output if an unexpected command happens to fail.
trap "echo git pre-commit hook failed." EXIT

# Redirect output to stderr.
exec 1>&2

changed_go_files=$(git diff --cached --name-only | grep -E '.go$' || true)

all_go_files=$(find . -name '*.go' | grep -v go-pkg-cache)

copyright_check_failed=false
copyright_owner="Tigera, Inc"

[[ -f "git-hooks/settings.sh" ]] && source "git-hooks/settings.sh"

echo "Running static-checks..."
static_checks_failed=false
if ! make static-checks; then
  echo " Static checks failed"
  static_checks_failed=true
fi

# Check copyright statement has been updated.
echo "Checking changed files for copyright statements..."
year=$(date +'%Y')
copyright_re="Copyright \(c\) .*${year}.* ${copyright_owner}\. All rights reserved\."

for filename in $changed_go_files; do
  if [ -e "${filename}" ] && ! grep -q -E "$copyright_re" "${filename}"; then
    echo "  Changed file is missing Tigera copyright:" ${filename}
    copyright_check_failed=true
  fi
done

if $copyright_check_failed; then
  echo
  echo "Copyright statement should match"
  echo "  ${copyright_re}"
  echo "Example for new files:"
  echo "  Copyright (c) ${year} ${copyright_owner}. All rights reserved."
  echo "Example for updated files (use commas and year ranges):"
  echo "  Copyright (c) 2012,2015-${year} ${copyright_owner}. All rights reserved."
  echo "Change expected copyright owner by creating git-hooks/settings.sh."
fi

if $copyright_check_failed || $static_checks_failed; then
  echo
  exit 1
fi

# Remove the trap handler.
trap "" EXIT
