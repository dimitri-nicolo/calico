# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# This script is run from the main TigeraCalico directory.
. .\config.ps1

ipmo .\libs\calico\calico.psm1 -Force

# Autoconfigure the IPAM block mode.
if ($env:CNI_IPAM_TYPE -EQ "host-local") {
  $env:USE_POD_CIDR = "true"
} else {
  $env:USE_POD_CIDR = "false"
}

if($env:CALICO_NETWORKING_BACKEND = "windows-bgp")
{
  Wait-ForCalicoInit
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
