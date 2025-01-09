package pinnedversion

import (
	"fmt"
	"os"
	"strings"

	"github.com/projectcalico/calico/release/pkg/manager/calico"
	"gopkg.in/yaml.v2"
)

func ParseVersionsFile(pinnedVersionPath string) ([]calico.Option, []string, error) {
	var pinnedVersionFile []PinnedVersion
	if pinnedVersionData, err := os.ReadFile(pinnedVersionPath); err != nil {
		return nil, nil, err
	} else if err := yaml.Unmarshal([]byte(pinnedVersionData), &pinnedVersionFile); err != nil {
		return nil, nil, err
	}
	pinnedVersion := pinnedVersionFile[0]
	imgs := []string{}
	for _, c := range pinnedVersion.Components {
		image := c.Image
		version := c.Version
		if image != "" && version != "" {
			image := strings.TrimPrefix(image, "tigera/")
			imgs = append(imgs, fmt.Sprintf("%s:%s", image, version))
		}
	}
	return []calico.Option{
		calico.WithVersion(pinnedVersion.Title),
		calico.WithOperatorVersion(pinnedVersion.TigeraOperator.Version),
	}, imgs, nil
}
