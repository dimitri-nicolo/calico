# Copyright (c) 2018 Tigera, Inc. All rights reserved.

. $PSScriptRoot\config.ps1

Start-Service TigeraNode
Start-Service TigeraFelix

if ($env:CALICO_NETWORKING_BACKEND -EQ "windows-bgp")
{
    Start-Service TigeraConfd
}
