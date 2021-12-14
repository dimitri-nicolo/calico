# Windows-specific configuration.

# KUBE_NETWORK should be set to the name of the HNS network that pods are attached to.
$env:KUBE_NETWORK = "l2bridge"

# Datastore configuration.

# Only the Kubernetes API Datastore is supported in this release.
$env:CALICO_DATASTORE_TYPE = "kubernetes"
# Set KUBECONFIG to the path of your kubeconfig file.
$env:KUBECONFIG = "<path-to-kubeconfig>"

# Node configuration.

# The FELIXHOSTNAME variable should be set to match the Kubernetes Node name of this host.
# The default will use this node's hostname.
$env:FELIX_FELIXHOSTNAME = $(hostname).ToLower()

# Logging.

# Felix logs to screen at info level by default.  Uncomment this line to override the log
# level.  Alternatively, the log level can be controlled via the FelixConfiguration
# resource in the datastore.
# $env:FELIX_LOGSEVERITYSCREEN = "info"
# Disable logging to file by default since the service wrapper will redirect our log to file.
$env:FELIX_LOGSEVERITYFILE = "none"
# Disable syslog logging, which is not supported on Windows.
$env:FELIX_LOGSEVERITYSYS = "none"

# Disable OpenStack metadata server support, which is not available on Windows.
$env:FELIX_METADATAADDR = "none"

# Run the calico-felix binary.
.\tigera-felix.exe
