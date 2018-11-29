package startup

import (
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/sirupsen/logrus"
)

const defaultNodenameFile = `c:\TigeraCalico\nodename`

var DEFAULT_INTERFACES_TO_EXCLUDE = []string{
	".*cbr.*",
	".*[Dd]ocker.*",
	".*\\(nat\\).*",
	".*Calico.*_ep", // Exclude our management endpoint.
	"Loopback.*",
}

// Checks that the filesystem is as expected and fix it if possible
func ensureFilesystemAsExpected() {
	logrus.Debug("ensureFilesystemAsExpected called on Windows; nothing to do.")
}

func ipv6Supported() bool {
	return false
}

// configureCloudOrchRef does not do anything for windows
func configureCloudOrchRef(node *api.Node) {
	logrus.Debug("configureCloudOrchRef called on Windows; nothing to do.")
}
