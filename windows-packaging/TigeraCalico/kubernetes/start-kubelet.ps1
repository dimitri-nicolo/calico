Param(
    [string]$NodeIp="",
    [string]$InterfaceName="Ethernet"
)

# Import the Calico library, included in the package.
ipmo c:\TigeraCalico\libs\calico\calico.psm1

# Wait for Calico to create the vSwitch.  This prevents kubelet's API server
# connection from being dropped when the vSwitch is created.
Wait-ForCalicoInit

. c:\TigeraCalico\config.ps1

Write-Host "Using configured nodename: $env:NODENAME."

if ($NodeIp -EQ "") {
    Write-Host "Auto-detecting node IP, looking for interface named 'vEthernet ($InterfaceName...'."
    $na = Get-NetAdapter | ? Name -Like "vEthernet ($InterfaceName*" | ? Status -EQ Up
    $NodeIp = (Get-NetIPAddress -InterfaceAlias $na.ifAlias -AddressFamily IPv4).IPAddress
    Write-Host "Detected node IP: $NodeIp."
}

$argList = @(`
    "--hostname-override=$env:NODENAME", `
    "--node-ip=$NodeIp", `
    "--max-pods=60",`
    "--v=4",`
    "--pod-infra-container-image=kubeletwin/pause",`
    "--resolv-conf=""""",`
    "--allow-privileged=true",`
    "--enable-debugging-handlers",`
    "--cluster-dns=10.96.0.10",`
    "--cluster-domain=cluster.local",`
    "--kubeconfig=c:\k\config",`
    "--hairpin-mode=promiscuous-bridge",`
    "--image-pull-progress-deadline=20m",`
    "--cgroups-per-qos=false",`
    "--logtostderr=true",`
    "--enforce-node-allocatable=""""",`
    "--network-plugin=cni",`
    "--cni-bin-dir=""c:\k\cni""",`
    "--cni-conf-dir ""c:\k\cni\config""",`
    "--kubeconfig=""c:\k\config"""`
)
Start-Process -FilePath c:\k\kubelet.exe `
    -ArgumentList $argList `
    -RedirectStandardOutput C:\k\kubelet.out.log `
    -RedirectStandardError C:\k\kubelet.err.log
