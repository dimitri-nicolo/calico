#!/bin/bash

# Copyright (c) 2024 Tigera, Inc. All rights reserved.

# This function normalizes versions to be compatible with rpm build system
#
# For Calico components, the second version parameter is empty.
# * Official release: v3.20.1 -> 3.20.1
# * Early preview release: v3.20.0-1.1 -> 3.20.0~pre1.1^20240823gitc210c473
# * Main branch: 0.0.0^20240823gitc210c473
#
# For third-party components built by us, the vendor version is passed in as the second parameter.
# * Official release: v3.1.6 -> 3.1.6
# * Early preview release: v3.1.6 -> 3.1.6~pre1.1^20240823gitc210c473
# * Main branch: 3.1.6^20240823gitc210c473

repo_root=$1

if [[ -z "$repo_root" ]]; then
    echo "empty repository root"
    exit 1
fi

version=${2#v}
title=$(bin/yq -r .[0].title "$repo_root/calico/_data/versions.yml")

is_master=false
is_prerelease=false

if [[ "$title" == "master" ]]; then
    is_master=true
elif [[ "$title" == *-* ]]; then
    is_prerelease=true
fi

# For master, we always use 0.0.0 as the version string.
# For prerelease, we convert x.y.z-n.m to x.y.z~pren.m.
# For release, we use the version string as-is.
if [[ -z "$version" ]]; then
    if [[ "$is_master" = true ]]; then
        version="0.0.0"
    elif [[ "$is_prerelease" = true ]]; then
        title=${title#v}
        version="${title//-/"~pre"}"
    else
        version=${title#v}
    fi
fi

if [[ "$is_master" = true ]] || [[ "$is_prerelease" = true ]]; then
    echo "$version^$(date -u +'%Y%m%d')git$(git rev-parse --short=7 HEAD)"
else
    echo "$version"
fi
