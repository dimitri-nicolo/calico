# Copyright (c) 2018 Tigera, Inc. All rights reserved.

. $PSScriptRoot\config.ps1

$ErrorActionPreference = 'SilentlyContinue'

if ($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp")
{
    Stop-Service TigeraConfd
}
Stop-Service TigeraFelix
Stop-Service TigeraNode
