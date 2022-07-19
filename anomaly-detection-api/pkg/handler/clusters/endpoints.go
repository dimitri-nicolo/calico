package clusters

import "regexp"

var (
	ClustersTopPath              = "/clusters/"
	ModelStorageEndpointRegex    = regexp.MustCompile(`^\/(clusters){1}\/(.+)\/(models){1}\/(dynamic|static){1}(\/[^\/]+){2}$`)
	LogTypeMetadataEndpointRegex = regexp.MustCompile(`^\/(clusters){1}\/(.+)\/(flow|dns|l7){1}\/(metadata){1}$`)
)
