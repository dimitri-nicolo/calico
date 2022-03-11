package managedcluster

import (
	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/license"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/client-go/kubernetes"
)

type Licensing struct {
	cfg config.LicenseControllerCfg
}

func (l Licensing) New(clusterName, ownerReference string,
	managedK8sCLI, managementK8sCLI kubernetes.Interface,
	managedCalicoCLI, managementCalicoCLI tigeraapi.Interface,
	management bool, restartChan chan<- string) controller.Controller {
	return license.New(clusterName, managedCalicoCLI, managementCalicoCLI, l.cfg)
}

func (l Licensing) HandleManagedClusterRemoved(clusterName string) {
	return
}

func (l Licensing) Initialize(stop chan struct{}, clusters ...string) {
	return
}

func NewLicensingController(cfg config.LicenseControllerCfg) Controller {
	return &Licensing{cfg: cfg}
}
