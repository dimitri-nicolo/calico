# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# This script is run from the main TigeraCalico folder.
. .\config.ps1

ipmo .\libs\calico\calico.psm1
ipmo .\libs\hns\hns.psm1

$lastBootTime = Get-LastBootTime

if ($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp")
{
    Write-Host "Calico Windows BGP networking enabled."

    # Check if the node has been rebooted.  If so, the HNS networks will be in unknown state so we need to
    # clean them up and recreate them.
    $prevLastBootTime = Get-StoredLastBootTime
    if ($prevLastBootTime -NE $lastBootTime)
    {
        if ((Get-HNSNetwork | ? Type -NE nat))
        {
            Write-Host "First time Calico has run since boot up, cleaning out any old network state."
            Get-HNSNetwork | ? Type -NE nat | Remove-HNSNetwork
            do
            {
                Write-Host "Waiting for network deletion to complete."
                Start-Sleep 1
            } while ((Get-HNSNetwork | ? Type -NE nat))
        }
    }

    # Create a L2Bridge to trigger a vSwitch creation. Do this only once
    Write-Host "`nStart creating vSwitch. Note: Connection may get lost for RDP, please reconnect...`n"
    while (!(Get-HnsNetwork | ? Name -EQ "External"))
    {
        $result = New-HNSNetwork -Type L2Bridge -AddressPrefix "192.168.255.0/30" -Gateway "192.168.255.1" -Name "External" -Verbose
        if ($result.Error -OR (!$result.Success)) {
            Write-Host "Failed to create network, retrying..."
            Start-Sleep 1
        } else {
            break
        }
    }

    # Wait for the management IP to show up and then give an extra grace period for
    # the networking stack to settle down.
    $mgmtIP = Wait-ForManagementIP "External"
    Write-Host "Management IP detected on vSwitch: $mgmtIP."
    Start-Sleep 10

    Write-Host "Restarting BGP service to pick up any interface renumbering..."
    Restart-Service RemoteAccess
}

$env:CALICO_NODENAME_FILE = ".\nodename"

# We use this setting as a trigger for the other scripts to proceed.
Set-StoredLastBootTime $lastBootTime

# Run the startup script whenever kubelet (re)starts. This makes sure that we refresh our Node annotations if
# kubelet recreates the Node resource.
$kubeletPid = -1
while ($True)
{
    try
    {
        # Run tigera-calico.exe if kubelet starts/restarts
        $currentKubeletPid = (Get-Process -Name kubelet -ErrorAction Stop).id
        if ($currentKubeletPid -NE $kubeletPid)
        {
            Write-Host "Kubelet has (re)started, (re)initialising the node..."
            $kubeletPid = $currentKubeletPid
            while ($true)
            {
                .\tigera-calico.exe -startup
                if ($LastExitCode -EQ 0)
                {
                    Write-Host "Calico node initialisation succeeded; monitoring kubelet for restarts..."
                    break
                }

                Write-Host "Calico node initialisation failed, will retry..."
                Start-Sleep 1
            }
        }
    }
    catch
    {
        Write-Host "Kubelet not running, waiting for Kubelet to start..."
        $kubeletPid = -1
    }
    Start-Sleep 10
}
