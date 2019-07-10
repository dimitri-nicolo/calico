{% assign cli = "kubectl" %}

In order to direct the flow/audit logs to syslog(along with elastic search), the following configuration is needed:

1.  Create a configmap containing the following information.
    ```
    {{cli}} create configmap tigera-syslog-archiving \
    --from-literal=audit-logs="true" \
    --from-literal=flow-logs="true" \
    --from-literal=host=<hostname or IP of syslog desintation host> \
    --from-literal=port=<destination port> \
    --from-literal=protocol=<udp or tcp> \
    --from-literal=flush-interval=<interval> \
    -n calico-monitoring
    ```

    > **Note**: `flush-interval` should be specified as `10s` for 10 seconds.
    {: .alert .alert-info}

1. Force a rolling update of fluentd by patching the DaemonSet.
   ```bash
   {{cli}} patch daemonset -n calico-monitoring tigera-fluentd-node -p \
     "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"update-date\":\"`date +'%s'`\"}}}}}"
   ```
