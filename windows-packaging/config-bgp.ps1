echo "Hello, this is config-bgp.ps1"

# Read in template-generated config.
. .\peerings.ps1
. .\blocks.ps1

# Look for existing BGP router with the correct ID.
$router = Get-BgpRouter| Where-Object BgpIdentifier -eq $bgp_id
if (-not $router) {
    # There may be an existing BGP router with the wrong ID; remove it
    # if so.
    Remove-BgpRouter -Force
    # Add BGP router with the desired ID and AS number.
    Add-BgpRouter -BgpIdentifier $bgp_id -LocalASN $local_asn
}

foreach ($block in $blocks) {
    if ($block -ne "") {
	Add-BgpCustomRoute -Network $block
    }
}

foreach ($peering in $peerings) {
    if ($peering.Name) {
	Add-BgpPeer -Name $peering.Name -LocalIPAddress $local_ip -PeerIPAddress $peering.IP -PeerASN $peering.AS
    }
}

echo "We're done now"
