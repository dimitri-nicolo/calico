package bootstrap

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"regexp"

	"github.com/pkg/errors"

	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/utils"
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

	for _, t := range tgts {
		pt := proxy.Target{
			Path: t.Path,
		}

		var err error
		pt.Dest, err = url.Parse(t.Dest)
		if err != nil {
			return nil, errors.Errorf("Incorrect URL %q for path %q: %s", t.Dest, t.Path, err)
		}

		if t.TokenPath != "" {
			token, err := ioutil.ReadFile(t.TokenPath)
			if err != nil {
				return nil, errors.Errorf("Failed reading token from %s: %s", t.TokenPath, err)
			}

			pt.Token = string(token)
		}

		if t.CABundlePath != "" {
			pt.CA, err = utils.LoadX509FromFile(t.CABundlePath)
			if err != nil {
				return nil, errors.WithMessage(err, "LoadX509FromFile")
			}
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

		ret = append(ret, pt)
	}

	return ret, nil
}
