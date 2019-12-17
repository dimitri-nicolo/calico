---
title: Enabling integration with AWS security groups (optional)
redirect_from: latest/reference/other-install-methods/kubernetes/installation/aws-sg-integration
canonical_url: https://docs.tigera.io/v2.5/getting-started/kubernetes/installation/aws-sg-integration
---


> Note: AWS security group integration is currently only supported when using the manual and helm installation paths.
{: .alert .alert-info}

AWS security group integration for {{site.prodname}} allows you to combine AWS security groups with network policy to enforce granular access control between Kubernetes pods and AWS VPC resources. See the [AWS security group integration overview](/{{page.version}}/security/aws-security-group-integration/) for more details on how to configure security groups in your VPC.

### Requirements

Your Kubernetes cluster must meet the following specifications:

- Exists within a single VPC.
- The Kubernetes AWS cloud provider is enabled.
- Networking provider is
  [Amazon VPC Networking]({{site.url}}/{{page.version}}/reference/public-cloud/aws#using-aws-networking).
  (You must be using the [AWS CNI Plugin](https://github.com/aws/amazon-vpc-cni-k8s).)
- You have already installed
  [Tigera Secure EE for policy]({{site.url}}/{{page.version}}/reference/other-install-methods/kubernetes/installation/other#installing-tigera-secure-ee-for-policy-only)
  on your cluster. The AWS security group integration requires the Kubernetes API datastore.
- You have not created any
  [host endpoints]({{site.url}}/{{page.version}}/reference/resources/hostendpoint)
  that have a `spec.node` value that matches any of your Kubernetes nodes. See the [AWS security group integration guide](/{{page.version}}/security/aws-security-group-integration/host-endpoints) for more information.


You will need a host equipped with the following:

 - `kubectl`: configured to access the cluster.
 - AWS Command Line Interface (CLI): The following commands are known to work well with AWS CLI 1.15.40
 - [jq](https://stedolan.github.io/jq/)

### Before you begin

Collect the following information about your cluster and export it as environment variables:

| Variable | Description |
|---|---|
| `AWS_REGION` | The region in which your cluster is located. The AWS CLI commands in this install assume this environment variable has been set. |
| `VPC_ID` | The ID of the VPC in which your cluster is located. |
| `CONTROL_PLANE_SG` | The ID of a security group to which all master nodes belong. |
| `K8S_NODE_SGS` | Group ID(s) of the security group(s) that each node should belong to. If more than one, use commas to separate them. |
| `CLUSTER_NAME` | A name that uniquely identifies this cluster. This is used as a prefix when naming per-cluser resources. It must satisfy the pattern `[a-zA-Z][-a-zA-Z0-9]*`. |

Most Kubernetes provisioners will set a different security group for masters and nodes. If your cluster uses the same security group
across both, it is OK to set `$CONTROL_PLANE_SG` and `$K8S_NODE_SGS` to the same value.

We've provided info below on how to gather the above info in common Kubernetes environments on AWS.

- **EKS cluster created with eksctl**

   The following commands gather the necessary information of a particular EKS
   cluster with name `$CLUSTER_NAME` that was created with [`eksctl`](https://github.com/weaveworks/eksctl){:target="_blank"}:

   ```
   VPC_ID=$(aws eks describe-cluster --name $CLUSTER_NAME --query 'cluster.resourcesVpcConfig.vpcId' --output text)
   K8S_NODE_SGS=$(aws ec2 describe-security-groups --filters Name=tag:aws:cloudformation:logical-id,Values=SG Name=vpc-id,Values=${VPC_ID} --query "SecurityGroups[0].GroupId" --output text)
   CONTROL_PLANE_SG=$(aws ec2 describe-security-groups --filters Name=tag:aws:cloudformation:logical-id,Values=ControlPlaneSecurityGroup Name=vpc-id,Values=${VPC_ID} --query "SecurityGroups[0].GroupId" --output text)
   ```

- **kops cluster**

   The following commands gather the necessary information of a particular kops cluster with name `$CLUSTER_NAME`:

   ```
   VPC_ID=$(aws ec2 describe-instances \
       --filters "Name=tag-value,Values=${CLUSTER_NAME}" \
       --output json \
       | jq -r '.Reservations[].Instances[].VpcId' \
       | grep -v null \
       | head -1)

   CONTROL_PLANE_SG=$(aws ec2 describe-instances \
       --filters "Name=tag-value,Values=${CLUSTER_NAME}" \
       --output json \
       | jq -r '.Reservations[].Instances[].SecurityGroups[] | select(.GroupName |startswith("master")) |.GroupId')

   K8S_NODE_SGS=$(aws ec2 describe-instances \
       --filters "Name=tag-value,Values=${CLUSTER_NAME}" \
       --output json \
       | jq -r '.Reservations[].Instances[].SecurityGroups[] | select(.GroupName |startswith("nodes")) |.GroupId' \
       | head -1)
   ```

### Procedure

1.  Ensure you have satisfied the [requirements](#requirements) above.

    * Verify the Kubernetes AWS cloud provider is enabled by confirming each node has a ProviderId:
       ```bash
       kubectl get node -o=jsonpath='{range .items[*]}{.metadata.name}{"\tProviderId: "}{.spec.providerID}{"\n"}{end}'
       ```

    * Verify the Amazon VPC Networking and CNI plugin is being used by confirming that
       an `aws-node` pod exists on each node:
       ```bash
       kubectl get pod -n kube-system -l k8s-app=aws-node -o wide
       ```

    * Verify TSEE has been installed by confirming that cnx-manager is running:
       ```bash
       kubectl get pod -n calico-monitoring -l k8s-app=cnx-manager
       ```

    * Verify that no Host Endpoints have been created by verifying that no entries are returned by:
       ```bash
       kubectl get hostendpoints
       ```

1.  Install AWS per-account resources.

    The per-account resources must be applied once per AWS account. Use the
    following command to see if `tigera-cloudtrail` has already been applied:

    ```bash
    aws cloudformation describe-stacks --stack-name tigera-cloudtrail
    ```

    If the command output does not output a stack run the following command
    to create the per-account stack:

    ```bash
    aws cloudformation create-stack \
    --stack-name tigera-cloudtrail \
    --template-body {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/aws-sg-integration/account-cf.yaml

    # Wait for the stack to finish provisioning
    aws cloudformation wait stack-create-complete --stack-name tigera-cloudtrail
    ```

1.  Install AWS per-VPC resources.

    The per-VPC CloudFormation must be created once on a VPC that contains (or will contain) clusters.
    Run the following command to see if this VPC has had the per-VPC stack applied:

    ```bash
    aws cloudformation describe-stacks --stack-name tigera-vpc-$VPC_ID
    ```

    If the stack is not found, run the following command to create it:

    ```bash
    aws cloudformation create-stack \
    --stack-name tigera-vpc-$VPC_ID \
    --parameters ParameterKey=VpcId,ParameterValue=$VPC_ID \
    --capabilities CAPABILITY_IAM \
    --template-body {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/aws-sg-integration/vpc-cf.yaml

    # Wait for the stack to finish provisioning
    aws cloudformation wait stack-create-complete --stack-name tigera-vpc-$VPC_ID
    ```

1.  Install AWS per-cluster resources.

    ```bash
    aws cloudformation create-stack \
    --stack-name tigera-cluster-$CLUSTER_NAME \
    --parameters ParameterKey=VpcId,ParameterValue=$VPC_ID \
                 ParameterKey=KubernetesHostDefaultSGId,ParameterValue=$K8S_NODE_SGS \
                 ParameterKey=KubernetesControlPlaneSGId,ParameterValue=$CONTROL_PLANE_SG \
    --template-body {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/aws-sg-integration/cluster-cf.yaml

    # Wait for the stack to finish provisioning
    aws cloudformation wait stack-create-complete --stack-name tigera-cluster-$CLUSTER_NAME
    ```

1.  Add the controller IAM user secrets in Kubernetes.

    ```bash
    # First, get the name of the created IAM user, which is an output field in your Cluster CF stack
    CONTROLLER_USERNAME=$(aws cloudformation describe-stacks \
    --stack-name tigera-vpc-$VPC_ID \
    --output text \
    --query "Stacks[0].Outputs[?OutputKey=='TigeraControllerUserName'][OutputValue]")

    # Then create an access key for that role
    aws iam create-access-key \
    --user-name $CONTROLLER_USERNAME \
    --output text \
    --query "AccessKey.{Key:SecretAccessKey,ID:AccessKeyId}" > controller-secrets.txt

    # Add the key as a k8s secret
    cat controller-secrets.txt | xargs bash -c \
    'kubectl create secret generic tigera-cloud-controllers-credentials \
    -n kube-system \
    --from-literal=aws_access_key_id=$0 \
    --from-literal=aws_secret_access_key=$1'

    # Delete local copy of the secret
    rm -f controller-secrets.txt
    ```

1.  Configure AWS Security Group Integration.

    ```bash
    # Get the SQS URL
    SQS_URL=$(aws cloudformation describe-stacks \
    --stack-name tigera-vpc-$VPC_ID \
    --output text \
    --query "Stacks[0].Outputs[?OutputKey=='QueueURL'][OutputValue]")

    # Get the default pod SG
    POD_SG=$(aws cloudformation describe-stacks \
    --stack-name tigera-cluster-$CLUSTER_NAME \
    --output text \
    --query "Stacks[0].Outputs[?OutputKey=='TigeraDefaultPodSG'][OutputValue]")

    # Get the SG for enforced nodes
    ENFORCED_SG=$(aws cloudformation describe-stacks \
    --stack-name tigera-vpc-$VPC_ID \
    --output text \
    --query "Stacks[0].Outputs[?OutputKey=='TigeraEnforcedSG'][OutputValue]")

    # Get the SG for enforced nodes
    TRUST_SG=$(aws cloudformation describe-stacks \
    --stack-name tigera-vpc-$VPC_ID \
    --output text \
    --query "Stacks[0].Outputs[?OutputKey=='TigeraTrustEnforcedSG'][OutputValue]")

    # Store both in a new configmap
    kubectl create configmap tigera-aws-config \
    -n kube-system \
    --from-literal=aws_region=$AWS_REGION \
    --from-literal=vpcs=$VPC_ID \
    --from-literal=sqs_url=$SQS_URL \
    --from-literal=pod_sg=$POD_SG \
    --from-literal=default_sgs=$K8S_NODE_SGS \
    --from-literal=enforced_sg=$ENFORCED_SG \
    --from-literal=trust_sg=$TRUST_SG
    ```

1. Configure failsafes for Kubernetes API access.

   The `{{site.noderunning}}` DaemonSet should be updated to include the following
   environment variables and values:

   | Variable Name | Value |
   |---|---|
   | `FELIX_FAILSAFEINBOUNDHOSTPORTS` | tcp:22,udp:68,tcp:179,tcp:443,tcp:5473,tcp:6443 |
   | `FELIX_FAILSAFEOUTBOUNDHOSTPORTS` | udp:53,udp:67,tcp:179,tcp:443,tcp:5473,tcp:6443 |

   Use one of the following approaches to add the above environment variables
   to the `{{site.noderunning}}` DaemonSet's environment variables:
   * Edit the manifest that was used to install {{site.noderunning}} then
     reapply the manifest. After editing the file use something like the
     following to reapply the manifest:
     `kubectl apply -f <updated manifest>`
   * Directly update the DaemonSet using the following command:
     `kubectl -n kube-system edit daemonset calico-node`

   > **Note:** See [Configuring Felix]({{site.url}}/{{page.version}}/reference/felix/configuration)
   > for more information on the configuration options.
   {: .alert .alert-info}

1. Restart the {{site.prodname}} components.

   If your cluster does not have production workloads yet feel free to restart
   all the components without concern. If your cluster has production workloads
   active then you should ensure that one pod is restarted at a time and
   it becomes healthy before restarting the next.
   The components that need restarting are:

   * `calico-typha`
   * `cnx-apiserver`
   * `calicoq` (if running as a pod)

1.  Download the Cloud Controller manifest.

    ```bash
    curl \
    {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/aws-sg-integration/cloud-controllers.yaml \
    -O
    ```

{% include {{page.version}}/cnx-cred-sed.md yaml="cloud-controllers" %}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f cloud-controllers.yaml
   ```
