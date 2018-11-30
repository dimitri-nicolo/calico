# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# This script is run from the main TigeraCalico folder.
. .\config.ps1

ipmo .\libs\calico\calico.psm1
ipmo .\libs\hns\hns.psm1

if ($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp")
{
    Write-Host "Calico Windows BGP networking enabled."

    # Create a L2Bridge to trigger a vSwitch creation. Do this only once
    if (!(Get-HnsNetwork | ? Name -EQ "External"))
    {
        Write-Host "`nStart creating vSwitch. Note: Connection may get lost for RDP, please reconnect...`n"
        New-HNSNetwork -Type L2Bridge -AddressPrefix "192.168.255.0/30" -Gateway "192.168.255.1" -Name "External" -Verbose

        # Wait for the management IP to show up and then give an extra grace period for
        # the networking stack to settle down.
        $mgmtIP = Wait-ForManagementIP "External"
        Write-Host "Management IP detected on vSwitch: $mgmtIP."
        Start-Sleep 10

        Write-Host "Restarting BGP service to pick up any interface renumbering..."
        Restart-Service RemoteAccess
    }
}

$env:CALICO_NODENAME_FILE = ".\nodename"

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
