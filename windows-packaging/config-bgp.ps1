Write-Host "Start to reconfigure BGP"

# Read in template-generated config.
. .\peerings.ps1
. .\blocks.ps1


FUNCTION ProcessBgpRouter ($BgpId, $LocalAsn)
{
    # Look for existing BGP router with the correct ID.
    $found = $True
    try
    {
        $router = Get-BgpRouter| Where-Object BgpIdentifier -eq $BgpId
    }
    catch
    {
        $ErrorMessage = $_.Exception.Message
        Write-Host "Get-BgpRouter error:", $ErrorMessage

        $found = $False
    }
    if ($found)
    {
        if ($router.LocalASN -ne $localAsn) {
            # An existing BGP router with the wrong ASN; remove it.
            Remove-BgpRouter -Force
            Write-Host "Remove existing BGP router"
        }
        else
        {
            Write-Host "Identical BGP router exists. Do nothing"
            Return
        }
    }

    # Add BGP router with the desired ID and AS number.
    Add-BgpRouter -BgpIdentifier $BgpId -LocalASN $localAsn
    Write-Host "Add BGP router"
}

FUNCTION ProcessBgpBlocks ($Blocks)
{
    $current_blocks = (Get-BgpCustomRoute).Network
    $unused_blocks = [System.Collections.ArrayList]$current_blocks

    foreach ($block in $Blocks)
    {
        if ($current_blocks -contains $block)
        {
            Write-Host "do nothing with ", $block
            $unused_blocks.Remove($block)
            continue
        }
        if ($block -ne "")
        {
            Add-BgpCustomRoute -Network $block
            Write-Host "Add custom route", $block
        }
    }

    # Remove unused blocks
    foreach ($unused_block in $unused_blocks)
    {
        Remove-BgpCustomRoute -Network $unused_block -Force

        Write-Host "Remove unused block ", $unused_block
    }
}

FUNCTION ProcessBgpPeers ($Peerings, $LocalIp)
{
    $current_peers = Get-BgpPeer
    $unused_peers = [System.Collections.ArrayList]$current_peers

    # Add peerings. We try to minmize calling to BGP daemon.
    foreach ($peering in $Peerings)
    {
        if (-not $peering.Name)
        {
            continue
        }

        $done = $False

        foreach ($current_peer in $current_peers)
        {
            if ($current_peer.PeerName -eq $peering.Name)
            {

                if (($current_peer.LocalIPAddress -eq $LocalIp) -And ($current_peer.PeerIPAddress -eq $peering.IP) -And ($current_peer.PeerASN -eq $peering.AS))
                {
                    # Peer exists and identical
                    # Do nothing
                    Write-Host "do nothing with ", $current_peer.PeerName
                }
                else
                {
                    # Peer exists but differ
                    Remove-BgpPeer -Name $current_peer.PeerName -Force

                    Add-BgpPeer -Name $peering.Name -LocalIPAddress $LocalIp -PeerIPAddress $peering.IP -PeerASN $peering.AS

                    Write-Host "Update on ", $current_peer.PeerName
                    Write-Host
                }

                $done = $True

                # Remove this peer from unused.
                $unused_peers.Remove($current_peer)
            }
        }

        if (-not $done)
        {
            Add-BgpPeer -Name $peering.Name -LocalIPAddress $LocalIp -PeerIPAddress $peering.IP -PeerASN $peering.AS

            Write-Host "Add peer ", $peering.Name
        }
    }

    # Remove unused peering
    foreach ($unused_peer in $unused_peers)
    {
        Remove-BgpPeer -Name $unused_peer.PeerName -Force

        Write-Host "Remove unused peer ", $unused_peer.PeerName
    }
}

ProcessBgpRouter -BgpId $bgp_id -LocalAsn $local_asn

ProcessBgpBlocks -Blocks $blocks

ProcessBgpPeers -Peerings $peerings -LocalIp $local_ip

Write-Host "Reconfigure BGP completed"