package managedcluster

import (
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/license"
)

type Licensing struct {
	cfg config.LicenseControllerCfg
}

func (l Licensing) CreateController(clusterName, ownerReference string,
	managedK8sCLI, managementK8sCLI kubernetes.Interface,
	managedCalicoCLI, managementCalicoCLI tigeraapi.Interface,
	restartChan chan<- string) controller.Controller {
	return license.New(clusterName, managedCalicoCLI, managementCalicoCLI, l.cfg)
}

func (l Licensing) HandleManagedClusterRemoved(clusterName string) {}

func (l Licensing) Initialize(stop chan struct{}, clusters ...string) {}

func NewLicensingController(cfg config.LicenseControllerCfg) ControllerManager {
	return &Licensing{cfg: cfg}
}
