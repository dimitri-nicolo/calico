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

# We run confd via a wrapper script to make it easier to update env vars.
& $NSSMPath install TigeraConfd $PowerShellPath
& $NSSMPath set TigeraConfd AppParameters $PSScriptRoot\confd-service.ps1
& $NSSMPath set TigeraConfd AppDirectory $PSScriptRoot
& $NSSMPath set TigeraConfd DisplayName "Tigera Calico BGP Agent"
& $NSSMPath set TigeraConfd Description "Tigera Calico BGP Agent, confd, configures BGP routing."

# Configure it to auto-start by default.
& $NSSMPath set TigeraConfd Start SERVICE_AUTO_START
& $NSSMPath set TigeraConfd ObjectName LocalSystem
& $NSSMPath set TigeraConfd Type SERVICE_WIN32_OWN_PROCESS

# Throttle process restarts if confd restarts in under 1500ms.
& $NSSMPath set TigeraConfd AppThrottle 1500

# Create the log directory if needed.
if (-Not (Test-Path "$LogDir"))
{
    write "Creating log directory."
    md -Path "$LogDir"
}
& $NSSMPath set TigeraConfd AppStdout $LogDir\tigera-confd.log
& $NSSMPath set TigeraConfd AppStderr $LogDir\tigera-confd.err.log

# Configure online file rotation.
& $NSSMPath set TigeraConfd AppRotateFiles 1
& $NSSMPath set TigeraConfd AppRotateOnline 1
# Rotate once per day.
& $NSSMPath set TigeraConfd AppRotateSeconds 86400
# Rotate after 10MB.
& $NSSMPath set TigeraConfd AppRotateBytes 10485760
