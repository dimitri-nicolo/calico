---
title: Enabling access to AWS metadata
canonical_url: https://docs.tigera.io/v2.3/usage/aws-security-group-integration/metadata-access
---


### About pod access to AWS metadata

By default, {{site.tseeprodname}} blocks pods from accessing the AWS metadata endpoint on their node. Access to the
[AWS metadata](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html)
endpoint allows a pod to obtain
[instance and user metadata](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html#instancedata-data-categories)
including the IAM credentials of the node. You may enable pod access for individual or all pods.


### Enabling individual pods to access the AWS metadata endpoint
The following command allows an individual pod to access the AWS metadata endpoint on its node.

````
kubectl label pods <pod-name> aws.tigera.io/allow-metadata-access=true
````

### Enabling all pods to access the AWS metadata endpoint

If If the number of pods you need to allow exceeds the number that you need to block,
you may find it more convenient to change the default to _allow_ access and then deny access
to individual pods that do not need it.

- Open the Tigera Cloud Controllers deployment up for editing.

  ````
kubectl edit deployment/tigera-cloud-controllers -n=kube-system
````

- In the __spec.template.spec.containers.env__ section, add the environment variable
`ALLOW_POD_METADATA_ACCESS`
set to to __true__.
This will give all pods access to the AWS metadata endpoint.

  ````
env:
  - name: ALLOW_POD_METADATA_ACCESS
    value: "true"
````

- Use the following command to then block specific pods.

  ````
kubectl label pods <pod-name> aws.tigera.io/allow-metadata-access=false
````

You may also make these changes in the `cloud-controller.yaml` manifest prior to applying it to the cluster.
