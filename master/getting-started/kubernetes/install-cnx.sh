#!/bin/bash
#
# Script to install CNX on a kubeadm cluster. Requires the docker
# authentication json file. Note the script must be run on master node.

trap "exit 1" TERM
export TOP_PID=$$

# Override DOCS_VERSION to point to alternate CNX docs version, e.g.
#   DOCS_VERSION=v2.1 ./install-cnx.sh
#
# DOCS_VERSION is used to retrieve manifests, e.g.
#   ${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
#      - resolves to -
#   http://0.0.0.0:4000/v2.1/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
DOCS_VERSION=${DOCS_VERSION:="v2.1"}

# Override DOCS_LOCATION to point to alternate CNX docs location, e.g.
#   DOCS_LOCATION=http://0.0.0.0:4000 ./install-cnx.sh
#
DOCS_LOCATION=${DOCS_LOCATION:="https://docs.tigera.io"}

# Override CREDENTIALS_FILE to point to alternate location
# of docker credentials json file, e.g.
#  CREDENTIALS_FILE=docker.json ./install-cnx.sh
#
CREDENTIALS_FILE=${CREDENTIALS_FILE:="config.json"}

# Override DATASTORE to point to kdd or etcd (default), e.g.
#  DATASTORE="kdd" ./install-cnx.sh
#
DATASTORE=${DATASTORE:="etcd"}

# when set to 1, don't prompt for agreement to proceed
QUIET=${QUIET:=0}

# when set to 1, don't install prometheus components
SKIP_PROMETHEUS=${SKIP_PROMETHEUS:=0}

# cleanup CNX installation
CLEANUP=0

# Convenience variables to cut down on tiresome typing
CNX_PULL_SECRET_FILENAME=${CNX_PULL_SECRET_FILENAME:="cnx-pull-secret.yml"}

#
# promptToContinue()
#
promptToContinue() {
  if [ "$QUIET" -eq 0 ]; then
    read -n 1 -p "Proceed? (y/n): " answer

    echo
    if [ "$answer" != "y" ] && [ "$answer" != "Y" ]; then
      echo Exiting.
      exit 1
    fi
  fi
  echo Proceeding ...
}

#
# checkSettings()
#
checkSettings() {
  echo Settings:
  echo '  CREDENTIALS_FILE='${CREDENTIALS_FILE}
  echo '  DOCS_LOCATION='${DOCS_LOCATION}
  echo '  DOCS_VERSION='${DOCS_VERSION}
  echo '  DATASTORE='${DATASTORE}

  echo
  echo -n "About to "$1" CNX. "
  promptToContinue
}

#
# fatalError() - log error to stderr, exit
#
fatalError() {
  >&2 echo "$@"
  kill -s TERM $TOP_PID   # we're likely running in a subshell, signal parent by PID
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
          [-k datastore]      # Specify the datastore ("etcd"|"kdd"); default: "etcd"
          [-v version]        # CNX version; default: "v2.1"
          [-u]                # Uninstall CNX
          [-q]                # Quiet (don't prompt)
          [-h]                # Print usage
          [-x]                # Enable verbose mode

HELP_USAGE
    exit 1
  }

  local OPTIND
  while getopts "c:d:hk:pqv:ux" opt; do
    case ${opt} in
      c )  CREDENTIALS_FILE=$OPTARG;;
      d )  DOCS_LOCATION=$OPTARG;;
      k )  DATASTORE=$OPTARG;;
      v )  DOCS_VERSION=$OPTARG;;
      x )  set -x;;
      q )  QUIET=1;;
      p )  SKIP_PROMETHEUS=1;;
      u )  CLEANUP=1;;
      h )  usage;;
      \? ) usage;;
    esac
  done
  shift $((OPTIND -1))

  [ "$DATASTORE" == "etcd" ] || [ "$DATASTORE" == "kdd" ] || fatalError "Datastore \"$DATASTORE\" is not valid, must be either \"etcd\" or \"kdd\"."
}

