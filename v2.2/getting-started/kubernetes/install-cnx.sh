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

# when set to 1, don't install monitoring components
SKIP_MONITORING=${SKIP_MONITORING:=0}

# when set to 1, download the manifests, then quit
DOWNLOAD_MANIFESTS_ONLY=${DOWNLOAD_MANIFESTS_ONLY:=0}

# Deployment type of the tigera installation.  One of basic, typha or federation.
# A deployment type of "typha" is only valid for kubernetes datastore.
DEPLOYMENT_TYPE=${DEPLOYMENT_TYPE:="basic"}

# when set to 1, install policy-only manifest
# Only used for the kubernetes datastore.
POLICY_ONLY=${POLICY_ONLY:=0}

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
          [-l license.yaml]    # Specify the path to the Tigera Secure EE license file; default "license.yaml". Note license is required.
          [-c config.json]     # Docker authentication config file (from Tigera); default: "config.json"
          [-d docs_location]   # Tigera Secure EE documentation location; default: "https://docs.tigera.io"
          [-e etcd_endpoints]  # etcd endpoint address, e.g. ("http://10.0.0.1:2379"); default: take from manifest automatically
          [-k datastore]       # Specify the datastore ("etcdv3"|"kubernetes"); default: "etcdv3"
          [-s elastic_storage] # Specify the elasticsearch storage to use ("none"|"local"); default: "local"
          [-t deployment_type] # Specify the deployment type ("basic"|"typha"|"federation"); default "basic"
          [-v version]         # Tigera Secure EE version; default: "v2.1"
          [-u]                 # Uninstall Tigera Secure EE
          [-q]                 # Quiet (don't prompt)
          [-m]                 # Download manifests (then quit)
          [-h]                 # Print usage
          [-x]                 # Enable verbose mode

HELP_USAGE
    exit 1
  }

  local OPTIND
  while getopts "c:d:e:hk:l:mpqs:t:v:ux" opt; do
    case ${opt} in
      c )  CREDENTIALS_FILE=$OPTARG;;
      d )  DOCS_LOCATION=$OPTARG;;
      e )  ETCD_ENDPOINTS=$OPTARG;;
      k )  DATASTORE=$OPTARG;;
      l )  LICENSE_FILE=$OPTARG;;
      s )  ELASTIC_STORAGE=$OPTARG;;
      t )  DEPLOYMENT_TYPE=$OPTARG;;
      v )  VERSION=$OPTARG;;
      x )  set -x;;
      q )  QUIET=1;;
      p )  SKIP_MONITORING=1;;
      m )  DOWNLOAD_MANIFESTS_ONLY=1;;
      u )  CLEANUP=1;;
      h )  usage;;
      \? ) usage;;
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
    POLICY_ONLY=1
  fi
}

#
# checkRequirementsInstalled() - check package dependencies
#
checkRequirementsInstalled() {
  programIsInstalled jq || fatalError "Please install \"jq\" and re-run $(basename $0)".
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
    dockerCredentials=$(cat "${CREDENTIALS_FILE}" | jq --raw-output '.auths[].auth' | base64 -d)
    username=$(echo -n $dockerCredentials | awk -F ":" '{print $1}')
    token=$(echo -n $dockerCredentials | awk -F ":" '{print $2}')
  fi

  echo -n "Logging in to ${CALICO_REGISTRY} ... "
  docker login --username=${username} --password="${token}" ${CALICO_REGISTRY}
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

    # $ETCD_ENDPOINTS is ""; user wants to use the default etcd endpoint specified in "calico.yaml"
    if [ -z "${ETCD_ENDPOINTS}" ]; then

      # Extract etcd server url from "calico.yaml"
      export ETCD_ENDPOINTS=$(grep etcd_endpoints calico.yaml | grep -i http | sed 's/etcd_endpoints: //g')

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
  downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/kibana-dashboards.yaml"

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
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/rbac.yaml"
    fi

    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/etcd.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml"

    # Grab calicoctl and calicoq manifests in order to extract the container url when we install the binaries
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/calicoctl.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/calicoq.yaml"
  else
    if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/federation/calico.yaml"
    elif [ "$DEPLOYMENT_TYPE" == "typha" ]; then
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/typha/calico.yaml"
    elif [ "${POLICY_ONLY}" -eq 1 ]; then
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only/1.7/calico.yaml"
    else
      downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml"
    fi

    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-kdd.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml"

    # Grab calicoctl and calicoq manifests in order to extract the container url when we install the binaries
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calicoctl.yaml"
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calicoq.yaml"
  fi

  if [ "$INSTALL_TYPE" == "ACS-ENGINE" ] && [ "$CLEANUP" -eq 0 ]; then
    setPodCIDR "calico.yaml"
  fi

  # Download appropriate elasticsearch storage manifest
  if [ "${ELASTIC_STORAGE}" != "none" ]; then
    downloadManifest "${DOCS_LOCATION}/${VERSION}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-${ELASTIC_STORAGE}.yaml"
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
  if [ "$DATASTORE" == "kubernetes" ]; then
    # Apply rbac for kdd datastore
    run kubectl apply -f rbac-kdd.yaml
    countDownSecs 5 "Applying \"rbac-kdd.yaml\" manifest: "
  elif [ "$DATASTORE" == "etcdv3" ]; then
    if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
      # Apply rbac for etcdv3 datastore with typha (currently only required for federation)
      run kubectl apply -f rbac-etcd-typha.yaml
      countDownSecs 5 "Applying \"rbac-etcd-typha.yaml\" manifest: "
    else
      # Apply rbac for etcdv3 datastore
      run kubectl apply -f rbac.yaml
      countDownSecs 5 "Applying \"rbac.yaml\" manifest: "
    fi
  fi
}

