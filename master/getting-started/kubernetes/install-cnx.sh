---
layout: null
---
#!/usr/bin/env bash
#
# Script to install Tigera Secure EE on a kubeadm cluster. Requires the docker
# authentication json file. Note the script must be run on master node.

trap "exit 1" TERM
export TOP_PID=$$

# Override VERSION to point to alternate Tigera Secure EE docs version, e.g.
#   VERSION=v2.1 ./install-cnx.sh
#
# VERSION is used to retrieve manifests, e.g.
#   ${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/calico.yaml
#      - resolves to -
#   https://docs.tigera.io/v2.1/getting-started/kubernetes/installation/hosted/calico.yaml
VERSION=${VERSION:="{{page.version}}"}

# Override DOCS_LOCATION to point to alternate Tigera Secure EE docs location, e.g.
#   DOCS_LOCATION="https://docs.tigera.io" ./install-cnx.sh
#
DOCS_LOCATION=${DOCS_LOCATION:="https://docs.tigera.io"}

# Override CREDENTIALS_FILE to point to alternate location
# of docker credentials json file, e.g.
#  CREDENTIALS_FILE=docker.json ./install-cnx.sh
#
CREDENTIALS_FILE=${CREDENTIALS_FILE:="config.json"}

# Override DATASTORE to point to "kubernetes" or "etcdv3" (default), e.g.
#  DATASTORE="kubernetes" ./install-cnx.sh
#
DATASTORE=${DATASTORE:="etcdv3"}

# Specify a license file, e.g.
#   LICENSE_FILE="./my-great-license.yaml" ./install-cnx.sh
#
LICENSE_FILE=${LICENSE_FILE:="license.yaml"}

# Specify the type of elasticsearch storage to use: "none" or "local".
#   ELASTIC_STORAGE="none" ./install-cnx.sh
ELASTIC_STORAGE=${ELASTIC_STORAGE:="local"}

# Specify an external etcd endpoint(s), e.g.
#   ETCD_ENDPOINTS=https://192.168.0.1:2379 ./install-cnx.sh
# Default is to pull the endpoint(s) from calico.yaml
ETCD_ENDPOINTS=${ETCD_ENDPOINTS:=""}

# when set to 1, don't prompt for agreement to proceed
QUIET=${QUIET:=0}

# when set to 1, download the manifests, then quit
DOWNLOAD_MANIFESTS_ONLY=${DOWNLOAD_MANIFESTS_ONLY:=0}

# Deployment type of the tigera installation.  One of basic, typha or federation.
# A deployment type of "typha" is only valid for kubernetes datastore.
DEPLOYMENT_TYPE=${DEPLOYMENT_TYPE:="basic"}

# Can be "calico", "aws", or "other"
# when set to calico, install manifest with calico networking
# when set to "aws", install manifest with no CNI plugin or configuration
# when set to "other", install manifest with Calico CNI that uses host-local IPAM
# Only used for the kubernetes datastore.
NETWORKING=${NETWORKING:="calico"}

# cleanup Tigera Secure EE installation
CLEANUP=0

# Convenience variables to cut down on tiresome typing
CNX_PULL_SECRET_FILENAME=${CNX_PULL_SECRET_FILENAME:="cnx-pull-secret.yml"}

# Registry to pull calicoctl and calicoq from
CALICO_REGISTRY=${CALICO_REGISTRY:="quay.io/tigera"}

# Calicoctl and calicoq binary install directory
CALICO_UTILS_INSTALL_DIR=${CALICO_UTILS_INSTALL_DIR:="/usr/local/bin"}

# Location of kube-apiserver manifest
KUBE_APISERVER_MANIFEST=${KUBE_APISERVER_MANIFEST:="/etc/kubernetes/manifests/kube-apiserver.yaml"}

# Location of kube-controller-manager manifest
KUBE_CONTROLLER_MANIFEST=${KUBE_CONTROLLER_MANIFEST:="/etc/kubernetes/manifests/kube-controller-manager.yaml"}

# Kubernetes installer type - "KOPS" or "KUBEADM" or "ACS-ENGINE"
INSTALL_TYPE=${INSTALL_TYPE:="KUBEADM"}

INSTALL_AWS_SG=${INSTALL_AWS_SG:=false}

SKIP_EE_INSTALLATION=${SKIP_EE_INSTALLATION:=false}


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
# checkRegistry()  default is quay.io
#
checkRegistry() {
 local IMAGE_REPO="{{page.registry}}"
 if [[ "$IMAGE_REPO" == "gcr.io/"* ]]; then
   CALICO_REGISTRY="gcr.io"
 fi
}

#
# checkSettings()
#
checkSettings() {
  checkRegistry

  echo Settings:
  echo '  CREDENTIALS_FILE='${CREDENTIALS_FILE}
  echo '  DOCS_LOCATION='${DOCS_LOCATION}
  echo '  VERSION='${VERSION}
  echo '  REGISTRY='${CALICO_REGISTRY}
  echo '  DATASTORE='${DATASTORE}
  echo '  DEPLOYMENT_TYPE='${DEPLOYMENT_TYPE}
  echo '  NETWORKING='${NETWORKING}
  echo '  LICENSE_FILE='${LICENSE_FILE}
  echo '  INSTALL_TYPE='${INSTALL_TYPE}
  echo '  ELASTIC_STORAGE='${ELASTIC_STORAGE}
  [ "$DOWNLOAD_MANIFESTS_ONLY" == 1 ] && echo '  DOWNLOAD_MANIFESTS_ONLY='${DOWNLOAD_MANIFESTS_ONLY}
  [ "$ETCD_ENDPOINTS" ] && echo '  ETCD_ENDPOINTS='${ETCD_ENDPOINTS}

  echo
  echo -n "About to "$1" Tigera Secure EE. "
  promptToContinue
}

#
# fatalError() - log error to stderr, exit
#
fatalError() {
  >&2 echo "Fatal Error: $@"
  if [ "$CLEANUP" -eq 0 ]; then  # If this is a fresh install (not an uninstall), tell user how to retry
    >&2 echo "In order to retry installation, uninstall Tigera Secure EE first (i.e. re-run with \"-u\" flag)."
  fi
  kill -s TERM $TOP_PID   # we're likely running in a subshell, signal parent by PID
}

#
# parseOptions() - parse command line options
#
parseOptions() {
  usage() {
    cat <<HELP_USAGE
Usage: $(basename "$0")
          [-a cluster_name]    # Install AWS Security Group Integration using Cluster Name
          [-l license.yaml]    # Specify the path to the Tigera Secure EE license file; default "license.yaml". Note license is required.
          [-c config.json]     # Docker authentication config file (from Tigera); default: "config.json"
          [-d docs_location]   # Tigera Secure EE documentation location; default: "https://docs.tigera.io"
          [-e etcd_endpoints]  # etcd endpoint address, e.g. ("http://10.0.0.1:2379"); default: take from manifest automatically
          [-k datastore]       # Specify the datastore ("etcdv3"|"kubernetes"); default: "etcdv3"
          [-n networking]      # Specify the networking ("calico"|"other"|"aws"); default "calico"
          [-s elastic_storage] # Specify the elasticsearch storage to use ("none"|"local"); default: "local"
          [-t deployment_type] # Specify the deployment type ("basic"|"typha"|"federation"); default "basic"
          [-v version]         # Tigera Secure EE version; default: "v2.1"
          [-u]                 # Uninstall Tigera Secure EE
          [-q]                 # Quiet (don't prompt)
          [-m]                 # Download manifests (then quit)
          [-h]                 # Print usage
          [-x]                 # Enable verbose mode
          [-0 vpc_id]          # VPC id for AWS Security Group Integration
          [-1 control_sg]      # Control plane SG for AWS Security Group Integration
          [-2 node_sg]         # Node SG for AWS Security Group Integration

HELP_USAGE
    exit 1
  }

  local OPTIND
  while getopts "a:c:d:e:hk:l:mn:pqs:t:v:ux0:1:2:" opt; do
    case ${opt} in
      a )  INSTALL_AWS_SG=true; CLUSTER_NAME=$OPTARG;;
      c )  CREDENTIALS_FILE=$OPTARG;;
      d )  DOCS_LOCATION=$OPTARG;;
      e )  ETCD_ENDPOINTS=$OPTARG;;
      k )  DATASTORE=$OPTARG;;
      l )  LICENSE_FILE=$OPTARG;;
      n )  NETWORKING=$OPTARG;;
      s )  ELASTIC_STORAGE=$OPTARG;;
      t )  DEPLOYMENT_TYPE=$OPTARG;;
      v )  VERSION=$OPTARG;;
      x )  set -x;;
      q )  QUIET=1;;
      m )  DOWNLOAD_MANIFESTS_ONLY=1;;
      u )  CLEANUP=1;;
      h )  usage;;
      \? ) usage;;
      0 )  VPC_ID=$OPTARG;;
      1 )  CONTROL_PLANE_SG=$OPTARG;;
      2 )  K8S_NODE_SGS=$OPTARG;;
    esac
  done
  shift $((OPTIND -1))
}

