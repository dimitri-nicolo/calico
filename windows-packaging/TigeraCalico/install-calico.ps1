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

.\start-calico.ps1

Write-Host "Done, the Calico services should be running:"
Get-Service | where Name -Like 'Tigera*'
