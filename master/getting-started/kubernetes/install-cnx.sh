#!/bin/bash
#
# Script to install CNX on a kubeadm cluster. Requires the docker
# authentication json file. Note the script must be run on master node.

trap "exit 1" TERM
export TOP_PID=$$

# Override DOCS_VERSION to point to alternate CNX docs version, e.g.
#   DOCS_VERSION=v2.0 ./install-cnx.sh
#
# DOCS_VERSION is used to retrieve manifests, e.g.
#   ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
#      - resolves to -
#   http://0.0.0.0:4000/v2.0/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
DOCS_VERSION=${DOCS_VERSION:="v2.0"}

# Override DOCS_LOCATION to point to alternate CNX docs location, e.g.
#   DOCS_LOCATION=http://0.0.0.0:4000 ./install-cnx.sh
#
DOCS_LOCATION=${DOCS_LOCATION:="https://docs.tigera.io"}

# Override CREDENTIALS_FILE to point to alternate location
# of docker credentials json file, e.g.
#  CREDENTIALS_FILE=docker.json ./install-cnx.sh
#
CREDENTIALS_FILE=${CREDENTIALS_FILE:="config.json"}

# don't prompt for agreement
QUIET=${QUIET:=0}

# cleanup CNX installation
CLEANUP=0

# Convenience variables to cut down on tiresome typing
CNX_PULL_SECRET_FILENAME=${CNX_PULL_SECRET_FILENAME:="cnx-pull-secret.yml"}

#
# checkSettings()
#
checkSettings() {
  echo Settings:
  echo '  CREDENTIALS_FILE='${CREDENTIALS_FILE}
  echo '  DOCS_LOCATION='${DOCS_LOCATION}
  echo '  DOCS_VERSION='${DOCS_VERSION}

  echo
  echo -n About to "$1" CNX.
  if [ "$QUIET" -eq 0 ]; then
    read -n 1 -p " Proceed? (y/n): " answer

    echo
    if [ "$answer" != "y" ] && [ "$answer" != "Y" ]; then
      echo Exiting.
      exit 1
    fi
  fi

  echo Proceeding ...
}

#
# parseOptions() - parse command line options
#
parseOptions() {
  usage() {
    cat <<HELP_USAGE
Usage: $(basename "$0")
          [-c config.json]    # Docker authentication config file (from Tigera); default: "config.json"
          [-d docs_location]  # CNX documentation location; default: "https://docs.tigera.io"
          [-v version]        # CNX version; default: "v2.0"
          [-u]                # Remove CNX
          [-q]                # Quiet (don't prompt)
          [-h]                # Print usage
          [-x]                # Enable verbose mode

HELP_USAGE
    exit 1
  }

  local OPTIND
  while getopts "c:d:hqv:ux" opt; do
    case ${opt} in
      c )  CREDENTIALS_FILE=$OPTARG;;
      d )  DOCS_LOCATION=$OPTARG;;
      v )  DOCS_VERSION=$OPTARG;;
      x )  set -x;;
      q )  QUIET=1;;
      u )  CLEANUP=1;;
      h )  usage;;
      \? ) usage;;
    esac
  done
  shift $((OPTIND -1))
}

#
# fatalError() - log error to stderr, exit
#
fatalError() {
  >&2 echo "$@"
  kill -s TERM $TOP_PID   # we're likely running in a subshell, signal parent by PID
}

#
# countDownSecs() - count down by seconds
#
countDownSecs() {
  secs="$1"
  echo -n "$2" '- waiting for' "$secs" 'seconds: '
  while [ $secs -gt 0 ]; do
    echo -n .
    sleep 1
    : $((secs--))
  done
  echo
}

#
# runGetOutput() - run command, bail on errors
# echo output (both stdout and stderr) to stdout.
#
runGetOutput() {
  args=("$@")      # convert args to array
  output="$(2>&1 ${args[@]})"

  if [ $? -ne 0 ]; then
    fatalError Running command \'"$@"\' failed: \'${output}\'.
  fi
  echo "$output"
}

