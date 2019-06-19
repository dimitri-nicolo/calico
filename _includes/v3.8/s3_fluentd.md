{% assign cli = "kubectl" %}

In order to redirect the flow/audit logs to Amazon S3 cloud object storage(along with elastic search), the following configuration is needed:

1. Create an AWS bucket that will store your logs, noting the bucket name, region, key,
   secret key, and choose the path you wish to write logs to in the bucket.

1.  Create a configmap containing the following information.
    ```
    {{cli}} create configmap tigera-s3-archiving \
    --from-literal=s3.storage="true" \
    --from-literal=s3.bucket.name=<S3-bucket-name> \
    --from-literal=aws.region=<S3-bucket region> \
    --from-literal=s3.bucket.path=<path-in-S3-bucket> \
    --from-literal=s3.flush-interval="30" \
    -n calico-monitoring
    ```

    > **Note**: `s3.flush-interval` is in seconds.
    {: .alert .alert-info}

    The [fluentd documentation for the S3 output plugin](https://docs.fluentd.org/output/s3#parameters) has more information on these options.

1.  Set up secret with AWS key ID and Secret key for authentication.
    ```
    {{cli}} create secret generic tigera-s3-archiving \
    --from-literal=aws.key.id=<AWS-access-key-id> \
    --from-literal=aws.secret.key=<AWS-secret-key> \
    -n calico-monitoring
    ```

1. Force a rolling update of fluentd by patching the DaemonSet.
   ```bash
   {{cli}} patch daemonset -n calico-monitoring tigera-fluentd-node -p \
     "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"update-date\":\"`date +'%s'`\"}}}}}"
   ```
