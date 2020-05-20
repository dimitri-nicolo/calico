---
title: Enabling integration with AWS security groups (optional)
description: Calico Enterprise lets you combine AWS security groups with network policy to enforce access control between Kubernetes pods and AWS VPC resources. 
---

> **Warning!** AWS security group integration is a tech preview feature.
{: .alert .alert-danger }

### Big picture

Enable {{ site.prodname }} integration with AWS Security Groups.

### Value

AWS security group integration for {{site.prodname}} allows you to combine AWS security groups with network policy to enforce granular access control between Kubernetes pods and AWS VPC resources.

### Before you begin...

Your Kubernetes cluster must meet the following specifications:

- Exists within a single VPC.
- The Kubernetes AWS cloud provider is enabled.

  Verify the Kubernetes AWS cloud provider is enabled by confirming each node has a ProviderId:

  ```bash
  kubectl get node -o=jsonpath='{range .items[*]}{.metadata.name}{"\tProviderId: "}{.spec.providerID}{"\n"}{end}'
  ```

- Networking provider is
  [Amazon VPC Networking]({{site.baseurl}}/reference/public-cloud/aws#using-aws-networking).
  (You must be using the [AWS CNI Plugin](https://github.com/aws/amazon-vpc-cni-k8s).)

  Verify the Amazon VPC Networking and CNI plugin is being used by confirming that an `aws-node` pod exists on each node:

  ```bash
  kubectl get pod -n kube-system -l k8s-app=aws-node -o wide
  ```

- You have already installed
  [Calico Enterprise for EKS]({{site.baseurl}}/getting-started/kubernetes/managed-public-cloud/eks)
  on your cluster. The AWS security group integration requires the Kubernetes API datastore.

  Verify Calico Enterprise has been installed by confirming that all tigerastatuses are available:

  ```bash
  kubectl get tigerastatus
  ```

- You are not using the [auto hostendpoints feature]({{ site.baseurl }}/security/kubernetes-nodes#enable-automatic-host-endpoints), nor have you created any
  [host endpoints]({{site.baseurl}}/reference/resources/hostendpoint)
  that have a `spec.node` value that matches any of your Kubernetes nodes.

  You can verify that no Host Endpoints have been created by verifying that no entries are returned by:

  ```bash
  kubectl get hostendpoints
  ```

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
| `CLUSTER_NAME` | A name that uniquely identifies this cluster within the VPC. This is used as a prefix when naming per-cluster resources. It must satisfy the pattern `[a-zA-Z][-a-zA-Z0-9]*`. |

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

  > Note: Commands above only applies to EKS cluster with unmanaged nodegroups i.e. [eksctl without --managed](https://eksctl.io/usage/eks-managed-nodes/) option.
  {: .alert .alert-info}

- **kops cluster**

   The following commands gather the necessary information of a particular kops cluster with name `$KOPS_CLUSTER_NAME`:

   ```
   VPC_ID=$(aws ec2 describe-instances \
       --filters "Name=tag-value,Values=${KOPS_CLUSTER_NAME}" \
       --output json \
       | jq -r '.Reservations[].Instances[].VpcId' \
       | grep -v null \
       | head -1)

   CONTROL_PLANE_SG=$(aws ec2 describe-instances \
       --filters "Name=tag-value,Values=${KOPS_CLUSTER_NAME}" \
       --output json \
       | jq -r '.Reservations[].Instances[].SecurityGroups[] | select(.GroupName |startswith("master")) |.GroupId')

   K8S_NODE_SGS=$(aws ec2 describe-instances \
       --filters "Name=tag-value,Values=${KOPS_CLUSTER_NAME}" \
       --output json \
       | jq -r '.Reservations[].Instances[].SecurityGroups[] | select(.GroupName |startswith("nodes")) |.GroupId' \
       | head -1)
   ```

   >Note: Since `KOPS_CLUSTER_NAMES` are FQDNs, you will need to pick a `CLUSTER_NAME` which does not contain any dot separators for use in the remainder of this guide. See [before you begin](#before-you-begin) for more information.{: .alert .alert-warn}

### Procedure

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
    --template-body {{ "/manifests/aws/security-group-integration/account-cf.yaml" | absolute_url }}

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
    --template-body {{ "/manifests/aws/security-group-integration/vpc-cf.yaml" | absolute_url }}

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
    --template-body {{ "/manifests/aws/security-group-integration/cluster-cf.yaml" | absolute_url }}

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
    cat controller-secrets.txt | tr -d '\n' | xargs bash -c \
    'kubectl create secret generic amazon-cloud-integration-credentials \
    -n tigera-operator \
    --from-literal=key-id=$0 \
    --from-literal=key-secret=$1'

    # Delete local copy of the secret
    rm -f controller-secrets.txt
    ```

1.  Gather the remaining bits of information:

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
    ```

1. Create the operator custom resource

    ```yaml
    apiVersion: operator.tigera.io/v1beta1
    kind: AmazonCloudIntegration
    metadata:
      name: tigera-secure
    spec:
      nodeSecurityGroupIDs:
      - $K8S_NODE_SGS
      podSecurityGroupID: $POD_SG
      vpcs:
        - $VPC_ID
      sqsURL: $SQS_URL
      awsRegion: $AWS_REGION
      enforcedSecurityGroupID: $ENFORCED_SG
      trustEnforcedSecurityGroupID: $TRUST_SG
    ```