#
# run() - run command, bail on errors, discard output
#
run() {
  output=$(runGetOutput "$@")
}

#
# runGetOutputIgnoreErrors() - run command, ignore errors
# echo output (both stdout and stderr) to stdout.
#
runGetOutputIgnoreErrors() {
  args=("$@")      # convert args to array
  output="$(2>&1 ${args[@]})"
  echo "$output"
}

#
# runIgnoreErrors() - run command, ignore errors, discard output
#
runIgnoreErrors() {
  output=$(runGetOutputIgnoreErrors "$@")
}

#
# runAsRoot() - run sudo command, bail on errors, discard output
#
runAsRoot() {
  run sudo -E "$@"
}

#
# runAsRootIgnoreErrors() - run sudo command, ignore errors, discard output
#
runAsRootIgnoreErrors() {
  runIgnoreErrors sudo -E "$@"
}

#
# runAsRoot() - run sudo command, bail on errors, echo output
#
runAsRootGetOutput() {
  runGetOutput sudo -E "$@"
}

#
# programIsInstalled()
#
programIsInstalled() {
  local return_=0
  type "$1" >/dev/null 2>&1 || { local return_=1; }
  return $return_
}

#
# checkRequirementsInstalled() - check package dependencies
#
checkRequirementsInstalled() {
  programIsInstalled jq || fatalError Please install 'jq' and re-run "$(basename $0)".
  programIsInstalled sed || fatalError Please install 'sed' and re-run "$(basename "$0")".
  programIsInstalled base64 || fatalError Please install 'base64' and re-run "$(basename "$0")".

  # Check that kubectl can connect to the cluster
  run kubectl version

  # Check that gnu-sed is installed. To install gnu-sed on MacOS:
  #  'brew install gnu-sed --with-default-names'
  sed --version 2>&1 | grep -q GNU
  if [ $? -ne 0 ]; then
    fatalError Please install gnu-sed. On MacOS, \'brew install gnu-sed --with-default-names \'
  fi
}

#
# checkRequiredFilesPresent() - check kubernetes files are present
#
checkRequiredFilesPresent() {
  echo -n Verifying that we\'re on the Kubernetes master node ...' '
  runAsRoot ls -l /etc/kubernetes/manifests/kube-apiserver.yaml
  echo verified.
}

#
# getAuthToken() - read docker credentials file, return base64 encoded string.
#
getAuthToken() {
  # Make sure credentials file exists
  if [ ! -e "$CREDENTIALS_FILE" ]; then
    fatalError Credentials file \'"$CREDENTIALS_FILE"\' does not exist.
  fi

  # Ensure the credentials file contains valid json
  jq -e '.' "$CREDENTIALS_FILE" >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    fatalError Please verify \'"$CREDENTIALS_FILE"\' contains valid json credentials.
  fi

  # Base64-encode the auth credentials file, removing newlines and whitespace first.
  SECRET_TOKEN=$(cat "$CREDENTIALS_FILE" | tr -d '\n\r\t ' | base64 -w 0)
  if [ $? -ne 0 ]; then
    fatalError Unable to base64 encode \'"$CREDENTIALS_FILE"\'. Please verify contains valid json credentials.
  fi

  echo "${SECRET_TOKEN}"
}

#
# createImagePullSecretYaml()
#
createImagePullSecretYaml() {
  SECRET=$(getAuthToken)

  cat > ${CNX_PULL_SECRET_FILENAME} <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: cnx-pull-secret
  namespace: kube-system
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: ${SECRET}
EOF
}

#
# createImagePullSecret() {
#
createImagePullSecret() {
  createImagePullSecretYaml       # always recreate the pull secret
  run kubectl create -f "${CNX_PULL_SECRET_FILENAME}"
}

#
# deleteImagePullSecret() {
#
deleteImagePullSecret() {
  createImagePullSecretYaml       # always recreate the pull secret
  runIgnoreErrors kubectl delete -f "${CNX_PULL_SECRET_FILENAME}"
}

