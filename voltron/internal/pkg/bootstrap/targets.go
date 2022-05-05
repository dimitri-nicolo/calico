package bootstrap

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"regexp"

	"github.com/pkg/errors"

	"github.com/tigera/voltron/internal/pkg/proxy"
)

// Target is the format for env variable to set proxy targets
type Target struct {
	// Path is the path portion of the URL based on which we proxy
	Path string `json:"path"`
	// Dest is the destination URL
	Dest string `json:"url"`
	// TokenPath is where we read the Bearer token from (if non-empty)
	TokenPath string `json:"tokenPath,omitempty"`
	// CABundlePath is where we read the CA bundle from to authenticate the
	// destination (if non-empty)
	CABundlePath string `json:"caBundlePath,omitempty"`
	// PathRegexp, if not nil, checks if Regexp matches the path
	PathRegexp strAsByteSlice `json:"pathRegexp,omitempty"`
	// PathReplace if not nil will be used to replace PathRegexp matches
	PathReplace strAsByteSlice `json:"pathReplace,omitempty"`
	// AllowInsecureTLS allows https with insecure tls settings
	AllowInsecureTLS bool `json:"allowInsecureTLS,omitempty"`
}

// Targets allows unmarshal the json array
type Targets []Target

// Decode deserializes the list of proxytargets
func (pt *Targets) Decode(envVar string) error {
	err := json.Unmarshal([]byte(envVar), pt)
	if err != nil {
		return err
	}

	return nil
}

type strAsByteSlice []byte

func (b *strAsByteSlice) UnmarshalJSON(j []byte) error {
	// strip the enclosing ""
	*b = j[1 : len(j)-1]
	return nil
}

// ProxyTargets decodes Targets into []proxy.Target
func ProxyTargets(tgts Targets) ([]proxy.Target, error) {
	var ret []proxy.Target

	// pathSet helps keep track of the paths we've seen so we don't have duplicates
	pathSet := make(map[string]bool)

	for _, t := range tgts {
		if t.Path == "" {
			return nil, errors.New("proxy target path cannot be empty")
		} else if pathSet[t.Path] {
			return nil, errors.Errorf("duplicate proxy target path %s", t.Path)
		}

		pt := proxy.Target{
			Path:             t.Path,
			AllowInsecureTLS: t.AllowInsecureTLS,
		}

		var err error
		pt.Dest, err = url.Parse(t.Dest)
		if err != nil {
			return nil, errors.Errorf("Incorrect URL %q for path %q: %s", t.Dest, t.Path, err)
		}

		if pt.Dest.Scheme == "https" && !t.AllowInsecureTLS && t.CABundlePath == "" {
			return nil, errors.Errorf("target for path '%s' must specify the ca bundle if AllowInsecureTLS is false when the scheme is https", t.Path)
		}

		if t.TokenPath != "" {
			token, err := ioutil.ReadFile(t.TokenPath)
			if err != nil {
				return nil, errors.Errorf("Failed reading token from %s: %s", t.TokenPath, err)
			}

			pt.Token = string(token)
		}

		if t.CABundlePath != "" {
			pt.CAPem = t.CABundlePath
		}

		if t.PathReplace != nil && t.PathRegexp == nil {
			return nil, errors.Errorf("PathReplace specified but PathRegexp is not")
		}

		if t.PathRegexp != nil {
			r, err := regexp.Compile(string(t.PathRegexp))
			if err != nil {
				return nil, errors.Errorf("PathRegexp failed: %s", err)
			}
			pt.PathRegexp = r
		}

		pt.PathReplace = t.PathReplace

		pathSet[pt.Path] = true
		ret = append(ret, pt)
	}

	return ret, nil
}
