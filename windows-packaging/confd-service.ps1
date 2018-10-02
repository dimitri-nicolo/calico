# Datastore configuration.

# Only the Kubernetes API Datastore is supported in this release.
$env:CALICO_DATASTORE_TYPE = "kubernetes"
# Set KUBECONFIG to the path of your kubeconfig file.
$env:KUBECONFIG = "C:\k\config"

# Node configuration.

# The NODENAME variable must be set to match the Kubernetes Node name of this host.
$env:NODENAME = $(hostname).ToLower()

# Logging.

# confd logs to screen at info level by default.  Uncomment this line to override the log
# level.
$env:BGP_LOGSEVERITYSCREEN = "debug"

# Run the tigera-confd binary.
& $PSScriptRoot\tigera-confd.exe -confdir="$PSScriptRoot"
