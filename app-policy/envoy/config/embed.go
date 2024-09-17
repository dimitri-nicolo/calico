package envoyconfig

import (
	_ "embed"
)

const Path string = "/etc/tigera/envoy.yaml"

//go:embed envoy-config.yaml.gotmpl
var Config string
