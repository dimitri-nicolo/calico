# Copyright (c) 2018 Tigera, Inc. All rights reserved.

ipmo "$PSScriptRoot\libs\calico\calico.psm1" -Force

. $PSScriptRoot\config.ps1

Test-CalicoConfiguration

$ErrorActionPreference = 'SilentlyContinue'

Write-Host "Stopping Calico if it is running..."
& $PSScriptRoot\stop-calico.ps1

if ($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp")
{
    Remove-ConfdService
    Remove-CNIPlugin
}
Remove-NodeService
Remove-FelixService

Write-Host "Done."
