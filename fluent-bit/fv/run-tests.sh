#!/bin/bash
# Copyright (c) 2023 Tigera, Inc. All rights reserved.

set -ex

zone=$1
vm_name=$2-rocky8

assert_file_exists() {
    if ! gcloud compute ssh --zone="$zone" "user@$vm_name" -- test -f $1; then
        echo "file $1 doesn't exist"
        exit 1
    fi
}

assert_folder_exists() {
    if ! gcloud compute ssh --zone="$zone" "user@$vm_name" -- test -d $1; then
        echo "folder $1 doesn't exist"
        exit 1
    fi
}

echo "running Fluent Bit FV tests ..."

# check folders
assert_folder_exists /etc/calico/calico-fluent-bit

# check files
assert_file_exists /etc/calico/calico-fluent-bit/calico-fluent-bit.conf
assert_file_exists /etc/calico/calico-fluent-bit/parsers.conf
assert_file_exists /etc/calico/calico-fluent-bit/plugins.conf
assert_file_exists /usr/bin/calico-fluent-bit
assert_file_exists /usr/lib/systemd/system/calico-fluent-bit.service

echo "Fluent Bit FV tests completed successfully."
