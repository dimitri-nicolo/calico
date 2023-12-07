package testdata

var (
	DefaultEmbeddedDirectives = []string{
		"Include @coraza.conf-recommended",
		"Include @crs-setup.conf.example",
		"Include @owasp_crs/*.conf",
		"SecRuleEngine On",
	}
)

func DirectivesToCLI(directives []string) []string {
	res := []string{}
	for _, d := range directives {
		res = append(res, "-waf-directive", d)
	}
	return res
}
