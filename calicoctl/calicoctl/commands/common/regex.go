package common

import "regexp"

var (
	// BGPPeerRegex checks for Word_<IP> where every octate is seperated by "_"
	// regardless of IP protocols.
	// Example match: "Mesh_192_168_56_101" or "Mesh_fd80_24e2_f998_72d7__2"
	BGPPeerRegex = regexp.MustCompile(`^(Global|Node|Mesh)_(.+)$`)
)
