#!/bin/bash

BRANCH_NAME=${BRANCH_NAME:=master}
QUAY_AUTH_CONFIG=${QUAY_AUTH_CONFIG:=config.json}
LICENSE_KEY_FILE=${LICENSE_KEY_FILE:=license.yaml}
CNX_PW=${CNX_PW:=cnx_jane_pw}
SSH_KEY_FILE=${SSH_KEY_FILE:=master_ssh_key}

$(terraform output master_connect_command) "curl --compressed -O https://docs.tigera.io/${BRANCH_NAME}/getting-started/kubernetes/install-cnx.sh"
$(terraform output master_connect_command) "chmod +x ./install-cnx.sh"

# copy local files over
scp -i ${SSH_KEY_FILE} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ${QUAY_AUTH_CONFIG} "$(terraform output master_connect)":~/config.json
scp -i ${SSH_KEY_FILE} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ${LICENSE_KEY_FILE} "$(terraform output master_connect)":~/license.yaml
if [ -f "$CNX_PW" ]; then
  scp -i ${SSH_KEY_FILE} -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ${CNX_PW}           "$(terraform output master_connect)":~/cnx_jane_pw
fi

$(terraform output master_connect_command) "sudo ./install-cnx.sh -v ${BRANCH_NAME} -q -l license.yaml"

# uncomment if you desire a UI session over SSH port forwarding. 30003 is for the UI, 30601 is Kibana
echo "$(terraform output master_connect_command) -L localhost:30003:localhost:30003 -L localhost:30601:localhost:30601"
