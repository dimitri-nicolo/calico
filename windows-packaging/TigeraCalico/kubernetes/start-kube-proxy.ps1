Param(
    [string]$NetworkName = "Calico"
)

# Import the Calico and HNS libraries, included in the package.
ipmo -Force c:\TigeraCalico\libs\calico\calico.psm1
ipmo -Force c:\TigeraCalico\libs\hns\hns.psm1

# Wait for Calico to create the vSwitch.  This prevents kube-proxy's API server
# connection from being dropped when the vSwitch is created.
Wait-ForCalicoInit

. c:\TigeraCalico\config.ps1

# Now, wait for the Calico network to be created when the first pod is networked.
Write-Host "Waiting for HNS network $NetworkName to be created..."
while (-Not (Get-HnsNetwork | ? Name -EQ $NetworkName)) {
    Write-Debug "Still waiting for HNS network..."
    Start-Sleep 1
}
Write-Host "HNS network $NetworkName found."

# Determine the kube-proxy version.
$kubeProxyVer = $(c:\k\kube-proxy.exe --version)
$kubeProxyGE114 = $false
if ($kubeProxyVer -match "v([0-9])\.([0-9]+)") {
    $major = $Matches.1 -as [int]
    $minor = $Matches.2 -as [int]
    $kubeProxyGE114 = ($major -GT 1 -OR $major -EQ 1 -AND $minor -GE 14)
}

# Build up the arguments for starting kube-proxy.
$argList = @(`
    "--hostname-override=$env:NODENAME", `
    "--v=4",`
    "--proxy-mode=kernelspace",`
    "--kubeconfig=""c:\k\config"""`
)
$extraFeatures = @()

if ($kubeProxyGE114) {
    Write-Host "Detected kube-proxy >= 1.14, enabling DSR feature gate."
    $extraFeatures += "WinDSR=true"
    $argList += "--enable-dsr=true"
}

$network = (Get-HnsNetwork | ? Name -EQ $NetworkName)
if ($network.Type -EQ "Overlay") {
    if (-NOT $kubeProxyGE114) {
        throw "Overlay network requires kube-proxy >= v1.14.  Detected $kubeProxyVer."
    }
    # This is a VXLAN network, kube-proxy needs to know the source IP to use for SNAT operations.
    Write-Host "Detected VXLAN network, waiting for Calico host endpoint to be created..."
    while (-Not (Get-HnsEndpoint | ? Name -EQ "Calico_ep")) {
        Start-Sleep 1
    }
    Write-Host "Host endpoint found."
    $sourceVip = (Get-HnsEndpoint | ? Name -EQ "Calico_ep").IpAddress
    $argList += "--source-vip=$sourceVip"
    $extraFeatures += "WinOverlay=true"
}

if ($extraFeatures.Length -GT 0) {
    $featuresStr = $extraFeatures -join ","
    $argList += "--feature-gates=$featuresStr"
    Write-Host "Enabling feature gates: $extraFeatures."
}

# kube-proxy doesn't handle resync if there are pre-exisitng policies, clean them
# all out before (re)starting kube-proxy.
$policyLists = Get-HnsPolicyList
if ($policyLists) {
    $policyLists | Remove-HnsPolicyList
}

# We'll also pick up a network name env var from the Calico config file.  Override it
# since hte value in the config file may be a regex.
$env:KUBE_NETWORK=$NetworkName
Start-Process `
    -FilePath c:\k\kube-proxy.exe `
    -ArgumentList $argList `
    -RedirectStandardOutput C:\k\kube-proxy.out.log `
    -RedirectStandardError C:\k\kube-proxy.err.log
