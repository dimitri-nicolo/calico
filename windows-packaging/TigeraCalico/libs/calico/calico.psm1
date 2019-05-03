# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# We require the 64-bit version of Powershell, which should live at the following path.
$powerShellPath = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
$baseDir = "$PSScriptRoot\..\.."
$NSSMPath = "$baseDir\nssm-2.24\win64\nssm.exe"

function fileIsMissing($path)
{
    return (("$path" -EQ "") -OR (-NOT(Test-Path "$path")))
}

function Test-CalicoConfiguration()
{
    Write-Host "Validating configuration..."
    if (!$env:CNI_BIN_DIR)
    {
        throw "Config not loaded?."
    }
    if ($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp" -OR $env:CALICO_NETWORKING_BACKEND -EQ "vxlan") {
        if (fileIsMissing($env:CNI_BIN_DIR))
        {
            throw "CNI binary directory $env:CNI_BIN_DIR doesn't exist.  Please create it and ensure kubelet " +  `
                    "is configured with matching --cni-bin-dir."
        }
        if (fileIsMissing($env:CNI_CONF_DIR))
        {
            throw "CNI config directory $env:CNI_CONF_DIR doesn't exist.  Please create it and ensure kubelet " +  `
                    "is configured with matching --cni-conf-dir."
        }
    }
    if ($env:CALICO_NETWORKING_BACKEND -EQ "vxlan" -AND $env:CNI_IPAM_TYPE -NE "calico-ipam") {
        throw "Calico VXLAN requires IPAM type calico-ipam, not $env:CNI_IPAM_TYPE."
    }
    if ($env:CALICO_DATASTORE_TYPE -EQ "kubernetes")
    {
        if (fileIsMissing($env:KUBECONFIG))
        {
            throw "kubeconfig file $env:KUBECONFIG doesn't exist.  Please update the configuration to match. " +  `
                    "the location of your kubeconfig file."
        }
    }
    elseif ($env:CALICO_DATASTORE_TYPE -EQ "etcdv3")
    {
        if (("$env:ETCD_ENDPOINTS" -EQ "") -OR ("$env:ETCD_ENDPOINTS" -EQ "<your etcd endpoints>"))
        {
            throw "Etcd endpoint not set, please update the configuration."
        }
        if (("$env:ETCD_KEY_FILE" -NE "") -OR ("$env:ETCD_CERT_FILE" -NE "") -OR ("$env:ETCD_CA_CERT_FILE" -NE ""))
        {
            if (fileIsMissing($env:ETCD_KEY_FILE))
            {
                throw "Some etcd TLS parameters are configured but etcd key file was not found."
            }
            if (fileIsMissing($env:ETCD_CERT_FILE))
            {
                throw "Some etcd TLS parameters are configured but etcd certificate file was not found."
            }
            if (fileIsMissing($env:ETCD_CA_CERT_FILE))
            {
                throw "Some etcd TLS parameters are configured but etcd CA certificate file was not found."
            }
        }
    }
    else
    {
        throw "Please set datastore type to 'etcdv3' or 'kubernetes'; current value: $env:CALICO_DATASTORE_TYPE."
    }
}

function Install-CNIPlugin()
{
    Write-Host "Copying CNI binaries into place."
    cp "$baseDir\cni\*.exe" "$env:CNI_BIN_DIR"

    $cniConfFile = $env:CNI_CONF_DIR + "\" + $env:CNI_CONF_FILENAME
    Write-Host "Writing CNI configuration to $cniConfFile."
    $nodeNameFile = "$baseDir\nodename".replace('\', '\\')
    $etcdKeyFile = "$env:ETCD_KEY_FILE".replace('\', '\\')
    $etcdCertFile = "$env:ETCD_CERT_FILE".replace('\', '\\')
    $etcdCACertFile = "$env:ETCD_CA_CERT_FILE".replace('\', '\\')
    $kubeconfigFile = "$env:KUBECONFIG".replace('\', '\\')
    $mode = ""
    if ($env:CALICO_NETWORKING_BACKEND -EQ "vxlan") {
        $mode = "vxlan"
    }

    (Get-Content "$baseDir\cni.conf.template") | ForEach-Object {
        $_.replace('__NODENAME_FILE__', $nodeNameFile).
                replace('__KUBECONFIG__', $kubeconfigFile).
                replace('__K8S_SERVICE_CIDR__', $env:K8S_SERVICE_CIDR).
                replace('__DATASTORE_TYPE__', $env:CALICO_DATASTORE_TYPE).
                replace('__ETCD_ENDPOINTS__', $env:ETCD_ENDPOINTS).
                replace('__ETCD_KEY_FILE__', $etcdKeyFile).
                replace('__ETCD_CERT_FILE__', $etcdCertFile).
                replace('__ETCD_CA_CERT_FILE__', $etcdCACertFile).
                replace('__IPAM_TYPE__', $env:CNI_IPAM_TYPE).
                replace('__MODE__', $mode).
                replace('__VNI__', $env:VXLAN_VNI).
                replace('__MAC_PREFIX__', $env:VXLAN_MAC_PREFIX)
    } | Set-Content "$cniConfFile"
    Write-Host "Wrote CNI configuration."
}

function Remove-CNIPlugin()
{
    $cniConfFile = $env:CNI_CONF_DIR + "\" + $env:CNI_CONF_FILENAME
    Write-Host "Removing $cniConfFile and Calico binaries."
    rm $cniConfFile
    rm "$env:CNI_BIN_DIR/calico*.exe"
}

function Install-NodeService()
{
    Write-Host "Installing node startup service..."

    ensureRegistryKey

    # Ensure our service file can run.
    Unblock-File $baseDir\node\node-service.ps1

    & $NSSMPath install TigeraNode $powerShellPath
    & $NSSMPath set TigeraNode AppParameters $baseDir\node\node-service.ps1
    & $NSSMPath set TigeraNode AppDirectory $baseDir
    & $NSSMPath set TigeraNode DisplayName "Tigera Calico Startup"
    & $NSSMPath set TigeraNode Description "Tigera Calico Startup, configures Calico datamodel resources for this node."

    # Configure it to auto-start by default.
    & $NSSMPath set TigeraNode Start SERVICE_AUTO_START
    & $NSSMPath set TigeraNode ObjectName LocalSystem
    & $NSSMPath set TigeraNode Type SERVICE_WIN32_OWN_PROCESS

    # Throttle process restarts if Felix restarts in under 1500ms.
    & $NSSMPath set TigeraNode AppThrottle 1500

    # Create the log directory if needed.
    if (-Not(Test-Path "$env:CALICO_LOG_DIR"))
    {
        write "Creating log directory."
        md -Path "$env:CALICO_LOG_DIR"
    }
    & $NSSMPath set TigeraNode AppStdout $env:CALICO_LOG_DIR\tigera-node.log
    & $NSSMPath set TigeraNode AppStderr $env:CALICO_LOG_DIR\tigera-node.err.log

    # Configure online file rotation.
    & $NSSMPath set TigeraNode AppRotateFiles 1
    & $NSSMPath set TigeraNode AppRotateOnline 1
    # Rotate once per day.
    & $NSSMPath set TigeraNode AppRotateSeconds 86400
    # Rotate after 10MB.
    & $NSSMPath set TigeraNode AppRotateBytes 10485760

    Write-Host "Done installing startup service."
}

function Remove-NodeService()
{
    & $NSSMPath remove TigeraNode confirm
}

function Install-FelixService()
{
    Write-Host "Installing Felix service..."

    # Ensure our service file can run.
    Unblock-File $baseDir\felix\felix-service.ps1

    # We run Felix via a wrapper script to make it easier to update env vars.
    & $NSSMPath install TigeraFelix $powerShellPath
    & $NSSMPath set TigeraFelix AppParameters $baseDir\felix\felix-service.ps1
    & $NSSMPath set TigeraFelix AppDirectory $baseDir
    & $NSSMPath set TigeraFelix DependOnService "TigeraNode"
    & $NSSMPath set TigeraFelix DisplayName "Tigera Calico Agent"
    & $NSSMPath set TigeraFelix Description "Tigera Calico Per-host Agent, Felix, provides network policy enforcement for Kubernetes."

    # Configure it to auto-start by default.
    & $NSSMPath set TigeraFelix Start SERVICE_AUTO_START
    & $NSSMPath set TigeraFelix ObjectName LocalSystem
    & $NSSMPath set TigeraFelix Type SERVICE_WIN32_OWN_PROCESS

    # Throttle process restarts if Felix restarts in under 1500ms.
    & $NSSMPath set TigeraFelix AppThrottle 1500

    # Create the log directory if needed.
    if (-Not(Test-Path "$env:CALICO_LOG_DIR"))
    {
        write "Creating log directory."
        md -Path "$env:CALICO_LOG_DIR"
    }
    & $NSSMPath set TigeraFelix AppStdout $env:CALICO_LOG_DIR\tigera-felix.log
    & $NSSMPath set TigeraFelix AppStderr $env:CALICO_LOG_DIR\tigera-felix.err.log

    # Configure online file rotation.
    & $NSSMPath set TigeraFelix AppRotateFiles 1
    & $NSSMPath set TigeraFelix AppRotateOnline 1
    # Rotate once per day.
    & $NSSMPath set TigeraFelix AppRotateSeconds 86400
    # Rotate after 10MB.
    & $NSSMPath set TigeraFelix AppRotateBytes 10485760

    Write-Host "Done installing Felix service."
}

function Remove-FelixService() {
    & $NSSMPath remove TigeraFelix confirm
}

function Install-ConfdService()
{
    Write-Host "Installing confd service..."

    # Ensure our service file can run.
    Unblock-File $baseDir\confd\confd-service.ps1

    # We run confd via a wrapper script to make it easier to update env vars.
    & $NSSMPath install TigeraConfd $powerShellPath
    & $NSSMPath set TigeraConfd AppParameters $baseDir\confd\confd-service.ps1
    & $NSSMPath set TigeraConfd AppDirectory $baseDir
    & $NSSMPath set TigeraConfd DependOnService "TigeraNode"
    & $NSSMPath set TigeraConfd DisplayName "Tigera Calico BGP Agent"
    & $NSSMPath set TigeraConfd Description "Tigera Calico BGP Agent, confd, configures BGP routing."

    # Configure it to auto-start by default.
    & $NSSMPath set TigeraConfd Start SERVICE_AUTO_START
    & $NSSMPath set TigeraConfd ObjectName LocalSystem
    & $NSSMPath set TigeraConfd Type SERVICE_WIN32_OWN_PROCESS

    # Throttle process restarts if confd restarts in under 1500ms.
    & $NSSMPath set TigeraConfd AppThrottle 1500

    # Create the log directory if needed.
    if (-Not(Test-Path "$env:CALICO_LOG_DIR"))
    {
        write "Creating log directory."
        md -Path "$env:CALICO_LOG_DIR"
    }
    & $NSSMPath set TigeraConfd AppStdout $env:CALICO_LOG_DIR\tigera-confd.log
    & $NSSMPath set TigeraConfd AppStderr $env:CALICO_LOG_DIR\tigera-confd.err.log

    # Configure online file rotation.
    & $NSSMPath set TigeraConfd AppRotateFiles 1
    & $NSSMPath set TigeraConfd AppRotateOnline 1
    # Rotate once per day.
    & $NSSMPath set TigeraConfd AppRotateSeconds 86400
    # Rotate after 10MB.
    & $NSSMPath set TigeraConfd AppRotateBytes 10485760

    Write-Host "Done installing confd service."
}

function Remove-ConfdService() {
    & $NSSMPath remove TigeraConfd confirm
}

function Wait-ForManagementIP($NetworkName)
{
    while ((Get-HnsNetwork | ? Name -EQ $NetworkName).ManagementIP -EQ $null)
    {
        Write-Host "Waiting for management IP to appear on network $NetworkName..."
        Start-Sleep 1
    }
    return (Get-HnsNetwork | ? Name -EQ $NetworkName).ManagementIP
}

function Get-LastBootTime()
{
    $bootTime = (Get-WmiObject win32_operatingsystem | select @{LABEL='LastBootUpTime';EXPRESSION={$_.lastbootuptime}}).LastBootUpTime
    if (($bootTime -EQ $null) -OR ($bootTime.length -EQ 0))
    {
        throw "Failed to get last boot time"
    }
    return $bootTime
}

$tigeraRegistryKey = "HKLM:\Software\Tigera"
$calicoRegistryKey = $tigeraRegistryKey + "\Calico"

function ensureRegistryKey()
{
    if (! (Test-Path $tigeraRegistryKey))
    {
        New-Item $tigeraRegistryKey
    }
    if (! (Test-Path $calicoRegistryKey))
    {
        New-Item $calicoRegistryKey
    }
}

function Get-StoredLastBootTime()
{
    try
    {
        return (Get-ItemProperty $calicoRegistryKey -ErrorAction Ignore).LastBootTime
    }
    catch
    {
        return ""
    }
}

function Set-StoredLastBootTime($lastBootTime)
{
    ensureRegistryKey

    return Set-ItemProperty $calicoRegistryKey -Name LastBootTime -Value $lastBootTime
}

function Wait-ForCalicoInit()
{
    Write-Host "Waiting for Calico initialisation to finish..."
    while ((Get-StoredLastBootTime) -NE (Get-LastBootTime)) {
        Write-Host "Waiting for Calico initialisation to finish..."
        Start-Sleep 1
    }
    Write-Host "Calico initialisation finished."
}

Export-ModuleMember -Function 'Test-*'
Export-ModuleMember -Function 'Install-*'
Export-ModuleMember -Function 'Remove-*'
Export-ModuleMember -Function 'Wait-*'
Export-ModuleMember -Function 'Get-*'
Export-ModuleMember -Function 'Set-*'
