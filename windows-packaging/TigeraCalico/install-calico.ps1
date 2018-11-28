# Copyright (c) 2018 Tigera, Inc. All rights reserved.

ipmo "$PSScriptRoot\libs\calico\calico.psm1" -Force

. $PSScriptRoot\config.ps1

Test-CalicoConfiguration

Install-NodeService
Install-FelixService
if ($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp")
{
    Install-ConfdService
    Install-CNIPlugin
}

Write-Host "Starting Calico..."
Write-Host "This may take several seconds if the vSwitch needs to be created."

Start-Service TigeraNode

# Wait for the management IP to show up and then give an extra grace period for
# the networking stack to settle down.
Wait-ForManagementIP "External"
Start-Sleep 10 

Start-Service TigeraFelix

if ($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp")
{
    Start-Service TigeraConfd
}

while ((Get-Service | where Name -Like 'Tigera*' | where Status -NE Running) -NE $null) {
    Write-Host "Waiting for the Calico services to be running..."
    Start-Sleep 1
}

Write-Host "Done, the Calico services are running:"
Get-Service | where Name -Like 'Tigera*'
