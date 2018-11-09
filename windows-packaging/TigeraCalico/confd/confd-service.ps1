# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# This script is run from the main TigeraCalico directory.
. .\config.ps1

ipmo .\libs\calico\calico.psm1 -Force

if($env:CALICO_NETWORKING_BACKEND = "windows-bgp")
{
  Write-Host "Waiting for the first HNS network to be initialised..."
  Wait-ForManagementIP "External"
  Write-Host "Windows BGP is enabled, running confd..."
  # Run the tigera-confd binary.
  cd "$PSScriptRoot"
  & ..\tigera-calico.exe -confd -confd-confdir="$PSScriptRoot"
} else {
  Write-Host "Windows BGP is disabled, not running confd."
  while($True) {
    Start-Sleep 10
  }
}