#
# setupBasicAuth()
#
setupBasicAuth() {
  # Create basic auth csv file
    cat > basic_auth.csv <<EOF
welc0me,jane,1
EOF

  runAsRoot mv basic_auth.csv /etc/kubernetes/pki/basic_auth.csv
  runAsRoot chown root /etc/kubernetes/pki/basic_auth.csv
  runAsRoot chmod 644 /etc/kubernetes/pki/basic_auth.csv

  # Append basic auth setting into kube-apiserver command line
  cat > sedcmd.txt <<EOF
/- kube-apiserver/a\    - --basic-auth-file=/etc/kubernetes/pki/basic_auth.csv
EOF

  # Insert basic-auth option into kube-apiserver manifest
  runAsRoot sed -i -f sedcmd.txt /etc/kubernetes/manifests/kube-apiserver.yaml
  run rm -f sedcmd.txt

  # Restart kubelet in order to make basic_auth settings take effect
  runAsRoot systemctl restart kubelet
  countDownSecs 20 "Setting up basic authentication in the kubernetes cluster"

  # Give user Jane cluster admin permissions
  runAsRoot kubectl create clusterrolebinding permissive-binding --clusterrole=cluster-admin --user=jane
}

#
# deleteBasicAuth()
#
deleteBasicAuth() {
  runAsRoot rm -f /etc/kubernetes/pki/basic_auth.csv
  runAsRootIgnoreErrors kubectl delete clusterrolebinding permissive-binding
  cat > sedcmd.txt <<EOF
/    - --basic-auth-file=\/etc\/kubernetes\/pki\/basic_auth.csv/d
EOF
  runAsRoot sed -i -f sedcmd.txt /etc/kubernetes/manifests/kube-apiserver.yaml
  run rm -f sedcmd.txt

  # Restart kubelet in order to make basic_auth settings take effect
  runAsRoot systemctl restart kubelet
  countDownSecs 20 "Restarting kubelet"
}

#
# applyCalicoManifest()
#
applyCalicoManifest() {
  run curl --compressed -O ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
  run kubectl apply -f calico.yaml
  countDownSecs 30 "Applying calico.yaml manifest"
}

#
# deleteCalicoManifest()
#
deleteCalicoManifest() {
  runIgnoreErrors kubectl delete -f ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
  countDownSecs 5 "Deleting calico.yaml manifest"
}

#
# removeMasterTaints()
#
removeMasterTaints() {
  runIgnoreErrors kubectl taint nodes --all node-role.kubernetes.io/master-
  countDownSecs 5 "Removing taints on master node"
}

#
# applyCNXManifest()
#
applyCNXManifest() {
  run curl --compressed -O ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml
  run kubectl apply -f cnx-etcd.yaml
  countDownSecs 30 "Applying cnx-etcd.yaml manifest"
}

#
# deleteCNXManifest()
#
deleteCNXManifest() {
  runIgnoreErrors kubectl delete -f ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml
  countDownSecs 5 "Deleting cnx-etcd.yaml manifest"
}

#
# applyCNXPolicyManifest()
#
applyCNXPolicyManifest() {
  run curl --compressed -O ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml
  run kubectl apply -f cnx-policy.yaml
  countDownSecs 10 "Applying cnx-policy.yaml manifest"
}

#
# deleteCNXPolicyManifest()
#
deleteCNXPolicyManifest() {
  runIgnoreErrors kubectl delete -f ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml
  countDownSecs 5 "Deleting cnx-policy.yaml manifest"
}

#
# isCRDRunning() - return 1 exit code if the CRD is running
#
isCRDRunning() {
  crd=$1

  if (kubectl get crd | grep -v NAME | grep -q $1); then
    return 0
  else
    return 1
  fi
}

