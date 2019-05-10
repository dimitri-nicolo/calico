# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# This script is run from the main TigeraCalico folder.
. .\config.ps1

ipmo .\libs\calico\calico.psm1 -Force

# Wait for vSwitch to be created, etc.
Wait-ForCalicoInit

# Copy the nodename from the global setting.
$env:FELIX_FELIXHOSTNAME = $env:NODENAME

# Disable OpenStack metadata server support, which is not available on Windows.
$env:FELIX_METADATAADDR = "none"

# VXLAN settings.
$env:FELIX_VXLANVNI = "$env:VXLAN_VNI"

# Autoconfigure the IPAM block mode.
if ($env:CNI_IPAM_TYPE -EQ "host-local") {
    $env:USE_POD_CIDR = "true"
} else {
    $env:USE_POD_CIDR = "false"
}

# Run the calico-felix binary.
.\tigera-calico.exe -felix
