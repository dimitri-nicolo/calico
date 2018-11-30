# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# This script is run from the main TigeraCalico folder.
. .\config.ps1

ipmo .\libs\calico\calico.psm1
ipmo .\libs\hns\hns.psm1

if($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp") {
    Write-Host "Calico Windows BGP networking enabled."

    # Create a L2Bridge to trigger a vSwitch creation. Do this only once
    if(!(Get-HnsNetwork | ? Name -EQ "External"))
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

# Run the startup script to create our Node resource.
$env:CALICO_NODENAME_FILE=".\nodename"
do
{
    .\tigera-calico.exe -startup
    $retValue = $LastExitCode
    Start-Sleep 1
} while ($retValue -NE 0)

# Since the startup script is a one-shot; sleep forever so the service appears up.
$kubeletPid = -1
while($True) {
    try
    {
        # Run tigera-calico.exe if kubelet starts/restarts
        $currentKubeletPid = (Get-Process -Name kubelet -ErrorAction Stop).id
        if( !($currentKubeletPid -EQ $kubeletPid) )
        {
            Write-Host "Re-Started tigera-calico.exe"
            $kubeletPid = $currentKubeletPid
            .\tigera-calico.exe -startup
        }

    }
    catch
    {
        Write-Host "Kubelet is not running."
    }
    Start-Sleep 10
}
