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

assert_selinux_context() {
    if ! gcloud compute ssh --zone="$zone" "user@$vm_name" -- ls -dZ $1 | grep $2; then
        echo "entry $1 doesn't have expected selinux context $2"
        exit 1
    fi
}

echo "running SELinux FV tests ..."

# check folders
assert_folder_exists /var/lib/calico
assert_folder_exists /var/log/calico
assert_folder_exists /var/run/calico
assert_folder_exists /mnt/tigera

# check files
assert_file_exists /usr/share/selinux/devel/include/contrib/calico.if
assert_file_exists /usr/share/selinux/packages/calico.pp

# check selinux contexts
assert_selinux_context /var/lib/calico system_u:object_r:container_var_lib_t:s0
assert_selinux_context /var/log/calico system_u:object_r:container_log_t:s0
assert_selinux_context /var/run/calico system_u:object_r:container_var_run_t:s0
assert_selinux_context /mnt/tigera system_u:object_r:container_file_t:s0

echo "SELinux FV tests completed successfully."
