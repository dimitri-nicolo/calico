---
title: Enabling integration with AWS security groups (optional)
canonical_url: https://docs.tigera.io/master/getting-started/kubernetes/installation/aws-sg-integration
---

### Requirements

Your Kubernetes cluster must meet the following specifications:

- Exists in a single VPC.
- Has the Kubernetes AWS cloud provider enabled.
- Networking provider is [Amazon VPC Networking]({{site.url}}/{{page.version}}/reference/public-cloud/aws#using-aws-networking).
- You have already installed
  [Tigera Secure EE for policy]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/other#installing-tigera-secure-ee-for-policy-only)
  on your cluster.

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

- **EKS cluster**

   The following commands gather the necessary information of a particular EKS cluster with name `$CLUSTER_NAME`:

   ```
   VPC_ID=$(aws eks describe-cluster --name $CLUSTER_NAME --query 'cluster.resourcesVpcConfig.vpcId' --output text)
   K8S_NODE_SGS=$(aws ec2 describe-security-groups --filters Name=tag:aws:cloudformation:logical-id,Values=NodeSecurityGroup Name=vpc-id,Values=${VPC_ID} --query "SecurityGroups[0].GroupId" --output text)
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

#### Procedure

1.  **Install AWS per-account resources.**

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

1.  **Install AWS per-VPC resources.**

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

1.  **Install AWS per-cluster resources.**

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

1.  **Add the controller IAM user secrets in Kubernetes.**

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

1.  **Configure AWS Security Group Integration.**

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

1.  **Install Kubernetes components.**

    ```bash
    kubectl apply -f {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/manifests/aws-sg-integration/cloud-controller.yaml
    ```

1. **Update {{site.noderunning}}, {{site.prodname}} API server, and {{site.prodname}} queryserver.**
   Edit the deployments for {{site.noderunning}} and cnx-apiserver and add the following
   environment variables to the {{site.noderunning}}, cnx-apiserver, and cnx-queryserver
   containers.

   | Key | Value (from specified environment variable) |
   |---|---|
   | `TIGERA_ENFORCED_GROUP_ID` | $TRUST_SG |
   | `TIGERA_TRUST_ENFORCED_GROUP_ID` | $ENFORCED_SG |

   **Note:** This step should be removed when components are updated to read
   these values from the datastore.