#
# validateSettings() - ensure all required files and directories are present,
# and that global variables have reasonable values.
#
validateSettings() {
  # Validate $DATASTORE is either "kubernetes" or "etcdv3"
  [ "$DATASTORE" == "etcdv3" ] || [ "$DATASTORE" == "kubernetes" ] || fatalError "Datastore \"$DATASTORE\" is not valid, must be either \"etcdv3\" or \"kubernetes\"."

  # Validate $INSTALL_TYPE is either "KOPS" or "KUBEADM" or "ACS-ENGINE"
  [ "$INSTALL_TYPE" == "KOPS" ] || [ "$INSTALL_TYPE" == "KUBEADM" ] || [ "$INSTALL_TYPE" == "ACS-ENGINE" ] || fatalError "Installation type \"$INSTALL_TYPE\" is not valid, must be either \"KOPS\" or \"KUBEADM\" or \"ACS-ENGINE\"."

  # Validate $ELASTIC_STORAGE is either "none" or "local"
  [ "$ELASTIC_STORAGE" == "local" ] || [ "$ELASTIC_STORAGE" == "none" ] || fatalError "Elasticsearch storage \"$ELASTIC_STORAGE\" is not valid, must be either \"local\" or \"none\"."

  # Validate $DEPLOYMENT_TYPE is either "basic", "typha", or "federation"
  [ "$DEPLOYMENT_TYPE" == "basic" ] || [ "$DEPLOYMENT_TYPE" == "typha" ] || [ "$DEPLOYMENT_TYPE" == "federation" ] || fatalError "Deployment type \"$DEPLOYMENT_TYPE\" is not valid, must be either \"basic\" or \"typha\" or \"federation\"."

  # Validate $DEPLOYMENT_TYPE is not "typha" if datastore is "etcdv3"
  [ "$DEPLOYMENT_TYPE" == "typha" ] && [ "$DATASTORE" == "etcdv3" ] && fatalError "Deployment type \"$DEPLOYMENT_TYPE\" is not valid for Datastore \"$DATASTORE\"."

  # Validate $NETWORKING is either "calico", "other", or "aws"
  [ "$NETWORKING" == "calico" ] || [ "$NETWORKING" == "other" ] || [ "$NETWORKING" == "aws" ] || fatalError "Networking type \"$NETWORKING\" is not valid, must be either \"calico\" or \"other\" or \"aws\"."

  # If we're installing, confirm user specified a readable license file
  if [ "$CLEANUP" -eq 0 ]; then
    [ -z "$LICENSE_FILE" ] && fatalError "Must specify the location of a Tigera Secure EE license file, e.g. '-l license.yaml'"

    # Confirm license file is readable
    [ ! -r "$LICENSE_FILE" ] && fatalError "Couldn't locate license file: $LICENSE_FILE"
  fi

  # Confirm kube-apiserver manifest exists
  [ ! -f "$KUBE_APISERVER_MANIFEST" ] && fatalError "Couldn't locate kube-apiserver manifest: $KUBE_APISERVER_MANIFEST"

  # If it's set, validate that ETCD_ENDPOINTS is of the form "http[s]://ipv4:port"
  if [ ! -z "$ETCD_ENDPOINTS" ]; then
    [[ "$ETCD_ENDPOINTS" =~ ^http(s)?://.*:[0-9]+$ ]] || fatalError "Invalid ETCD_ENDPOINTS=\"${ETCD_ENDPOINTS}\", expect \"http(s)://ipv4:port\""
  fi

  if "$INSTALL_AWS_SG" ; then
    checkAwsIntegration
  fi
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
# determineInstallerType()
# Determine whether this cluster was installed by one of the three supported
# kubernetes installers - "KOPS" or "KUBEADM" or "ACS-ENGINE"
# Set INSTALL_TYPE to "KOPS" or "ACS-ENGINE" or "KUBEADM"
#
determineInstallerType() {
  # Assume Kubeadm by default. Check for directories and files used by Kops and override if necessary
  if [[ -d "/srv/kubernetes/" && -e "/etc/kubernetes/manifests/kube-apiserver.manifest" ]]; then
    INSTALL_TYPE="KOPS"
    KUBE_APISERVER_MANIFEST="/etc/kubernetes/manifests/kube-apiserver.manifest"

    # give kops clusters a working default etcd endpoint, if not already set
    [ -z "$ETCD_ENDPOINTS" ] && ETCD_ENDPOINTS="http://100.64.1.5:6666"
  fi

  # Assume Kubeadm by default. Check for directories and files used by acs-engine and override if necessary
  if [[ -d "/etc/kubernetes/" && -e "/etc/kubernetes/azure.json" ]]; then
    INSTALL_TYPE="ACS-ENGINE"
    DATASTORE="kubernetes"
    NETWORKING="other"
  fi
}

#
# jq container replacement
#
function jq-container() {
  sudo docker run -i quay.io/bcreane/jq:latest "$@"
}

#
# checkRequirementsInstalled() - check package dependencies
#
checkRequirementsInstalled() {
  programIsInstalled sed || fatalError "Please install \"sed\" and re-run $(basename "$0")".
  programIsInstalled base64 || fatalError "Please install \"base64\" and re-run $(basename "$0")".
  programIsInstalled ip || fatalError "Please install \"ip\" and re-run $(basename "$0")".

  # Check that kubectl can connect to the cluster
  run kubectl version

  # Check that gnu-sed is installed.
  sed --version 2>&1 | grep -q GNU
  if [ $? -ne 0 ]; then
    fatalError Please install gnu-sed. On MacOS, \'brew install gnu-sed --with-default-names \'
  fi

  if "$INSTALL_AWS_SG" ; then
    # This verifies AWS is installed and that credentials are configured
    run aws sts get-caller-identity
  fi
}

#
# checkNetworkManager()
# Warn user if NetworkManager is enabled and there's
# no exception for "cali*" interfaces.
#
checkNetworkManager() {

  NMConfig="/etc/NetworkManager/NetworkManager.conf"

  echo -n "Checking status of NetworkManager: "

  $(nmcli dev status 2>/dev/null 1>/dev/null)
  if [ $? -eq 0 ]; then
    echo "running."

    # Raise a warning that NM is running. It's possible the user has
    # configured exceptions for "cali*" and "tunl*" interfaces which
    # should be sufficient for NM and Tigera Secure EE to interoperate.
    echo
    echo "  WARNING: We've detected that NetworkManager is running. Unless you've"
    echo "  configured exceptions for Tigera Secure EE network interfaces, NetworkManager"
    echo "  will interfere with Tigera Secure EE networking. Remove, disable, or configure"
    echo "  NetworkingManager to ignore Tigera Secure EE network interfaces. Refer to"
    echo "  \"Troubleshooting\" on https://docs.tigera.io for more information."
    promptToContinue
  else
    echo "not running."
  fi
}

#
# checkNumberOfCores() - ensure master node has at least
# $MIN_REQUIRED_CORES. Note, just issue a warning rather
# than blocking the installation to avoid false negatives.
#
checkNumberOfCores() {
  local MIN_REQUIRED_CORES=2
  cores=$(getconf _NPROCESSORS_ONLN 2>/dev/null)
  if [ $? -eq 0 ]; then
    if [ "$cores" \< "$MIN_REQUIRED_CORES" ]; then
      echo "Warning: Tigera Secure EE requires ${MIN_REQUIRED_CORES} processor cores, however it looks as though your machine has only ${cores} core(s)."
    else
      echo "Verified your machine has the required minimum number (${MIN_REQUIRED_CORES}) of processor cores : ${cores} present."
    fi
  else
    echo "Unable to retrieve number of processor cores - minimum required is ${MIN_REQUIRED_CORES}."
  fi
}

#
# checkRequiredFilesPresent() - check kubernetes files are present
#
checkRequiredFilesPresent() {
  echo -n Verifying that we\'re on the Kubernetes master node ...' '
  runAsRoot ls -l "$KUBE_APISERVER_MANIFEST"
  echo verified.
}

#
# podStatus() - takes a selector as argument, e.g. "k8s-app=kube-dns"
# and returns a line for each container in the selected pods with
# the container "ready" status as a bool string ("true" or "false").
#
function podStatus() {
  label="$1"
  pod_info=$(kubectl get pods --selector="${label}" -o json --all-namespaces 2> /dev/null)
  echo $pod_info | jq-container -r '.items[] | "\(.metadata.name) \(.spec.containers[] | .name)"' \
  | while read pod_name container_name; do
    status=$(echo $pod_info | jq-container -r ".items[] | select(.metadata.name == \"$pod_name\") | .status | if .containerStatuses then .containerStatuses[] | select(.name == \"$container_name\") | if .ready then \"\(.ready|tostring)\" else \"false\" end else \"false\" end")
    if [ -z "$status" ]; then
      status="false"
    fi
    echo $pod_name:$container_name:$status
  done
  # If there are no pods then there is not one running
  if [ -z "$(kubectl get pods --selector="${label}" --all-namespaces 2> /dev/null)" ]; then
    echo "${label}:false"
  fi
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
  while [[ $(podStatus "${label}") =~ "false" ]]; do
    if [ "$secs" -eq 0 ]; then
      fatalError "\"${friendlyPodName}\" never stabilized."
    fi

    : $((secs--))
    echo -n .
    sleep 1
  done

  echo " \"${friendlyPodName}\" is ready."
  podStatus "${label}"
  kubectl get pods --selector="${label}" -o wide --all-namespaces
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
  cat "$CREDENTIALS_FILE" | jq-container --exit-status >/dev/null 2>&1
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

  # Need the pull secret in both kube-system and calico-monitoring
  cat > ${CNX_PULL_SECRET_FILENAME} <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: cnx-pull-secret
  namespace: kube-system
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: ${SECRET}
---
apiVersion: v1
kind: Secret
metadata:
  name: cnx-pull-secret
  namespace: calico-monitoring
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: ${SECRET}
EOF
}

#
# createImagePullSecret() {
#
createImagePullSecret() {
  kubectl create namespace calico-monitoring

  if [ $CALICO_REGISTRY == "gcr.io" ]; then
    kubectl create secret docker-registry cnx-pull-secret --namespace=kube-system --docker-server=https://gcr.io --docker-username=_json_key --docker-email=user@example.com --docker-password="$(cat $CREDENTIALS_FILE)"
    kubectl create secret docker-registry cnx-pull-secret --namespace=calico-monitoring --docker-server=https://gcr.io --docker-username=_json_key --docker-email=user@example.com --docker-password="$(cat $CREDENTIALS_FILE)"

    return
  fi

  createImagePullSecretYaml       # always recreate the pull secret
  run kubectl create -f "${CNX_PULL_SECRET_FILENAME}"
}

#
# deleteImagePullSecret() {
#
deleteImagePullSecret() {
  if [ $CALICO_REGISTRY == "gcr.io" ]; then
    runIgnoreErrors kubectl delete secret cnx-pull-secret --namespace=kube-system
    runIgnoreErrors kubectl delete secret cnx-pull-secret --namespace=calico-monitoring
    runIgnoreErrors kubectl delete namespace calico-monitoring

    return
  fi

  createImagePullSecretYaml       # always recreate the pull secret
  runIgnoreErrors kubectl delete -f "${CNX_PULL_SECRET_FILENAME}"
  runIgnoreErrors kubectl delete namespace calico-monitoring
}

#
# createPassword() - create a simple password and save it to cnx_jane_pw
#
createPassword() {
  if [ ! -f cnx_jane_pw ]; then
    cat /dev/urandom | tr -dc A-Za-z0-9 | head -c16 > cnx_jane_pw
  fi
}

#
# setupBasicAuthKubeadm() - specialized function for Kubeadm-based kubernetes
# clusters
#
setupBasicAuthKubeadm() {
  createPassword

  # Create basic auth csv file
    cat > basic_auth.csv <<EOF
`cat cnx_jane_pw`,jane,1
EOF

  runAsRoot mv basic_auth.csv /etc/kubernetes/pki/basic_auth.csv
  runAsRoot chown root /etc/kubernetes/pki/basic_auth.csv
  runAsRoot chmod 600 /etc/kubernetes/pki/basic_auth.csv

  # Append basic auth setting into kube-apiserver command line
  cat > sedcmd.txt <<EOF
/- kube-apiserver/a\    - --basic-auth-file=/etc/kubernetes/pki/basic_auth.csv
EOF

  # Insert basic-auth option into kube-apiserver manifest
  runAsRoot sed -i -f sedcmd.txt "$KUBE_APISERVER_MANIFEST"
  run rm -f sedcmd.txt

  # Restart kubelet in order to make basic_auth settings take effect
  runAsRoot systemctl restart kubelet
  blockUntilSuccess "kubectl get nodes" 60

  # Give user "Jane" cluster admin permissions
  runAsRoot kubectl create clusterrolebinding permissive-binding --clusterrole=cluster-admin --user=jane
}

#
# setupBasicAuthAcsEngine() - specialized function for acs-engine-based kubernetes
# clusters
#
setupBasicAuthAcsEngine() {
  createPassword

  # Create basic auth csv file
    cat > basic_auth.csv <<EOF
`cat cnx_jane_pw`,jane,1
EOF

  runAsRoot mkdir -p /etc/kubernetes/pki
  runAsRoot mv basic_auth.csv /etc/kubernetes/pki/basic_auth.csv
  runAsRoot chown root /etc/kubernetes/pki/basic_auth.csv
  runAsRoot chmod 600 /etc/kubernetes/pki/basic_auth.csv

  # Append basic auth setting into kube-apiserver command line
  cat > sedcmd.txt <<EOF
s?\"--cloud-provider=azure\"?\"--cloud-provider=azure\", \"--basic-auth-file=/etc/kubernetes/pki/basic_auth.csv\"?g
EOF

  # Insert basic-auth option into kube-apiserver manifest
  runAsRoot sed -i -f sedcmd.txt "$KUBE_APISERVER_MANIFEST"
  run rm -f sedcmd.txt

  # Restart kubelet in order to make basic_auth settings take effect
  runAsRoot systemctl restart kubelet
  blockUntilSuccess "kubectl get nodes" 60

  # Give user "Jane" cluster admin permissions
  runAsRoot kubectl create clusterrolebinding permissive-binding --clusterrole=cluster-admin --user=jane
}

#
# setupBasicAuthKops() - currently a no-op
# because Kops preprovisions an "admin" user and the
# /srv/kubernetes/basic_auth.csv is not modifiable.
#
setupBasicAuthKops() {
  echo "Using \"admin\" user provisioned by Kops."
}

#
# setupBasicAuth()
#
setupBasicAuth() {

  if [ "$INSTALL_TYPE" == "KUBEADM" ]; then
    setupBasicAuthKubeadm
  elif [ "$INSTALL_TYPE" == "KOPS" ]; then
    setupBasicAuthKops
  elif [ "$INSTALL_TYPE" == "ACS-ENGINE" ]; then
    setupBasicAuthAcsEngine
  fi
}

#
# deleteBasicAuth() - clean up basic auth file and
# delete cluster admin role for user=jane.
#
deleteBasicAuth() {
  if [ "$INSTALL_TYPE" == "KOPS" ]; then
    return
  fi

  runAsRoot rm -f /etc/kubernetes/pki/basic_auth.csv

  if [ "$INSTALL_TYPE" == "KUBEADM" ]; then
    cat > sedcmd.txt <<EOF
/    - --basic-auth-file=\/etc\/kubernetes\/pki\/basic_auth.csv/d
EOF
  elif [ "$INSTALL_TYPE" == "ACS-ENGINE" ]; then
    cat > sedcmd.txt <<EOF
s? \"--basic-auth-file=/etc/kubernetes/pki/basic_auth.csv\",??g
EOF
  fi

  runAsRoot sed -i -f sedcmd.txt "$KUBE_APISERVER_MANIFEST"
  run rm -f sedcmd.txt

  # Restart kubelet in order to make basic_auth settings take effect
  runAsRoot systemctl restart kubelet
  blockUntilSuccess "kubectl get nodes" 60

  # Cleanup permissive-binding role
  runAsRootIgnoreErrors kubectl delete clusterrolebinding permissive-binding
}

#
# deleteKubeDnsPod() - delete kube-dns pod so that subsequent installs
# block on DNS pod coming up.
#
deleteKubeDnsPod() {
  echo -n "Restarting kube-dns ... "
  runAsRoot "kubectl -n kube-system delete pod --selector=k8s-app=kube-dns"
  echo "done."
}

#
# dockerLogin() - login to Tigera registry
#
dockerLogin() {
  if [ ! -f "$CREDENTIALS_FILE" ]; then
    fatalError "$CREDENTIALS_FILE" does not exist.
  fi

  if [ $CALICO_REGISTRY == "gcr.io" ]; then
    username="_json_key"
    token=$(cat "${CREDENTIALS_FILE}")
  else
    dockerCredentials=$(cat "${CREDENTIALS_FILE}" | jq-container --raw-output '.auths[].auth' | base64 -d)
    username=$(echo -n $dockerCredentials | awk -F ":" '{print $1}')
    token=$(echo -n $dockerCredentials | awk -F ":" '{print $2}')
  fi

  echo -n "Logging in to ${CALICO_REGISTRY} ... "
  sudo -E docker login --username=${username} --password="${token}" ${CALICO_REGISTRY}
  echo "done."
}

#
# setupEtcdEndpoints() - if DATASTORE is etcdv3, setup
# ${ETCD_ENDPOINTS}. Optionally update calico.yaml if
# the user specified a different etcd endpoint.
#
setupEtcdEndpoints() {

  if [ "$DATASTORE" == "etcdv3" ]; then

    # Sanity checking - we need calico.yaml
    if [ ! -f calico.yaml ]; then
      fatalError "Error: did not find \"calico.yaml\" manifest."
    fi

    # $ETCD_ENDPOINTS is unspecified; user wants to use the default etcd endpoint
    # associated with self-hosted etcd pod
    if [ -z "${ETCD_ENDPOINTS}" ]; then

      export SELF_HOSTED_ETCD_ADDR="10.96.232.136:6666"
      export ETCD_ENDPOINTS="http://${SELF_HOSTED_ETCD_ADDR}"

      cat > sedcmd.txt <<EOF
s?<ETCD_IP>:<ETCD_PORT>?${SELF_HOSTED_ETCD_ADDR}?g
EOF

     # update calico.yaml
     sed -i -f sedcmd.txt calico.yaml
     rm -f sedcmd.txt

    else

      # User specified an ETCD_ENDPOINT setting - overwrite default etcd_endpoints in calico.yaml.
      # This sed cmd matches <etcd_endpoints: "http*> up to the end of the line and replaces it with:
      #                      <etcd_endpoints: "$ETCD_ENDPOINTS">
      sed -Ei "s|etcd_endpoints: \"http.*|etcd_endpoints: \"$ETCD_ENDPOINTS\"|" calico.yaml

      # Update the clusterIP of the etcd service as well
      # Take a schema, e.g. "https://10.1.1.9:6666", and convert to IPv4 address, e.g. "10.1.1.9"
      local ETCD_IP=$(echo "$ETCD_ENDPOINTS" | sed 's/.*\(http:\/\/\|https:\/\/\)//' | sed 's/:.*//')
      sed -Ei "s|clusterIP:.*|clusterIP: \"$ETCD_IP\"|" calico.yaml
    fi

    echo "Setting etcd_endpoints to: ${ETCD_ENDPOINTS}"
  fi
}

#
# createCalicoctlCfg() - create or replace "/etc/calico/calicoctl.cfg"
#
createCalicoctlCfg() {
  local cfgFile="/etc/calico/calicoctl.cfg"

  # etcd: create or replace /etc/calico/calicoctl.cfg
  if [ "$DATASTORE" == "etcdv3" ]; then

    # Sanity checking
    if [ -z "$ETCD_ENDPOINTS" ]; then
      fatalError "ETCD_ENDPOINTS not set"
    fi

    cat > calicoctl.cfg <<EOF
apiVersion: projectcalico.org/v3
kind: CalicoAPIConfig
metadata:
spec:
  etcdEndpoints: ${ETCD_ENDPOINTS}
EOF

  # kubernetes: create or replace /etc/calico/calicoctl.cfg
  elif [ "$DATASTORE" == "kubernetes" ]; then
    local kubeConfig="$HOME"/.kube/config
    if [ ! -f "$kubeConfig" ]; then
      fatalError "Did not find $kubeConfig"
    fi

    cat > calicoctl.cfg <<EOF
apiVersion: projectcalico.org/v3
kind: CalicoAPIConfig
metadata:
spec:
  datastoreType: "kubernetes"
  kubeconfig: "${kubeConfig}"
EOF
  fi

  runAsRoot "mkdir -p /etc/calico/"
  runAsRoot "mv calicoctl.cfg $cfgFile"
}

#
# deleteCalicoBinaries() - delete "/etc/calico/calicoctl.cfg",
# calicoctl, and calicoq binaries.
#
deleteCalicoBinaries() {
  local cfgFile="/etc/calico/calicoctl.cfg"
  runAsRoot "rm -f $cfgFile"
  runAsRoot "rm -f ${CALICO_UTILS_INSTALL_DIR}/calicoctl"
  runAsRoot "rm -f ${CALICO_UTILS_INSTALL_DIR}/calicoq"
}

#
# installCalicoBinary() - takes utility name as arg, e.g. "calicoctl".
# Pulls the appropriate container image, copies the binary to local
# file system (/usr/local/bin/ by default). Also creates calicoctl
# config file.
#
installCalicoBinary() {
  dockerLogin             # login to quay.io/tigera or gcr.io

  local utilityName=$1    # either "calicoctl" or "calicoq"

  # Validate arg
  [ "$utilityName" == "calicoctl" ] || [ "$utilityName" == "calicoq" ] || fatalError "Utility name \"${utilityName}\" is not valid, must be either \"calicoctl\" or \"calicoq\"."

  # We've already downloaded the utility manifests, i.e. "calicoctl.yaml"
  # or "calicoq.yaml". Perform sanity checking to make sure they exist.
  utilityManifest=${utilityName}.yaml

  if [ ! -f ${utilityManifest} ]; then
    fatalError "Unable to locate ${utilityManifest}"
  fi

  # Extract the versioned container url from the appropriate manifest, e.g. "quay.io/tigera/calicoq:v2.1.0-rc1"
  utilityContainerURL=$(grep image: ${utilityManifest} | sed 's/\s*image:\s*//')

  # Pull utility's container image
  echo -n "Pulling ${utilityContainerURL} ... "
  runAsRoot "docker pull ${utilityContainerURL}"
  echo "done."

  # Create a local copy of utility image
  echo -n "Copying the ${utilityContainerURL} container ... "
  runAsRootIgnoreErrors "docker rm calico-utility-copy"
  runAsRoot "docker create --name calico-utility-copy ${utilityContainerURL}"
  echo "done."

  # Copy binary to current directory
  echo -n "Copying \"${utilityName}\" to current directory ... "
  runAsRoot "docker cp calico-utility-copy:/${utilityName} ./${utilityName}"
  runAsRoot "chmod +x ./${utilityName}"
  echo "done."

  # Copy binary to ${CALICO_UTILS_INSTALL_DIR} (/usr/local/bin/)
  echo -n "Installing ${utilityName} in ${CALICO_UTILS_INSTALL_DIR} ... "
  runAsRoot "mkdir -p ${CALICO_UTILS_INSTALL_DIR}"
  runAsRoot "cp ./${utilityName} ${CALICO_UTILS_INSTALL_DIR}"
  echo "done."

  # Clean up
  runAsRootIgnoreErrors "docker rm calico-utility-copy"
  runAsRoot "docker rmi ${utilityContainerURL}"
  runAsRoot "rm -f ./${utilityName}"

  createCalicoctlCfg    # If not already present, create "/etc/calico/calicoctl.cfg"
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
# setPodCIDR()
# Read pod cidr from kube-controller-manager manifest and replace default 192.168.0.0/16
#
setPodCIDR() {
  local filename="$1"
  local cidrMatch="[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\/[0-9]\{1,\}"

  podCIDR=`grep -o "\--cluster-cidr=$cidrMatch" $KUBE_CONTROLLER_MANIFEST | grep -o $cidrMatch`

  cat > sedcmd.txt <<EOF
s?192.168.0.0/16?$podCIDR?g
EOF

  sed -i -f sedcmd.txt "$filename"

  echo "Set pod CIDR $podCIDR for $filename"
}

#
# downloadManifests() - download all required manifests. Note
# that if a particular manifest exists already in current dir,
# do not download/overwrite that manifest.
#
downloadManifests() {
  downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml"
  downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml"
  downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml"
  downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/helm/tigera-secure-ee/operator-crds.yaml"


  # Irrespective of datastore type, for federation we'll need to create a federation secret since the manifests assume
  # this exists.
  if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/tigera-federation-secret.yaml"
  fi

  if [ "$DATASTORE" == "etcdv3" ]; then
    if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/federation/calico.yaml"
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/rbac-etcd-typha.yaml"
    else
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/calico.yaml"
    fi

    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/etcd.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-api-etcd.yaml"

    # Grab calicoctl and calicoq manifests in order to extract the container url when we install the binaries
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/calicoctl.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/calicoq.yaml"
  else
    if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/federation/calico.yaml"
    elif [ "$DEPLOYMENT_TYPE" == "typha" ]; then
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/typha/calico.yaml"
    else
      if [ "${NETWORKING}" == "other" ]; then
        downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only/1.7/calico.yaml"
      elif [ "${NETWORKING}" == "aws" ]; then
        downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only-ecs/1.7/calico.yaml"
      else
        downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml"
      fi
    fi

    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-api-kdd.yaml"

    # Grab calicoctl and calicoq manifests in order to extract the container url when we install the binaries
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calicoctl.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calicoq.yaml"
  fi

  downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx.yaml"

  if [ "$INSTALL_TYPE" == "ACS-ENGINE" ] && [ "$CLEANUP" -eq 0 ]; then
    setPodCIDR "calico.yaml"
  fi

  # Download appropriate elasticsearch storage manifest
  if [ "${ELASTIC_STORAGE}" != "none" ]; then
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-${ELASTIC_STORAGE}.yaml"
  fi

  if "$INSTALL_AWS_SG" ; then
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/manifests/aws-sg-integration/vpc-cf.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/manifests/aws-sg-integration/cluster-cf.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/manifests/aws-sg-integration/cloud-controllers.yaml"
  fi

  if [ "${DOWNLOAD_MANIFESTS_ONLY}" -eq 1 ]; then
    echo "Tigera Secure EE manifests downloaded."
    kill -s TERM $TOP_PID   # quit
  fi
}

#
# applyRbacManifest() - apply appropriate rbac manifest based on datastore.
#
applyRbacManifest() {
  if [ "$DATASTORE" == "etcdv3" ]; then
    if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
      # Apply rbac for etcdv3 datastore with typha (currently only required for federation)
      run kubectl apply -f rbac-etcd-typha.yaml
      countDownSecs 5 "Applying \"rbac-etcd-typha.yaml\" manifest: "
    fi
  fi
}

#
# deleteRbacManifest() - delete appropriate rbac manifest based on datastore.
#
deleteRbacManifest() {
  if [ "$DATASTORE" == "etcdv3" ]; then
    if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
      # Delete rbac for etcdv3 datastore with typha (currently only required for federation)
      runIgnoreErrors kubectl delete -f rbac-etcd-typha.yaml
      countDownSecs 5 "Deleting \"rbac-etcd-typha.yaml\" manifest: "
    fi
  fi
}

#
# applyEtcdDeployment() - etcd-only, apply etcd.yaml
#
applyEtcdDeployment() {
  if [ "$DATASTORE" == "etcdv3" ]; then
    run kubectl apply -f etcd.yaml
    countDownSecs 5 "Applying \"etcd.yaml\" manifest: "
  fi
}

#
# deleteEtcdDeployment() - etcd-only, delete etcd.yaml
#
deleteEtcdDeployment() {
  if [ "$DATASTORE" == "etcdv3" ]; then
    runIgnoreErrors kubectl delete -f etcd.yaml
    countDownSecs 5 "Deleting \"etcd.yaml\" manifest: "
  fi
}

#
# applyLicenseManifest()
# If the user specified a license file, apply it. Note there may be a race condition
# where felix (calico-node) starts before we apply the license. In this case, the
# license will be installed, but felix will think that it is unlicensed. Restart
# calico-node pod(s) as a workaround if you encounter this issue.
#
applyLicenseManifest() {
  if [ "$LICENSE_FILE" ]; then
    echo -n "Applying license file: ${LICENSE_FILE} "
    run "${CALICO_UTILS_INSTALL_DIR}/calicoctl apply -f ${LICENSE_FILE}"
    echo "done."
  fi
}

#
# applyTigeraFederationSecretManifest()
#
applyTigeraFederationSecretManifest() {
  echo -n "Applying \"tigera-federation-secret.yaml\" ("$DATASTORE") manifest: "
  run kubectl apply -f tigera-federation-secret.yaml
}

#
# deleteTigeraFederationSecretManifest()
#
deleteTigeraFederationSecretManifest() {
  echo -n "Deleting tigera-federation-secret.yaml manifest"
  runIgnoreErrors kubectl delete -f tigera-federation-secret.yaml
}

#
# createTyphaCerts()
#
createTyphaCerts() {
  openssl req -x509 -newkey rsa:4096 \
                    -keyout typhaca.key \
                    -nodes \
                    -out typhaca.crt \
                    -subj "/CN=Calico Typha CA" \
                    -days 365
  openssl req -newkey rsa:4096 \
              -keyout typha.key \
              -nodes \
              -out typha.csr \
              -subj "/CN=calico-typha"
  openssl x509 -req -in typha.csr \
                    -CA typhaca.crt \
                    -CAkey typhaca.key \
                    -CAcreateserial \
                    -out typha.crt \
                    -days 365
  openssl req -newkey rsa:4096 \
              -keyout felix.key \
              -nodes \
              -out felix.csr \
              -subj "/CN=calico-felix"
  openssl x509 -req -in felix.csr \
                    -CA typhaca.crt \
                    -CAkey typhaca.key \
                    -out felix.crt \
                    -days 3650
  sed -e "s/<replace with base64-encoded Typha certificate>/$(cat typha.crt | base64 -w 0)/" \
      -e "s/<replace with base64-encoded Typha private key>/$(cat typha.key | base64 -w 0)/" \
      -e "s/<replace with base64-encoded Felix certificate>/$(cat felix.crt | base64 -w 0)/" \
      -e "s/<replace with base64-encoded Felix private key>/$(cat felix.key | base64 -w 0)/" \
      -i calico.yaml 
  sed -e 's/^/    /' < typhaca.crt > typhaca_indented.crt
  sed '/<replace with PEM-encoded (not base64) Certificate Authority bundle>/ {
    r typhaca_indented.crt
    d
  }' -i calico.yaml
  rm typhaca_indented.crt  
}

#
# applyCalicoManifest()
#
applyCalicoManifest() {
  echo -n "Applying \"calico.yaml\" ("$DATASTORE") manifest: "
  run kubectl apply -f calico.yaml
  if [ "$DEPLOYMENT_TYPE" == "typha" ] || [ "$DEPLOYMENT_TYPE" == "federation" ]; then
    updateTyphaReplicas 1 # Set the number of replicas of the typha deployment to 1
  fi
  blockUntilPodIsReady "k8s-app=calico-node" 180 "calico-node" # Block until calico-node pod is running & ready
  blockUntilPodIsReady "k8s-app=kube-dns" 180 "kube-dns"       # Block until kube-dns pod is running & ready
}

#
# deleteCalicoManifest()
#
deleteCalicoManifest() {
  runIgnoreErrors kubectl delete -f calico.yaml
  countDownSecs 5 "Deleting calico.yaml manifest"
}

#
# updateTyphaReplicas()
#
updateTyphaReplicas() {
  local replicas="$1"
  echo "Scaling deployment/calico-typha replicas to ${replicas}"
  run kubectl scale deployment/calico-typha --replicas=${replicas} --namespace=kube-system
  blockUntilPodIsReady "k8s-app=calico-typha" 180 "calico-typha"
}

#
# validateDatastore() - warn the user if they're switching
# between kubernetes/etcdv3 datastore, but leaving the wrong
# manifest laying around (specifically calico.yaml). Use the
# existence of cnx-api-kdd.yaml|cnx-api-etcd.yaml as an indicator for
# which type of install/uninstall the user tried earlier.
#
function validateDatastore() {
  local operation="$1"           # install, uninstall
  local installedManifest=""     # set to "kdd" or "etcd" if there's a problem.

  # Check if we're installing one datastore type but there's a manifest
  # for the "opposite" datastore type laying around the current directory.

  if [ "$DATASTORE" == "etcdv3" ]; then
    if [ -f "cnx-api-kdd.yaml" ]; then
      installedManifest="kdd"
      datastoreFlag="kubernetes"
    fi
  elif [ -f "cnx-api-etcd.yaml" ]; then
      installedManifest="etcd"
      datastoreFlag="etcdv3"
  fi

  if [ "$installedManifest" ]; then
    echo
    echo "Warning: the current $operation specifies \"$DATASTORE\", however \"cnx-api-$installedManifest.yaml\" exists"
    echo "         in the current directory. Either remove all the manifests from the"
    echo "         currect directory or use the \"-k $datastoreFlag\" flag and restart the $operation."
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
# createCNXAPICerts
#
createCNXAPICerts() {
  openssl req -x509 -newkey rsa:4096 \
                    -keyout apiserver.key \
                    -nodes \
                    -out apiserver.crt \
                    -subj "/CN=cnx-api.kube-system.svc" \
                    -days 365
  if [ "$DATASTORE" == "kubernetes" ]; then
    apimanifest="cnx-api-kdd.yaml"
  elif [ "$DATASTORE" == "etcdv3" ]; then
    apimanifest="cnx-api-etcd.yaml"
  fi
  sed -e "s/<replace with base64 encoded certificate>/$(cat apiserver.crt | base64 -w 0)/" \
      -e "s/<replace with base64 encoded private key>/$(cat apiserver.key | base64 -w 0)/" \
      -e "s/<replace with base64 encoded Certificate Authority bundle>/$(cat apiserver.crt | base64 -w 0)/" \
      -i $apimanifest
}

#
# applyCNXAPIManifest()
#
applyCNXAPIManifest() {
  if [ "$DATASTORE" == "kubernetes" ]; then
    echo -n "Applying \"cnx-api-kdd.yaml\" manifest: "
    run kubectl apply -f cnx-api-kdd.yaml
  elif [ "$DATASTORE" == "etcdv3" ]; then
    echo -n "Applying \"cnx-api-etcd.yaml\" manifest: "
    run kubectl apply -f cnx-api-etcd.yaml
  fi
  blockUntilPodIsReady "k8s-app=cnx-apiserver" 180 "cnx-apiserver"  # Block until cnx-apiserver pod is running & ready
  if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
    blockUntilPodIsReady "k8s-app=tigera-federation-controller" 180 "tigera-federation-controller"  # Block until tigera-federation-controller pod is running & ready
  fi
  countDownSecs 10 "Waiting for cnx-apiserver to stabilize"         # Wait until cnx-apiserver completes registration w/kube-apiserver
}

#
# deleteCNXAPIManifest()
#
deleteCNXAPIManifest() {
  if [ "$DATASTORE" == "kubernetes" ]; then
    runIgnoreErrors "kubectl delete -f cnx-api-kdd.yaml"
    countDownSecs 30 "Deleting \"cnx-api-kdd.yaml\" manifest"
  elif [ "$DATASTORE" == "etcdv3" ]; then
    runIgnoreErrors "kubectl delete -f cnx-api-etcd.yaml"
    countDownSecs 30 "Deleting \"cnx-api-etcd.yaml\" manifest"
  fi
}

#
# applyCNXManifest()
#
applyCNXManifest() {
  echo -n "Applying \"cnx.yaml\" manifest: "
  run kubectl apply -f cnx.yaml
  blockUntilPodIsReady "k8s-app=cnx-manager" 180 "cnx-manager"      # Block until cnx-manager pod is running & ready
  blockUntilPodIsReady "k8s-app=compliance-controller" 180 "compliance-controller"
  blockUntilPodIsReady "k8s-app=compliance-server" 180 "compliance-server"
  blockUntilPodIsReady "k8s-app=compliance-snapshotter" 180 "compliance-snapshotter"
  blockUntilPodIsReady "k8s-app=compliance-benchmarker" 180 "compliance-benchmarker"
  countDownSecs 10 "Waiting for cnx-manager to stabilize"         # Wait until cnx-manager starts to run.
}

#
# deleteCNXManifest()
#
deleteCNXManifest() {
  runIgnoreErrors "kubectl delete -f cnx.yaml"
  countDownSecs 30 "Deleting \"cnx.yaml\" manifest"
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
# applyElasticStorageManifest()
#
applyElasticStorageManifest() {
  run kubectl apply -f elastic-storage-${ELASTIC_STORAGE}.yaml
  countDownSecs 10 "Applying \"elastic-storage-${ELASTIC_STORAGE}.yaml\" manifest"
}

#
# deleteElasticStorageManifest()
#
deleteElasticStorageManifest() {
  runIgnoreErrors kubectl delete -f elastic-storage-${ELASTIC_STORAGE}.yaml
  countDownSecs 20 "Deleting \"elastic-storage-${ELASTIC_STORAGE}.yaml\" manifest"
}

#
# applyElasticCRDManifest()
#
applyElasticCRDManifest() {
  run kubectl apply -f operator-crds.yaml
  countDownSecs 10 "Applying \"operator-crds.yaml\" manifest"
}

#
# deleteElasticCRDManifest()
#
deleteElasticCRDManifest() {
  runIgnoreErrors kubectl delete -f operator-crds.yaml
  countDownSecs 10 "Deleting \"operator-crds.yaml\" manifest"
}

#
# applyOperatorManifest()
#
applyOperatorManifest() {
  echo -n "Applying \"operator.yaml\" manifest: "
  run kubectl apply -f operator.yaml
  blockUntilPodIsReady "control-plane=elastic-operator" 180 "elastic-operator-0"      # Block until prometheus-calico-nod pod is running & ready
}

#
# deleteOperatorManifest()
#
deleteOperatorManifest() {
  runIgnoreErrors kubectl delete elasticsearch tigera-elasticsearch -n calico-monitoring
  countDownSecs 5 "Deleting tigera-kibana"
  runIgnoreErrors kubectl delete kibana tigera-kibana -n calico-monitoring
  countDownSecs 5 "Deleting tigera-elasticsearch"
  runIgnoreErrors kubectl delete -f operator.yaml
  countDownSecs 5 "Deleting \"operator.yaml\" manifest"
  runIgnoreErrors kubectl delete daemonset elasticsearch-operator-sysctl
  countDownSecs 5 "Deleting daemonset \"elasticsearch-operator-sysctl\" created by operator"
}

#
# applyMonitorCalicoManifest()
#
applyMonitorCalicoManifest() {
  # if given tighten the address used to allow access only to K8S API Server.
  if [ -n "${REPLACE_K8S_CIDR}" ]; then
    sed -i "s|\(.*\)- .* #K8S_API_SERVER_IP|\1- \"$REPLACE_K8S_CIDR\"|g" monitor-calico.yaml
  fi
  echo -n "Applying \"monitor-calico.yaml\" manifest: "
  run kubectl apply -f monitor-calico.yaml
  blockUntilPodIsReady "app=prometheus" 180 "prometheus-calico-node"      # Block until prometheus-calico-nod pod is running & ready
  blockUntilPodIsReady "elasticsearch.k8s.elastic.co/cluster-name=tigera-elasticsearch" 180 "elasticsearch-client"      # Block until elasticsearch-client pod is running & ready
  blockUntilPodIsReady "name=tigera-kibana" 180 "kibana"    # Block until kibana pod is running & ready
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
  # Note the below sequence for cert generation is different from all others in this script because of an issue 
  # with parsing a key generated using openssl -newkey flag inside Voltron (the new proxy for CNX Manager). 
  # For more details on this problem please see: https://tigera.atlassian.net/browse/SAAS-390
  # Generating the key separately from the openssl cert generation appears to get around this problem. 
  ssh-keygen -m PEM -b 4096 -t rsa -f manager.key -N ""
  openssl req -x509 -new \
                  -key manager.key \
                  -nodes \
                  -out manager.crt \
                  -subj "/CN=cnx-manager.calico-monitoring.svc" \
                  -days 3650

  runAsRoot kubectl create secret generic cnx-manager-tls --from-file=cert=./manager.crt --from-file=key=./manager.key -n calico-monitoring
  countDownSecs 5 "Creating cnx-manager-tls secret"

  run rm -f manager.key 
}

#
# deleteCNXManagerSecret()
#
deleteCNXManagerSecret() {
  runAsRootIgnoreErrors kubectl delete secret cnx-manager-tls -n calico-monitoring
  echo "Deleting cnx-manager-tls secret"
}

#
# checkAwsIntegration() - Check
#
checkAwsIntegration() {
  fetchAwsVpcId
  checkAwsIntegrationSgEnvVars
  checkAwsIntegrationCloudTrail
}

#
# fetchAwsVpcId() - Fetch the VPC ID from the metadata
#
fetchAwsVpcId() {
  if [ -z "$VPC_ID" ]; then
    mac=`runGetOutput curl -s http://169.254.169.254/latest/meta-data/network/interfaces/macs/ | head -1`

    VPC_ID=`runGetOutput curl -s http://169.254.169.254/latest/meta-data/network/interfaces/macs/${mac}vpc-id`
  fi
}

#
# checkAwsIntegrationSgEnvVars() - Check Env vars for AWS SG installation
#
checkAwsIntegrationSgEnvVars() {
  prefix="Failed pre-condition for AWS Security Group Integration:"
  [ -z "$VPC_ID" ] && fatalError "$prefix No VPC_ID is set."
  # If cleaning up we do not need the control plane and node SGs
  if [ "$CLEANUP" -ne 1 ]; then
    [ -z "$CONTROL_PLANE_SG" ] && fatalError "$prefix No CONTROL_PLANE_SG is set."
    [ -z "$K8S_NODE_SGS" ] && fatalError "$prefix No K8S_NODE_SGS is set."
  fi
}

#
# checkAwsIntegrationCloudTrail() - Check AWS account CloudTrail exists
#
checkAwsIntegrationCloudTrail() {
  echo "Checking that 'tigera-cloudtrail' CloudStack is created for the account"
  run aws cloudformation describe-stacks --stack-name tigera-cloudtrail
}

#
# createAwsSgCloudStacks() - Create the CloudStacks that are needed for each VPC/cluster
#
createAwsSgCloudStacks() {
  echo -n"Creating Cloudformation stacks ... "
  run aws cloudformation create-stack \
    --stack-name tigera-vpc-$VPC_ID \
    --parameters ParameterKey=VpcId,ParameterValue=$VPC_ID \
    --capabilities CAPABILITY_IAM \
    --template-body file://vpc-cf.yaml

  run aws cloudformation create-stack \
    --stack-name tigera-cluster-$CLUSTER_NAME \
    --parameters ParameterKey=VpcId,ParameterValue=$VPC_ID \
    ParameterKey=KubernetesHostDefaultSGId,ParameterValue=$K8S_NODE_SGS \
    ParameterKey=KubernetesControlPlaneSGId,ParameterValue=$CONTROL_PLANE_SG \
    --template-body file://cluster-cf.yaml

  run aws cloudformation wait stack-create-complete --stack-name tigera-vpc-$VPC_ID
  run aws cloudformation wait stack-create-complete --stack-name tigera-cluster-$CLUSTER_NAME
  echo "done"
}

#
# createAwsSgControllerSecret() - Fetch the needed information then create the access
#    key for the controller then load that into a K8s secret
createAwsSgControllerSecret() {
  # First, get the name of the created IAM user, which is an output field in your Cluster CF stack
  CONTROLLER_USERNAME=$(runGetOutput aws cloudformation describe-stacks \
      --stack-name tigera-vpc-$VPC_ID \
      --output text \
      --query "Stacks[0].Outputs[?OutputKey=='TigeraControllerUserName'][OutputValue]")

  echo "Creating access key for controller user $CONTROLLER_USERNAME"
  # Then create an access key for that role
  runGetOutput aws iam create-access-key \
      --user-name $CONTROLLER_USERNAME \
      --output text \
      --query "AccessKey.{Key:SecretAccessKey,ID:AccessKeyId}" > controller-secrets.txt

  echo "Creating secret to store cloud controller user credentials"
  # Add the key as a k8s secret
  cat controller-secrets.txt | xargs bash -c \
      'kubectl create secret generic tigera-cloud-controllers-credentials \
      -n kube-system \
      --from-literal=aws_access_key_id=$0 \
      --from-literal=aws_secret_access_key=$1'

  # Check that the credentials were created
  run kubectl get secret tigera-cloud-controllers-credentials -n kube-system

  # Delete local copy of the secret
  rm -f controller-secrets.txt
}

#
# createAwsSgConfigMap() - Gather the needed information and then create the
#   tigera-aws-config ConfigMap
createAwsSgConfigMap() {
  # Get the SQS URL
  SQS_URL=$(runGetOutput aws cloudformation describe-stacks \
      --stack-name tigera-vpc-$VPC_ID \
      --output text \
      --query "Stacks[0].Outputs[?OutputKey=='QueueURL'][OutputValue]")

  # Get the default pod SG
  POD_SG=$(runGetOutput aws cloudformation describe-stacks \
      --stack-name tigera-cluster-$CLUSTER_NAME \
      --output text \
      --query "Stacks[0].Outputs[?OutputKey=='TigeraDefaultPodSG'][OutputValue]")

  # Get the SG for enforced nodes
  ENFORCED_SG=$(runGetOutput aws cloudformation describe-stacks \
      --stack-name tigera-vpc-$VPC_ID \
      --output text \
      --query "Stacks[0].Outputs[?OutputKey=='TigeraEnforcedSG'][OutputValue]")

  # Get the SG for enforced nodes
  TRUST_SG=$(runGetOutput aws cloudformation describe-stacks \
      --stack-name tigera-vpc-$VPC_ID \
      --output text \
      --query "Stacks[0].Outputs[?OutputKey=='TigeraTrustEnforcedSG'][OutputValue]")

  REGION=`runGetOutput curl -s http://169.254.169.254/latest/dynamic/instance-identity/document | jq-container -r '.region'`

  echo "Creating tigera-aws-config"
  # Store both in a new configmap
  run kubectl create configmap tigera-aws-config \
      -n kube-system \
      --from-literal=aws_region=$REGION \
      --from-literal=vpcs=$VPC_ID \
      --from-literal=sqs_url=$SQS_URL \
      --from-literal=pod_sg=$POD_SG \
      --from-literal=default_sgs=$K8S_NODE_SGS \
      --from-literal=enforced_sg=$ENFORCED_SG \
      --from-literal=trust_sg=$TRUST_SG
}

#
# restartAwsConfigDependentPods() - Restart the tigera pods that depend on the
#   tigera-aws-config ConfigMap since updating it does not cause them to restart
restartAwsConfigDependentPods() {
  echo "Restarting components that reference tigera-aws-config ConfigMap ... "
  for podLabel in calico-typha cnx-apiserver calico-node; do
    runIgnoreErrors kubectl -n kube-system delete pod -l k8s-app=$podLabel
  done

  blockUntilPodIsReady "k8s-app in (calico-typha, cnx-apiserver, calico-node)" 300 AWSConfigDependentPods
}

#
# removeAwsSgIntegrationSgsFromAwsResources() - Remove the Tigera Security Groups
#   from the AWS resources so the Security Groups can be deleted, when the
#   CloudStacks are deleted.
removeAwsSgIntegrationSgsFromAwsResources() {
  # Get the SG for enforced nodes
  ENFORCED_SG=$(runGetOutputIgnoreErrors aws cloudformation describe-stacks \
      --stack-name tigera-vpc-$VPC_ID \
      --output text \
      --query "Stacks[0].Outputs[?OutputKey=='TigeraEnforcedSG'][OutputValue]")

  # Get the SG for enforced nodes
  TRUST_SG=$(runGetOutputIgnoreErrors aws cloudformation describe-stacks \
      --stack-name tigera-vpc-$VPC_ID \
      --output text \
      --query "Stacks[0].Outputs[?OutputKey=='TigeraTrustEnforcedSG'][OutputValue]")

  # Remove the tigera security groups from all instances.
  # This needs to be done so the delete-stacks below will succeed.
  # This could be further improved to remove them from RDS and LoadBalancers
  for sg in $ENFORCED_SG $TRUST_SG; do
    aws ec2 describe-network-interfaces --filters Name=group-id,Values=$sg \
    | jq-container -r '.NetworkInterfaces[] | .NetworkInterfaceId' | while read interface ;
    do
        sgs=$(aws ec2 describe-network-interfaces --network-interface-ids $interface \
            | jq-container -r '.NetworkInterfaces[] | .Groups[].GroupId' \
            | sed -e "s/$sg//")
        aws ec2 modify-network-interface-attribute --network-interface $interface --groups $sgs
    done
  done
}

#
# deleteAwsSgCrdResources() - Clean up the AWS Security Group integration resources
#
deleteAwsSgIntegrationResources() {
  echo "Deleting all resources associated with the AWS Security Group Integration"
  runIgnoreErrors kubectl delete globalnetworkpolicies -l 'projectcalico.org/tier in (sg-local, sg-remote, metadata)'
  runIgnoreErrors kubectl delete hostendpoints -l tigera.io/managed-hep
}

#
# installAwsSg() - install AWS SG integration
#
installAwsSg() {
  createAwsSgCloudStacks

  createAwsSgControllerSecret

  createAwsSgConfigMap

  # Update calico-node with failsafes
  # We could do this to exactly copy the installation directions but it is
  # not necessary.

  # Restart components
  restartAwsConfigDependentPods

  # Install cloud-controller
  echo -n "Applying \"cloud-controllers.yaml\" manifest: "
  run kubectl apply -f cloud-controllers.yaml
  # Block until cloud-controllers pod is running & ready
  blockUntilPodIsReady "k8s-app=tigera-cloud-controllers" 300 "cloud-controllers"
}

#
# uninstallAwsSg() - uninstall AWS SG integration
#
uninstallAwsSg() {
  runIgnoreErrors kubectl delete -f cloud-controllers.yaml
  countDownSecs 10 "Deleting cloud-controllers.yaml manifest"

  deleteAwsSgIntegrationResources

  removeAwsSgIntegrationSgsFromAwsResources

  runIgnoreErrors kubectl delete configmap tigera-aws-config -n kube-system

  restartAwsConfigDependentPods

  runIgnoreErrors aws cloudformation delete-stack \
    --stack-name tigera-vpc-$VPC_ID

  runIgnoreErrors aws cloudformation delete-stack \
    --stack-name tigera-cluster-$CLUSTER_NAME

  runIgnoreErrors kubectl delete secret tigera-cloud-controllers-credentials \
      -n kube-system
}

#
# reportSuccess() - tell user how to login
#
reportSuccess() {
  echo
  echo "---------------------------------------------------------------------------------------"
  echo "Tigera Secure EE installation is complete. Tigera Secure EE is listening on all network interfaces on port 30003."
  echo "For example:"
  echo "  https://127.0.0.1:30003"
  echo "  https://$(ip route get 8.8.8.8 | head -1 | awk '{print $7}'):30003"
  echo

  if [ "$INSTALL_TYPE" == "KUBEADM" ]; then
    echo "Login credentials: username=\"jane\", password=\"`cat cnx_jane_pw`\""
  elif [ "$INSTALL_TYPE" == "KOPS" ]; then
    echo "Login credentials: username=\"admin\", password=\`kops get secrets kube -o plaintext\`"
  fi
}

#
# installCNX() - install Tigera Secure EE
#
installCNX() {
  checkNumberOfCores              # Warn user if insufficient processor cores are present (>= 2).
  checkRequiredFilesPresent       # Validate kubernetes files are present
  checkNetworkManager             # Warn user if NetworkMgr is enabled w/o "cali" interface exception

  downloadManifests               # Download all manifests, if they're not already present in current dir
  createImagePullSecret           # Create the image pull secret
  setupBasicAuth                  # Create 'jane/******' account w/cluster admin privs

  setupEtcdEndpoints              # Determine etcd endpoints, set ${ETCD_ENDPOINTS}

  installCalicoBinary "calicoctl" # Install quay.io/tigera/calicoctl binary, create "/etc/calico/calicoctl.cfg"
  installCalicoBinary "calicoq"   # Install quay.io/tigera/calicoq binary

  if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
    applyTigeraFederationSecretManifest # Apply the tigera-federation-remotecluster secret
  fi

  applyRbacManifest               # Apply RBAC resources based on the configured datastore type.
  applyEtcdDeployment             # Install a single-node etcd (etcd datastore only)
  if [ "$DEPLOYMENT_TYPE" == "typha" ] || [ "$DEPLOYMENT_TYPE" == "federation" ]; then
    createTyphaCerts              # Create TLS certificates for Felix and Typha communication
  fi
  applyCalicoManifest             # Apply calico.yaml

  removeMasterTaints              # Remove master taints
  createCNXAPICerts               # Create TLS certificates for the CNX APIServer
  applyCNXAPIManifest             # Apply cnx-api-[etcd|kdd].yaml

  applyLicenseManifest            # If the user specified a license file, apply it

  applyCNXPolicyManifest          # Apply cnx-policy.yaml
  applyElasticStorageManifest     # Apply elastic-storage.yaml
  applyElasticCRDManifest        # Apply operator-crds.yaml containing Elasticsearch, Kibana and Prometheus resources
  applyOperatorManifest           # Apply operator.yaml
  applyMonitorCalicoManifest      # Apply monitor-calico.yaml
  createCNXManagerSecret          # Create cnx-manager-tls to enable manager/apiserver communication
  applyCNXManifest                # Apply cnx.yaml
}

#
# uninstallCNX() - remove Tigera Secure EE and related changes.
# Ignore errors.
#
uninstallCNX() {
  deleteCalicoBinaries           # delete /etc/calico/calicoct.cfg, calicoctl, and calicoq

  downloadManifests              # Download all manifests
  deleteCNXManifest              # Delete cnx.yaml
  deleteCNXManagerSecret         # Delete TLS secret

  deleteMonitorCalicoManifest    # Delete monitor-calico.yaml
  deleteOperatorManifest         # Delete operator.yaml
  deleteElasticCRDManifest      # Delete operator-crds.yaml containing Elasticsearch, Kibana and Prometheus resources
  deleteElasticStorageManifest   # Delete elastic-storage.yaml
  deleteCNXPolicyManifest        # Delete cnx-policy.yaml

  deleteCNXAPIManifest           # Delete cnx-api-[etcd|kdd].yaml
  deleteCalicoManifest           # Delete calico.yaml
  deleteRbacManifest             # Delete rbac.yaml
  deleteEtcdDeployment           # Delete etcd.yaml (etcd datatstore only)
  deleteImagePullSecret          # Delete pull secret
  deleteBasicAuth                # Remove basic auth updates, restart kubelet
  deleteKubeDnsPod               # Return kube-dns pod to "pending" state

  if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
    deleteTigeraFederationSecretManifest # Delete the tigera-federation-remotecluster secret
  fi
}

#
# main()
#
parseOptions "$@"               # Set up variables based on args
checkRequirementsInstalled      # Validate that all required programs are installed

determineInstallerType          # Set INSTALL_TYPE to "KOPS" or "KUBEADM" or "ACS-ENGINE"
validateSettings                # Check for missing files/dirs

if [ "$CLEANUP" -eq 1 ]; then
  checkSettings uninstall       # Verify settings are correct with user
  validateDatastore uninstall   # Warn if there's etcd manifest, but we're doing kdd uninstall (and vice versa)

  if "$INSTALL_AWS_SG"; then
    uninstallAwsSg              # Removes the AWS SG integration
  fi
  if ! "$SKIP_EE_INSTALLATION"; then
    uninstallCNX                # Remove Tigera Secure EE, cleanup related files
  fi
else
  checkSettings install         # Verify settings are correct with user
  validateDatastore install     # Warn if there's etcd manifest, but we're doing kdd install (and vice versa)

  if ! "$SKIP_EE_INSTALLATION"; then
    installCNX                  # Install Tigera Secure EE
  fi
  if "$INSTALL_AWS_SG"; then
    installAwsSg                # Install AWS SG integration, depends on EE already being installed
  fi

  reportSuccess                 # Tell user how to login
fi
