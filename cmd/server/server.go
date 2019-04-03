package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/caimeo/iniflags"

	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/tls"
	"github.com/tigera/compliance/pkg/version"
)

var versionFlag = flag.Bool("version", false, "Print version information")
var complianceServerCertPath = flag.String("certpath", "apiserver.local.config/certificates/apiserver.crt", "tls cert path")
var complianceServerKeyPath = flag.String("keypath", "apiserver.local.config/certificates/apiserver.key", "tls key path")

var els *elastic.Client
var sig chan os.Signal

func main() {
	initIniflags()

	if *versionFlag {
		runVersion()
	}

	//initElastic()

	initTLS()

	initSystemSignals()
}

// reads .settings
func initIniflags() {
	iniflags.SetConfigFile(".settings")
	iniflags.SetAllowMissingConfigFile(true)
	iniflags.Parse()
}

func runVersion() {
	version.Version()
	os.Exit(0)
}

func initElastic() {
	els, _ := elastic.NewFromEnv()
	log.Infof("Created %s", els)
}

func initSystemSignals() {
	sig = make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}

func initTLS() {
	altIPs := []net.IP{net.ParseIP("127.0.0.1")}
	if err := tls.GenerateSelfSignedCerts("localhost", nil, altIPs, *complianceServerCertPath, *complianceServerKeyPath); err != nil {
		log.Errorf("Error creating self-signed certificates: %v", err)
		os.Exit(1)
	}
}
