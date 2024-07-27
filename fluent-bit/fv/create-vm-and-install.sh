#!/bin/bash
# Copyright (c) 2024 Tigera, Inc. All rights reserved.

set -ex

zone=$1
vm_name=$2-rocky8
project=${GCP_PROJECT:-unique-caldron-775}
gcp_secret_key=${GCP_SECRET_KEY:-$HOME/secrets/secret.google-service-account-key.json}

gcloud config set project "$project"
gcloud auth activate-service-account --key-file="$gcp_secret_key"

create_vm() {
    if gcloud --quiet compute instances create "$vm_name" \
        --zone="$zone" \
        --machine-type=e2-medium \
        --image=rocky-linux-8-optimized-gcp-v20230411 \
        --image-project=rocky-linux-cloud \
        --boot-disk-size=20GB \
        --boot-disk-type=pd-standard; then
        for ssh_try in $(seq 1 20); do
            echo "trying to ssh in: attempt $ssh_try ..."
            if gcloud --quiet compute ssh --zone="$zone" "user@$vm_name" -- cat /etc/os-release; then
                echo "vm $vm_name is created successfully."
                return 0
            else
                sleep 3
            fi
        done
    fi

    return 1
}

delete_vm() {
    echo "deleting $vm_name ..."
    gcloud --quiet compute instances delete "$vm_name" --zone="$zone"
}

copy_and_install() {
    local package=calico-fluent-bit-3.1.4-1.el8.x86_64.rpm
    echo "scp $package to $vm_name ..."
    if gcloud --quiet compute scp --zone="$zone" "package/rhel8/RPMS/x86_64/$package" "user@$vm_name:/tmp/$package"; then
        echo "install $package to $vm_name ..."
        if gcloud --quiet compute ssh --zone="$zone" "user@$vm_name" -- sudo dnf install -y /tmp/$package; then
            echo "$package is installed on $vm_name successfully."
            return 0
        fi
    fi

    return 1
}

for attempt in $(seq 1 3); do
    echo "creating $vm_name, attempt $attempt ..."
    if create_vm; then
        if copy_and_install; then
            echo "calico fluent-bit package is installed successfully."
            exit 0
        fi

        echo "failed to copy and install calico fluent-bit package."
    else
        echo "failed to create or ssh into $vm_name."
    fi

    delete_vm
done

echo "out of retries."
exit 1
