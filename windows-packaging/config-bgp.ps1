Write-Host "Start to reconfigure BGP"

# Read in template-generated config.
. .\peerings.ps1
. .\blocks.ps1

ipmo .\config-bgp.psm1

ProcessBgpRouter -BgpId $bgp_id -LocalAsn $local_asn

ProcessBgpBlocks -Blocks $blocks

ProcessBgpPeers -Peerings $peerings -LocalIp $local_ip

Write-Host "Reconfigure BGP completed"