#
# checkCRDs() - poll running CRDs until all
# operator CRDs are running, or timeout and fail.
#
checkCRDs() {
  alertCRD="alertmanagers.monitoring.coreos.com"
  promCRD="prometheuses.monitoring.coreos.com"
  svcCRD="servicemonitors.monitoring.coreos.com"

  echo -n "Waiting for Custom Resource Defintions to be created: "

  count=30
  while [[ $count -ne 0 ]]; do
    if (isCRDRunning $alertCRD) && (isCRDRunning $promCRD) && (isCRDRunning $svcCRD); then
        echo " all CRDs running!"
        return
    fi

    echo -n .
    ((count = count - 1))
    sleep 1
  done

  fatalError "Not all CRDs are running."
}

#
# applyOperatorManifest()
#
applyOperatorManifest() {
  run curl --compressed -O ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml
  run kubectl apply -f operator.yaml
  checkCRDs
}

#
# deleteOperatorManifest()
#
deleteOperatorManifest() {
  runIgnoreErrors kubectl delete -f ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml
  countDownSecs 5 "Deleting operator.yaml manifest"
}

#
# applyMonitorCalicoManifest()
#
applyMonitorCalicoManifest() {
  run curl --compressed -O ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml
  run kubectl apply -f monitor-calico.yaml
  countDownSecs 10 "Applying monitor-calico.yaml manifest"
}

#
# deleteMonitorCalicoManifest()
#
deleteMonitorCalicoManifest() {
  runIgnoreErrors kubectl delete -f ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml
  countDownSecs 5 "Deleting monitor-calico.yaml manifest"
}

#
# createCNXManagerSecret()
#
createCNXManagerSecret() {
  runAsRoot kubectl create secret generic cnx-manager-tls --from-file=cert=/etc/kubernetes/pki/apiserver.crt --from-file=key=/etc/kubernetes/pki/apiserver.key -n kube-system
  countDownSecs 1 "Creating cnx-manager-tls secret"
}

#
# deleteCNXManagerSecret()
#
deleteCNXManagerSecret() {
  runAsRootIgnoreErrors kubectl delete secret cnx-manager-tls -n kube-system
  countDownSecs 1 "Deleting cnx-manager-tls secret"
}

#
# installCNX() - install CNX
#
installCNX() {
  checkSettings install         # Verify settings are correct with user
  checkRequiredFilesPresent     # Validate kubernetes files are present
  checkRequirementsInstalled    # Validate that all required programs are installed

  createImagePullSecret         # Create the image pull secret
  setupBasicAuth                # Create 'jane/welc0me' account w/cluster admin privs

  applyCalicoManifest           # Apply calico.yaml
  removeMasterTaints            # Remove master taints
  createCNXManagerSecret        # Create cnx-manager-tls to enable manager/apiserver communication
  applyCNXManifest              # Apply cnx-etcd.yaml
  applyCNXPolicyManifest        # Apply cnx-policy.yaml
  applyOperatorManifest         # Apply operator.yaml
  applyMonitorCalicoManifest    # Apply monitor-calico.yaml

  echo CNX installation complete. Point your browser to https://127.0.0.1:30003, username=jane, password=welc0me
}

#
# cleanup() - remove CNX and related changes.
# Ignore errors.
#
cleanup() {
  checkSettings uninstall       # Verify settings are correct with user

  deleteMonitorCalicoManifest   # Delete monitor-calico.yaml
  deleteOperatorManifest        # Delete operator.yaml
  deleteCNXPolicyManifest       # Delete cnx-policy.yaml
  deleteCNXManifest             # Delete cnx-etcd.yaml
  deleteCalicoManifest          # Delete calico.yaml
  deleteCNXManagerSecret        # Delete TLS secret
  deleteImagePullSecret         # Delete pull secret
  deleteBasicAuth               # Remove basic auth updates, restart kubelet
}

#
# main()
#
parseOptions "$@"               # Set up variables based on args

if [ "$CLEANUP" -eq 1 ]; then
  cleanup                       # Remove CNX, cleanup related files
else
  installCNX                    # Install CNX
fi