#
# deleteRbacManifest() - delete appropriate rbac manifest based on datastore.
#
deleteRbacManifest() {
  if [ "$DATASTORE" == "kubernetes" ]; then
    # Delete rbac for kdd datastore
    runIgnoreErrors kubectl delete -f rbac-kdd.yaml
    countDownSecs 5 "Deleting \"rbac-kdd.yaml\" manifest: "
  elif [ "$DATASTORE" == "etcdv3" ]; then
    if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
      # Delete rbac for etcdv3 datastore with typha (currently only required for federation)
      runIgnoreErrors kubectl delete -f rbac-etcd-typha.yaml
      countDownSecs 5 "Deleting \"rbac-etcd-typha.yaml\" manifest: "
    else
      # Delete rbac for etcdv3 datastore
      runIgnoreErrors kubectl delete -f rbac.yaml
      countDownSecs 5 "Deleting \"rbac.yaml\" manifest: "
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
# existence of cnx-kdd.yaml|cnx-etcd.yaml as an indicator for
# which type of install/uninstall the user tried earlier.
#
function validateDatastore() {
  local operation="$1"           # install, uninstall
  local installedManifest=""     # set to "kdd" or "etcd" if there's a problem.

  # Check if we're installing one datastore type but there's a manifest
  # for the "opposite" datastore type laying around the current directory.

  if [ "$DATASTORE" == "etcdv3" ]; then
    if [ -f "cnx-kdd.yaml" ]; then
      installedManifest="kdd"
      datastoreFlag="kubernetes"
    fi
  elif [ -f "cnx-etcd.yaml" ]; then
      installedManifest="etcd"
      datastoreFlag="etcdv3"
  fi

  if [ "$installedManifest" ]; then
    echo
    echo "Warning: the current $operation specifies \"$DATASTORE\", however \"cnx-$installedManifest.yaml\" exists"
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
# applyCNXManifest()
#
applyCNXManifest() {
  if [ "$DATASTORE" == "kubernetes" ]; then
    echo -n "Applying \"cnx-kdd.yaml\" manifest: "
    run kubectl apply -f cnx-kdd.yaml
  elif [ "$DATASTORE" == "etcdv3" ]; then
    echo -n "Applying \"cnx-etcd.yaml\" manifest: "
    run kubectl apply -f cnx-etcd.yaml
  fi
  blockUntilPodIsReady "k8s-app=cnx-apiserver" 180 "cnx-apiserver"  # Block until cnx-apiserver pod is running & ready
  blockUntilPodIsReady "k8s-app=cnx-manager" 180 "cnx-manager"      # Block until cnx-manager pod is running & ready
  if [ "$DEPLOYMENT_TYPE" == "federation" ]; then
    blockUntilPodIsReady "k8s-app=tigera-federation-controller" 180 "tigera-federation-controller"  # Block until tigera-federation-controller pod is running & ready
  fi
  countDownSecs 10 "Waiting for cnx-apiserver to stabilize"         # Wait until cnx-apiserver completes registration w/kube-apiserver
}

#
# deleteCNXManifest()
#
deleteCNXManifest() {
  if [ "$DATASTORE" == "kubernetes" ]; then
    runIgnoreErrors "kubectl delete -f cnx-kdd.yaml"
    countDownSecs 30 "Deleting \"cnx-kdd.yaml\" manifest"
  elif [ "$DATASTORE" == "etcdv3" ]; then
    runIgnoreErrors "kubectl delete -f cnx-etcd.yaml"
    countDownSecs 30 "Deleting \"cnx-etcd.yaml\" manifest"
  fi
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
# doesCRDExist() - return 1 exit code if the CRD exists
#
doesCRDExist() {
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
  elasticCRD="elasticsearchclusters.enterprises.upmc.com"

  echo -n "waiting for Custom Resource Definitions to be created: "

  count=30
  while [[ $count -ne 0 ]]; do
    if (doesCRDExist $alertCRD) && (doesCRDExist $promCRD) && (doesCRDExist $svcCRD) && (doesCRDExist $elasticCRD); then
        echo "all CRDs exist!"
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
  blockUntilPodIsReady "name=es-client-tigera-elasticsearch" 180 "elasticsearch-client"      # Block until elasticsearch-client pod is running & ready
  blockUntilPodIsReady "name=kibana-tigera-elasticsearch" 180 "kibana"    # Block until kibana pod is running & ready
  run kubectl apply -f kibana-dashboards.yaml
}

#
# deleteMonitorCalicoManifest()
#
deleteMonitorCalicoManifest() {
  runIgnoreErrors kubectl delete -f kibana-dashboards.yaml
  runIgnoreErrors kubectl delete -f monitor-calico.yaml
  countDownSecs 5 "Deleting \"monitor-calico.yaml\" manifest"
}

#
# createCNXManagerSecret()
#
createCNXManagerSecret() {

  if [ "$INSTALL_TYPE" == "KUBEADM" ]; then
    local API_SERVER_CRT="/etc/kubernetes/pki/apiserver.crt"
    local API_SERVER_KEY="/etc/kubernetes/pki/apiserver.key"
  elif [ "$INSTALL_TYPE" == "KOPS" ]; then
    local API_SERVER_CRT="/srv/kubernetes/server.cert"
    local API_SERVER_KEY="/srv/kubernetes/server.key"
  elif [ "$INSTALL_TYPE" == "ACS-ENGINE" ]; then
    local API_SERVER_CRT="/etc/kubernetes/certs/apiserver.crt"
    local API_SERVER_KEY="/etc/kubernetes/certs/apiserver.key"
  fi

  runAsRoot kubectl create secret generic cnx-manager-tls --from-file=cert="$API_SERVER_CRT" --from-file=key="$API_SERVER_KEY" -n kube-system
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
  checkSettings install           # Verify settings are correct with user
  validateDatastore install       # Warn if there's etcd manifest, but we're doing kdd install (and vice versa)

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
  applyCalicoManifest             # Apply calico.yaml

  applyLicenseManifest            # If the user specified a license file, apply it

  removeMasterTaints              # Remove master taints
  createCNXManagerSecret          # Create cnx-manager-tls to enable manager/apiserver communication
  applyCNXManifest                # Apply cnx-[etcd|kdd].yaml

  if [ "${SKIP_MONITORING}" -eq 0 ]; then
    applyCNXPolicyManifest        # Apply cnx-policy.yaml
    applyOperatorManifest         # Apply operator.yaml
    applyElasticStorageManifest   # Apply elastic-storage.yaml
    applyMonitorCalicoManifest    # Apply monitor-calico.yaml
  fi

  reportSuccess                   # Tell user how to login
}

#
# uninstallCNX() - remove Tigera Secure EE and related changes.
# Ignore errors.
#
uninstallCNX() {
  checkSettings uninstall        # Verify settings are correct with user
  validateDatastore uninstall    # Warn if there's etcd manifest, but we're doing kdd uninstall (and vice versa)
  deleteCalicoBinaries           # delete /etc/calico/calicoct.cfg, calicoctl, and calicoq

  downloadManifests              # Download all manifests

  if [ "${SKIP_MONITORING}" -eq 0 ]; then
    deleteMonitorCalicoManifest  # Delete monitor-calico.yaml
    deleteElasticStorageManifest # Delete elastic-storage.yaml
    deleteOperatorManifest       # Delete operator.yaml
    deleteCNXPolicyManifest      # Delete cnx-policy.yaml
  fi

  deleteCNXManifest              # Delete cnx-[etcd|kdd].yaml
  deleteCalicoManifest           # Delete calico.yaml
  deleteRbacManifest             # Delete rbac.yaml
  deleteEtcdDeployment           # Delete etcd.yaml (etcd datatstore only)
  deleteCNXManagerSecret         # Delete TLS secret
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
checkRequirementsInstalled      # Validate that all required programs are installed

determineInstallerType          # Set INSTALL_TYPE to "KOPS" or "KUBEADM" or "ACS-ENGINE"
parseOptions "$@"               # Set up variables based on args
validateSettings                # Check for missing files/dirs

if [ "$CLEANUP" -eq 1 ]; then
  uninstallCNX                  # Remove Tigera Secure EE, cleanup related files
else
  installCNX                    # Install Tigera Secure EE
fi
