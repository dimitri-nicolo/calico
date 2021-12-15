 ## Ensure the function is available
ipmo .\config-bgp.psm1 -Force

describe 'BGP Router' {

    it 'should add new router if nothing there' {
        try { Remove-BgpRouter -Force } catch {}

        ProcessBgpRouter -BgpId "10.10.10.10" -LocalAsn 64512

        $router = Get-BgpRouter
	
        $router.BgpIdentifier | should be "10.10.10.10"
        $router.LocalASN | should be 64512
    }

    it 'should take no action if router exists' {
        try { Remove-BgpRouter -Force } catch {}

        Add-BgpRouter -BgpIdentifier "10.10.10.10" -LocalASN 64512

        $output = ProcessBgpRouter -BgpId "10.10.10.10" -LocalAsn 64512
        $output | should be $null

        $router = Get-BgpRouter

        $router.BgpIdentifier | should be "10.10.10.10"
        $router.LocalASN | should be 64512
    }

    it 'should add new router and remove old one with wrong id' {
        try { Remove-BgpRouter -Force } catch {}

        Add-BgpRouter -BgpIdentifier "9.9.9.9" -LocalASN 64512

        ProcessBgpRouter -BgpId "10.10.10.10" -LocalAsn 64512

        $router = Get-BgpRouter
	
        $router.BgpIdentifier | should be "10.10.10.10"
        $router.LocalASN | should be 64512
    }

    it 'should add new router and remove old one with wrong ASN' {
        try { Remove-BgpRouter -Force } catch {}

        Add-BgpRouter -BgpIdentifier "10.10.10.10" -LocalASN 64515

        ProcessBgpRouter -BgpId "10.10.10.10" -LocalAsn 64512

        $router = Get-BgpRouter
	
        $router.BgpIdentifier | should be "10.10.10.10"
        $router.LocalASN | should be 64512
    }
}

function Ensure-NewRouter {
    try { Remove-BgpRouter -Force } catch {}
    Add-BgpRouter -BgpIdentifier "10.10.10.10" -LocalASN 64512
}

describe 'BGP custom routes' {
    
    $blocks =
            "10.244.199.0/26",
            "10.244.200.0/26",
            ""

    $blocks_old =
            "10.244.198.0/26",
            "10.244.200.0/26",
            "10.244.201.0/26",
            ""

    it 'should add new routes if nothing there' {
        Ensure-NewRouter
        ProcessBgpBlocks -Blocks $blocks

	$routes = Get-BgpCustomRoute
        $routes.Network | should be $blocks
    }

    it 'should take no action if routes exist' {
        Ensure-NewRouter
        ProcessBgpBlocks -Blocks $blocks

        $output = ProcessBgpBlocks -Blocks $blocks
        $output | should be $null

	$routes = Get-BgpCustomRoute
        $routes.Network | should be $blocks
    }

    it 'should remove old routes' {
        Ensure-NewRouter
        ProcessBgpBlocks -Blocks $blocks_old

        $routes = Get-BgpCustomRoute
        $routes.Network | should be $blocks_old

        $output = ProcessBgpBlocks -Blocks $blocks

	    $routes = Get-BgpCustomRoute
        $routes.Network | should be $blocks
    }
}

function CheckPeers($Peers) {
    $BgpPeers = Get-BgpPeer
    $BgpPeers.Count | should be $peers.Count
    
    foreach ($peer in $Peers)
    {
        $index = $Peers.IndexOf($peer)
        $bgpPeer = $BgpPeers[$index]

        $bgpPeer.PeerName | should be $peer.Name
        $bgpPeer.PeerIPAddress | should be $peer.IP
        $bgpPeer.PeerASN | should be $peer.AS
    }
}

describe 'BGP peerings' {
    $local_ip = "10.10.10.10"

    $peerings =
        @{ Name = "Mesh_10_10_10_1"; IP = "10.10.10.1"; AS = 64512 },
        @{ Name = "Mesh_10_10_10_2"; IP = "10.10.10.2"; AS = 64512 }

    $peerings_old =
        @{ Name = "Mesh_10_10_10_0"; IP = "10.10.10.0"; AS = 64512 },
        @{ Name = "Mesh_10_10_10_1"; IP = "10.10.10.1"; AS = 64511 },
        @{ Name = "Mesh_10_10_10_2"; IP = "10.10.10.2"; AS = 64512 },
        @{ Name = "Mesh_10_10_10_3"; IP = "10.10.10.3"; AS = 64512 }

    it 'should add new peers if nothing there' {
        Ensure-NewRouter
        ProcessBgpPeers -Peerings $peerings -LocalIp $local_ip

        CheckPeers -Peers $peerings
    }
    
    it 'should take no action if peers exist' {
        Ensure-NewRouter
        ProcessBgpPeers -Peerings $peerings -LocalIp $local_ip

        $output = ProcessBgpPeers -Peerings $peerings -LocalIp $local_ip
        $output | should be $null

        CheckPeers -Peers $peerings
    }
 
    it 'should remove old peers' {
        Ensure-NewRouter
        ProcessBgpPeers -Peerings $peerings_old -LocalIp $local_ip
        CheckPeers -Peers $peerings_old

        ProcessBgpPeers -Peerings $peerings -LocalIp $local_ip

        CheckPeers -Peers $peerings
    }

    $peerings_a =
        @{ Name = "Mesh_10_10_10_1"; IP = "10.10.10.1"; AS = 64512 },
        @{ Name = "Mesh_10_10_10_2"; IP = "10.10.10.2"; AS = 64512 }

    $peerings_b =
        @{ Name = "Mesh_10_10_10_1"; IP = "10.10.10.1"; AS = 64512 },
        @{ Name = "Something_10_10_10_2"; IP = "10.10.10.2"; AS = 64512 }

    it 'should handle renamed peer' {
        Ensure-NewRouter
        ProcessBgpPeers -Peerings $peerings_a -LocalIp $local_ip
        CheckPeers -Peers $peerings_a

        ProcessBgpPeers -Peerings $peerings_b -LocalIp $local_ip
        CheckPeers -Peers $peerings_b
    }
}

