#!/bin/bash
# Copyright (c) 2018 Tigera, Inc. All rights reserved.

set -e

if [[ $2 != *"-private"* ]]; then
    echo "Remote $1 not allowed: do not push closed source to a non-private repo" ;
    exit 1
fi