#
# countDownSecs() - count down by seconds
#
countDownSecs() {
  secs="$1"
  echo -n "$2" '- waiting ' "$secs" 'seconds: '
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
# blockUntilSuccess()
#
blockUntilSuccess() {
  cmd="$1"
  secs="$2"
  echo -n "waiting up to $secs seconds for \"$cmd\" to complete: "
  until $cmd 2>/dev/null 1>/dev/null; do
    : $((secs--))
    echo -n .
    sleep 1
    if [ "$secs" -eq 0 ]; then
      fatalError "\"$cmd\" failed."
    fi
  done
  echo " \"$cmd\" succeeded."
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
# checkNetworkManager()
# Warn user if NetworkManager is enabled and there's
# no exception for "cali*" interfaces.
#
checkNetworkManager() {

  NMConfig="/etc/NetworkManager/NetworkManager.conf"

  echo -n "Checking status of Network Manager: "

  $(nmcli dev status 2>/dev/null 1>/dev/null)
  if [ $? -eq 0 ]; then
    echo "running."

    # Raise a warning if NM is running and the "cali" interfaces are
    # not marked as "unmanaged" by NM.
    if $( ! test -f "${NMConfig}" || ! grep -qs "cali" "${NMConfig}" ); then
      echo "  WARNING: We've detected that Network Manager is running and that"
      echo "  it is not configured to ignore \"cali\" interfaces. This will"
      echo "  interfere with Calico networking. Remove, disable, or configure"
      echo "  Network Manager to ignore \"cali\" interfaces. Refer to"
      echo "  \"Troubleshooting\" on https://docs.tigera.io for more information."
      promptToContinue
    fi
  else
    echo "not running."
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
# podStatus() - takes a selector as argument, e.g. "k8s-app=kube-dns"
# and returns the pod "ready" status as a bool string ("true"). Note that if
# the pod is in the "pending" state, there is no containerStatus yet, so
# podStatus() returns an empty string. Success means seeing the "true"
# substring, but failure can be "false" or an empty string.
#
function podStatus() {
  label="$1"
  status=$(kubectl get pods --selector="${label}" -o json --all-namespaces | jq -r '.items[] | .status.containerStatuses[]? | [.name, .image, .ready|tostring] |join(":")')
  echo "${status}"
}

#
# blockUntilPodIsReady() - takes a pod selector, a timeout in seconds
# and a "friendly" pod name as arguements. If the pod never stabilizes,
# bail. Otherwise return as soon as the pod is "ready."
#
function blockUntilPodIsReady() {
  label="$1"
  secs="$2"
  friendlyPodName="$3"

  echo -n "waiting up to ${secs} seconds for \"${friendlyPodName}\" to be ready: "
  until [[ $(podStatus "${label}") =~ "true" ]]; do
    if [ "$secs" -eq 0 ]; then
      fatalError "\"${friendlyPodName}\" never stabilized."
    fi

    : $((secs--))
    echo -n .
    sleep 1
  done

  echo " \"${friendlyPodName}\" is ready."
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
  blockUntilSuccess "kubectl get nodes" 60

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
  blockUntilSuccess "kubectl get nodes" 60
}

#
# downloadManifest()
# Do not overwrite existing copy of the manifest.
#
downloadManifest() {
  local manifest="$1"
  local filename=$(basename -- "$manifest")

  if [ ! -f "$filename" ]; then
    echo Downloading "$manifest"
    run curl --compressed -O "$manifest"
  else
    echo "\"$filename\" already exists, not downloading."
  fi
}

#
# downloadManifests()
#
downloadManifests() {
  downloadManifest "${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml"
  downloadManifest "${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml"
  downloadManifest "${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml"

  if [ "$DATASTORE" == "etcd" ]; then
    downloadManifest "${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml"
    downloadManifest "${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml"
  else
    downloadManifest "${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml"
    downloadManifest "${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-kdd.yaml"
    downloadManifest "${DOCS_LOCATION}/${DOCS_VERSION}/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml"
  fi
}

#
# applyCalicoManifest()
#
applyCalicoManifest() {
  # Apply rbac for kdd datastore
  if [ "$DATASTORE" == "kdd" ]; then
    run kubectl apply -f rbac-kdd.yaml
    countDownSecs 5 "Applying \"rbac-kdd.yaml\" manifest: "
  fi

  echo -n "Applying \"calico.yaml\" ("$DATASTORE") manifest: "
  run kubectl apply -f calico.yaml
  blockUntilPodIsReady "k8s-app=kube-dns" 180 "kube-dns"  # Block until kube-dns pod is running & ready
}

#
# deleteCalicoManifest()
#
deleteCalicoManifest() {
  runIgnoreErrors kubectl delete -f calico.yaml
  countDownSecs 5 "Deleting calico.yaml manifest"
}

#
# validateDatastore() - warn the user if they're switching
# between kdd/etcd datastore, but leaving the wrong manifest
# laying around (specifically calico.yaml). Use the existence
# of cnx-kdd.yaml|cnx-etcd.yaml as an indicator for which
# type of install/uninstall the user tried earlier.
#
function validateDatastore() {
  local operation="$1"           # install, uninstall
  local installedManifest=""     # set to "kdd" or "etcd" if there's a problem

  # Check if we're installing one datastore type but there's a manifest
  # for the "opposite" datastore type laying around the current directory.

  if [ "$DATASTORE" == "etcd" ]; then
    if [ -f "cnx-kdd.yaml" ]; then
      installedManifest="kdd"
    fi
  elif [ -f "cnx-etcd.yaml" ]; then
      installedManifest="etcd"
  fi

  if [ "$installedManifest" ]; then
    echo
    echo "Warning: the current $operation specifies \"$DATASTORE\", however \"cnx-$installedManifest.yaml\" exists"
    echo "         in the current directory. Either remove all the manifests from the"
    echo "         currect directory or use the \"-k $installedManifest\" flag and restart the $operation."
    promptToContinue
  fi
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
  echo -n "Applying \"cnx-${DATASTORE}.yaml\" manifest: "
  run kubectl apply -f cnx-"${DATASTORE}".yaml
  blockUntilPodIsReady "k8s-app=cnx-apiserver" 180 "cnx-apiserver"  # Block until cnx-apiserver pod is running & ready
  blockUntilPodIsReady "k8s-app=cnx-manager" 180 "cnx-manager"      # Block until cnx-manager pod is running & ready
  countDownSecs 10 "Waiting for cnx-apiserver to stabilize"         # Wait until cnx-apiserver completes registration w/kube-apiserver
}

#
# deleteCNXManifest()
#
deleteCNXManifest() {
  runIgnoreErrors kubectl delete -f cnx-"${DATASTORE}".yaml
  countDownSecs 30 "Deleting \"cnx-${DATASTORE}.yaml\" manifest"
}

#
# applyCNXPolicyManifest()
#
applyCNXPolicyManifest() {
  run kubectl apply -f cnx-policy.yaml
  countDownSecs 10 "Applying \"cnx-policy.yaml\" manifest"
}

#
# deleteCNXPolicyManifest()
#
deleteCNXPolicyManifest() {
  runIgnoreErrors kubectl delete -f cnx-policy.yaml
  countDownSecs 20 "Deleting \"cnx-policy.yaml\" manifest"
}

#
# isCRDRunning() - return 1 exit code if the CRD is running
#
isCRDRunning() {
  crd=$1

  if (kubectl get crd 2>/dev/null | grep -v NAME | grep -q $1); then
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

  echo -n "waiting for Custom Resource Defintions to be created: "

  count=30
  while [[ $count -ne 0 ]]; do
    if (isCRDRunning $alertCRD) && (isCRDRunning $promCRD) && (isCRDRunning $svcCRD); then
        echo "all CRDs running!"
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
  echo -n "Applying \"operator.yaml\" manifest: "
  run kubectl apply -f operator.yaml
  checkCRDs
}

#
# deleteOperatorManifest()
#
deleteOperatorManifest() {
  runIgnoreErrors kubectl delete -f operator.yaml
  countDownSecs 5 "Deleting \"operator.yaml\" manifest"
}

#
# applyMonitorCalicoManifest()
#
applyMonitorCalicoManifest() {
  echo -n "Applying \"monitor-calico.yaml\" manifest: "
  run kubectl apply -f monitor-calico.yaml
  blockUntilPodIsReady "app=prometheus" 180 "prometheus-calico-node"      # Block until prometheus-calico-nod pod is running & ready
}

#
# deleteMonitorCalicoManifest()
#
deleteMonitorCalicoManifest() {
  runIgnoreErrors kubectl delete -f monitor-calico.yaml
  countDownSecs 5 "Deleting \"monitor-calico.yaml\" manifest"
}

#
# createCNXManagerSecret()
#
createCNXManagerSecret() {
  runAsRoot kubectl create secret generic cnx-manager-tls --from-file=cert=/etc/kubernetes/pki/apiserver.crt --from-file=key=/etc/kubernetes/pki/apiserver.key -n kube-system
  countDownSecs 5 "Creating cnx-manager-tls secret"
}

#
# deleteCNXManagerSecret()
#
deleteCNXManagerSecret() {
  runAsRootIgnoreErrors kubectl delete secret cnx-manager-tls -n kube-system
  echo "Deleting cnx-manager-tls secret"
}

#
# installCNX() - install CNX
#
installCNX() {
  checkSettings install         # Verify settings are correct with user
  validateDatastore install     # Warn if there's etcd manifest, but we're doing kdd install (and vice versa)

  checkRequiredFilesPresent     # Validate kubernetes files are present
  checkRequirementsInstalled    # Validate that all required programs are installed
  checkNetworkManager           # Warn user if NetworkMgr is enabled w/o "cali" interace exception

  createImagePullSecret         # Create the image pull secret
  setupBasicAuth                # Create 'jane/welc0me' account w/cluster admin privs

  downloadManifests             # Download all manifests
  applyCalicoManifest           # Apply calico.yaml
  removeMasterTaints            # Remove master taints
  createCNXManagerSecret        # Create cnx-manager-tls to enable manager/apiserver communication
  applyCNXManifest              # Apply cnx-[etcd|kdd].yaml

  if [ "${SKIP_PROMETHEUS}" -eq 0 ]; then
    applyCNXPolicyManifest      # Apply cnx-policy.yaml
    applyOperatorManifest       # Apply operator.yaml
    applyMonitorCalicoManifest  # Apply monitor-calico.yaml
  fi

  echo CNX installation complete. Point your browser to https://127.0.0.1:30003, username=jane, password=welc0me
}

#
# uninstallCNX() - remove CNX and related changes.
# Ignore errors.
#
uninstallCNX() {
  checkSettings uninstall       # Verify settings are correct with user
  validateDatastore uninstall   # Warn if there's etcd manifest, but we're doing kdd uninstall (and vice versa)

  downloadManifests             # Download all manifests

  if [ "${SKIP_PROMETHEUS}" -eq 0 ]; then
    deleteMonitorCalicoManifest # Delete monitor-calico.yaml
    deleteOperatorManifest      # Delete operator.yaml
    deleteCNXPolicyManifest     # Delete cnx-policy.yaml
  fi

  deleteCNXManifest             # Delete cnx-[etcd|kdd].yaml
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
  uninstallCNX                  # Remove CNX, cleanup related files
else
  installCNX                    # Install CNX
fi

