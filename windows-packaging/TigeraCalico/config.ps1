# KUBE_NETWORK should be set to a regular expression that matches the HNS networks used for pods.
# The default, "Calico.*", is correct for Calico CNI.  For flannel, the netowrk is typically called "cbr0".
$env:KUBE_NETWORK = "Calico.*"

# For Calico BGP networking, set this to "windows-bgp"; for flannel, use "none".
$env:CALICO_NETWORKING_BACKEND="windows-bgp"

# Place to install the CNI plugin to.  Should match kubelet's --cni-bin-dir.
$env:CNI_BIN_DIR = "c:\k\cni"
# Place to install the CNI config to.  Should be located in kubelet's --cni-conf-dir.
$env:CNI_CONF_DIR = "c:\k\cni\config"
$env:CNI_CONF_FILENAME = "10-calico.conf"
# Set to match your Kubernetes service CIDR.
$env:K8s_SERVICE_CIDR = "10.96.0.0/12"

# Datastore configuration:

# Set this to "kubernetes" to use the kubernetes datastore, or "etcdv3" for etcd.
$env:CALICO_DATASTORE_TYPE = "<your datastore type>"

# Set KUBECONFIG to the path of your kubeconfig file.
$env:KUBECONFIG = "c:\k\config"

# For the "etcdv3" datastore, set ETCD_ENDPOINTS, format: "http://<host>:<port>,..."
$env:ETCD_ENDPOINTS = "<your etcd endpoints>"
# For etcd over TLS, set these lines to point to your keys/certs:
$env:ETCD_KEY_FILE = ""
$env:ETCD_CERT_FILE = ""
$env:ETCD_CA_CERT_FILE = ""

# Node configuration.

# The NODENAME variable should be set to match the Kubernetes Node name of this host.
# The default uses this node's hostname (which is the same as kubelet).
$env:NODENAME = $(hostname).ToLower()

# BGP configuration.

# The IP of the node; the default will auto-detect a usable IP in most cases.
$env:IP = "autodetect"

# Logging.

$env:CALICO_LOG_DIR = "$PSScriptRoot\logs"

# Felix logs to screen at info level by default.  Uncomment this line to override the log
# level.  Alternatively, (if this is commented out) the log level can be controlled via
# the FelixConfiguration resource in the datastore.
# $env:FELIX_LOGSEVERITYSCREEN = "info"
# Disable logging to file by default since the service wrapper will redirect our log to file.
$env:FELIX_LOGSEVERITYFILE = "none"
# Disable syslog logging, which is not supported on Windows.
$env:FELIX_LOGSEVERITYSYS = "none"

# confd logs to screen at info level by default.  Uncomment this line to override the log
# level.
#$env:BGP_LOGSEVERITYSCREEN = "debug"
