#!/bin/bash

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
mkdir -p ~/.local/bin/
mv ./kubectl ~/.local/bin/kubectl
export PATH=~/.local/bin:$PATH

# Prepare creating an image
cd cloud-on-k8s
make go-generate
make generate-config-file
