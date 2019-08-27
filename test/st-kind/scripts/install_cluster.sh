#!/usr/bin/env bash

# Prepare for Voltron st:
chmod -R 777 /var/run/docker.sock
ln -s /usr/local/bin/google-cloud-sdk/bin/gcloud /usr/bin/gcloud
# exec /sbin/su-exec user "$@"
#
alias gcloud=/root/google-cloud-sdk/bin/gcloud

echo "INFO: Checking for the presence of the right credentials"

config_json="config.json"
docker_auth="docker_auth.json"

# Mounted Directory
secrets_directory="/home/runner/config/"

if [ ! -f ${config_json} ];
then
    if [ -f $secrets_directory$config_json ]
    then
        config_json="$secrets_directory$config_json"
    else
        echo "ERROR: We need ${config_json} to be present to run the test suite properly. $(pwd)"
        exit 1
    fi
fi

if [ ! -f ${docker_auth} ];
then
    if [ -f $secrets_directory$docker_auth ]
    then
        docker_auth="$secrets_directory$docker_auth"
    else
        echo "ERROR: We need ${docker_auth} to be present to run the test suite properly. $(pwd)"
        exit 1
    fi
fi

if [[ -z `command -v helm` ]]
then
    echo "ERROR: You should have helm installed on your machine. To install, run: "
    echo "$ curl -O https://get.helm.sh/helm-v2.14.2-linux-amd64.tar.gz"
    echo "$ tar xzvf helm-v2.14.2-linux-amd64.tar.gz"
    echo "$ install linux-amd64/helm ."
    echo "$ helm version"
    exit 1
fi

if [[ -z `command -v kind` ]]
then
    echo "ERROR: You should have kind installed on your machine. To install, run: "
    echo "$ go get sigs.k8s.io/kind"
    exit 1
fi

###########################
# Create the kind cluster #
###########################

echo "INFO: Creating a kind cluster to setup st tests."
kind create cluster

# Setup kubectl
export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"

kubectl create namespace calico-monitoring

# Make sure you can pull images and that your settings go back to normal if the script fails to finish
currentaccount=`gcloud config get-value account`
if [[ -n ${currentaccount} ]]
then
    trap 'gcloud config set account ${currentaccount}' EXIT
fi

# If your account has no access, use these lines after authenticating using the service account.
# echo "INFO: Temporarily setting your gcloud identity"
gcloud auth activate-service-account --key-file "$docker_auth"
gcloud auth configure-docker

# In order for this to work `make images` should have already run. We now add a new docker tag, so kind can use it.
# Using a `latest` tag, will make calls to external hub, regardless of the pull policy.
docker tag $(docker image ls tigera/voltron:latest -q) tigera/voltron:st-image
docker tag $(docker image ls tigera/guardian:latest -q) tigera/guardian:st-image

# Images downloaded through Makefile
echo "INFO: Loading the private images into kind."
kind load docker-image gcr.io/unique-caldron-775/cnx/tigera/cnx-node:master
kind load docker-image gcr.io/unique-caldron-775/cnx/tigera/cnx-apiserver:master
kind load docker-image gcr.io/unique-caldron-775/cnx/tigera/cnx-queryserver:master
kind load docker-image gcr.io/unique-caldron-775/cnx/tigera/kube-controllers:master
kind load docker-image tigera/voltron:st-image
kind load docker-image tigera/guardian:st-image

###########################
# Install EE core         #
###########################

# Needs to align with check-prerequisites.sh
echo "INFO: Install helm and tsee core (which includes the cnx-apiserver)"
kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
kubectl create serviceaccount --namespace kube-system tiller
helm init --net-host --service-account tiller --override "spec.template.spec.tolerations[0].effect=NoSchedule" --override "spec.template.spec.tolerations[0].key=node.kubernetes.io/not-ready"  --override "spec.template.spec.tolerations[0].operator=Exists"
helm plugin install https://github.com/viglesiasce/helm-gcs.git
helm repo add tigera gs://tigera-helm-charts
helm repo update tigera
#Add a sleep, as the tiller deployment rollout command can time out from time to time, making the script fail
sleep 8s
kubectl rollout status deployment/tiller-deploy -n kube-system
kubectl rollout status deployment/coredns -n kube-system
helm install tigera/tigera-secure-ee-core --set-file imagePullSecrets.cnx-pull-secret=${config_json}
kubectl rollout status deployment/cnx-apiserver -n kube-system
kubectl rollout status deployment/calico-kube-controllers -n kube-system


###########################
# Install Voltron         #
###########################

# Bind cluster role binding to the authenticated users so we can make requests to new clusters.
kubectl create clusterrolebinding mcm-binding-user \
 --clusterrole=tigera-mcm-no-crd --group=system:authenticated

echo "INFO: Add tiers and network policies"
kubectl apply -f https://docs.tigera.io/v2.5/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml

echo "INFO: Create Voltron keys"
mkdir keyfiles
../../scripts/certs/clean-self-signed.sh keyfiles
../../scripts/certs/self-signed.sh keyfiles

echo "INFO: Install voltron"
helm install tigera/tigera-secure-ee-mcm --set-file certs.provided.crt=keyfiles/voltron.crt --set-file certs.provided.key=keyfiles/voltron.key --set image.pullPolicy=Never --set image.tag=st-image --set image.repository=tigera/voltron --namespace calico-monitoring
kubectl rollout status deployment/cnx-voltron -n calico-monitoring
mkdir -p test-resources

echo "INFO: Clean up Voltron keys"
rm -R keyfiles

# Open a socket so we can curl it.
kubectl port-forward -n calico-monitoring svc/cnx-voltron-server 9443&
