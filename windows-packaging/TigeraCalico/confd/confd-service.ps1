# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# This script is run from the main TigeraCalico directory.
. .\config.ps1

ipmo .\libs\calico\calico.psm1 -Force

if($env:CALICO_NETWORKING_BACKEND = "windows-bgp")
{
  Write-Host "Waiting for the first HNS network to be initialised..."
  Wait-ForManagementIP "External"
  Write-Host "Windows BGP is enabled, running confd..."

  cd "$PSScriptRoot"

  # Remove the old peerings and blocks so that confd will always trigger
  # reconfiguration at start of day.  This ensures that stopping and starting the service
  # reliably recovers from previous failures.
  rm peerings.ps1 -ErrorAction SilentlyContinue
  rm blocks.ps1 -ErrorAction SilentlyContinue

  # Run the tigera-confd binary.
  & ..\tigera-calico.exe -confd -confd-confdir="$PSScriptRoot"
} else {
  Write-Host "Windows BGP is disabled, not running confd."
  while($True) {
    Start-Sleep 10
  }
}
