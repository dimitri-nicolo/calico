param(
    [string]$NSSMPath="nssm",
    [string]$PowerShellPath="$PsHome\powershell.exe",
    [string]$LogDir="$PSScriptRoot\logs"
)

# Copyright (c) 2018 Tigera, Inc. All rights reserved.

if (-Not (Get-Command $NSSMPath -errorAction SilentlyContinue))
{
    throw "The nssm command wasn't found; please either install nssm to your path, or pass the -NSSMPath parameter."
}

if (-Not (Get-Command $PowerShellPath -errorAction SilentlyContinue))
{
    throw "The powershell command wasn't found at $PowerShellPath."
}

# We run Felix via a wrapper script to make it easier to update env vars.
& $NSSMPath install TigeraFelix $PowerShellPath
& $NSSMPath set TigeraFelix AppParameters $PSScriptRoot\felix-service.ps1
& $NSSMPath set TigeraFelix AppDirectory $PSScriptRoot
& $NSSMPath set TigeraFelix DisplayName "Tigera Secure EE Agent"
& $NSSMPath set TigeraFelix Description "Tigera Secure EE Per-host Agent, Felix, provides network policy enforcement for Kubernetes."

# Configure it to auto-start by default.
& $NSSMPath set TigeraFelix Start SERVICE_AUTO_START
& $NSSMPath set TigeraFelix ObjectName LocalSystem
& $NSSMPath set TigeraFelix Type SERVICE_WIN32_OWN_PROCESS

# Throttle process restarts if Felix restarts in under 1500ms.
& $NSSMPath set TigeraFelix AppThrottle 1500

# Create the log directory if needed.
if (-Not (Test-Path "$LogDir"))
{
    write "Creating log directory."
    md -Path "$LogDir"
}
& $NSSMPath set TigeraFelix AppStdout $LogDir\tigera-felix.log
& $NSSMPath set TigeraFelix AppStderr $LogDir\tigera-felix.err.log

# Configure online file rotation.
& $NSSMPath set TigeraFelix AppRotateFiles 1
& $NSSMPath set TigeraFelix AppRotateOnline 1
# Rotate once per day.
& $NSSMPath set TigeraFelix AppRotateSeconds 86400
# Rotate after 10MB.
& $NSSMPath set TigeraFelix AppRotateBytes 10485760
