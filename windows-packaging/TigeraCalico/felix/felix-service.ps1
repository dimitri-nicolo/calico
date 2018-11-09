# Copyright (c) 2018 Tigera, Inc. All rights reserved.

# This script is run from the main TigeraCalico folder.
. .\config.ps1

# Copy the nodename from the global setting.
$env:FELIX_FELIXHOSTNAME = $env:NODENAME

# Disable OpenStack metadata server support, which is not available on Windows.
$env:FELIX_METADATAADDR = "none"

# Run the calico-felix binary.
.\tigera-calico.exe -felix
