# Datastore configuration.

# Datastore type can be either "etcdv3" or "kubernetes"
$env:CALICO_DATASTORE_TYPE = "etcdv3"

If ($env:CALICO_DATASTORE_TYPE -eq "kubernetes"){
  # Set KUBECONFIG to the path of your kubeconfig file.
  $env:KUBECONFIG = "C:\k\config"
} elseif ($env:CALICO_DATASTORE_TYPE -eq "etcdv3"){
  # Set CalicoConfig to the path of your calico config file.
  $CalicoConfig = "$PSScriptRoot\calicocfg"
} else {
  Write-Error 'Invalid datastore type' -ErrorAction Stop
}

# Logging.

# confd logs to screen at info level by default.  Uncomment this line to override the log
# level.
$env:BGP_LOGSEVERITYSCREEN = "debug"

# Node configuration.

# The NODENAME variable must be set to match the Kubernetes Node name of this host.
$env:NODENAME = $(hostname).ToLower()

# Run the tigera-confd binary.
If ($env:CALICO_DATASTORE_TYPE -eq "kubernetes"){
  & $PSScriptRoot\tigera-confd.exe -confdir="$PSScriptRoot"
} elseif ($env:CALICO_DATASTORE_TYPE -eq "etcdv3"){
  & $PSScriptRoot\tigera-confd.exe -confdir="$PSScriptRoot" -calicoconfig="$CalicoConfig"
}
