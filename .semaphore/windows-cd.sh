#!/bin/bash
# Copyright (c) 2021 Tigera, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# This script makes and pushes images for Windows.
# It assumes Windows instances are provisioned with the terraform scripts
# located at github.com/tigera/process/testing/windows-instances.
set -ex

SSH_ARGS="-i ${PROCESS_REPO}/tf/master_ssh_key -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"

# Build eks log forwarder binaries on the Semaphore VM.
make eks-log-forwarder-startup

# Tar up repo so we can send it to the Windows instances where we will build the docker image.
tar --exclude='eks/.go-pkg-cache' --exclude='eks/.go-build-cache' -cvf fluentd-docker.tar .

GCR_SECRET=$(cat ~/secrets/banzai-google-service-account.json | base64 -w0)
GCR_LOGIN="[System.Text.Encoding]::ASCII.GetString([System.Convert]::FromBase64String('${GCR_SECRET}')) | docker login -u _json_key --password-stdin https://gcr.io"

# The set of commands we run on each Windows instance:
# - Untar the fluentd-docker repo tarball
# - Login to gcr.io
# - Tell git to ignore permission differences between Linux and Windows systems
# - Build and push images
CMD="powershell -Command \"7z x c:\\fluentd-docker.tar -oc:\\fluentd-docker ; cd c:\\fluentd-docker ; ${GCR_LOGIN} ; git config core.filemode false ; make cd BRANCH_NAME=${BRANCH_NAME} CONFIRM=${CONFIRM} PUSH_MANIFEST_IMAGES=""\""

# Loop over all host IPs. hosts.txt is created by the Windows host creation
# script and each host IP, one per line.
while read -r host; do
  LOG_FILE=~/log-${host}.txt

  # Copy over the fluentd-docker tarball.
  scp -vr ${SSH_ARGS} fluentd-docker.tar Administrator@${host}:/ </dev/null

  # Kick off cmd in the background, saving the individual logs.
  ssh ${SSH_ARGS} Administrator@${host} "${CMD}" </dev/null | tee ${LOG_FILE} &
done < ${PROCESS_REPO}/hosts.txt

# Wait until all image pushing is done.
wait

# Upload logs to artifact store.
while read -r host; do
  LOG_FILE=~/log-${host}.txt
  artifact push job --expire-in 2w ${LOG_FILE}
done < ${PROCESS_REPO}/hosts.txt
