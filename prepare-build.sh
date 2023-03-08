#!/bin/bash

if ! command -v kubectl &> /dev/null
then
  # Install kubectl
  curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
  chmod +x kubectl
  mkdir -p ~/.local/bin/
  mv ./kubectl ~/.local/bin/kubectl
  export PATH=~/.local/bin:$PATH
fi

# Prepare creating an image
cd cloud-on-k8s
patch -p1 -i ../patches/0001-Update-go-restful-to-v3.10.1.patch
make go-generate
make generate-config-file
