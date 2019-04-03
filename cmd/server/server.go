package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	certutil "k8s.io/client-go/util/cert"

	"github.com/caimeo/iniflags"
	//	"github.com/tigera/compliance/pkg/errors"

	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/version"
)

var versionFlag = flag.Bool("version", false, "Print version information")
var complianceServerCertPath = flag.String("certpath", "apiserver.local.config/certificates/apiserver.crt", "ssl cert path")
var complianceServerKeyPath = flag.String("keypath", "apiserver.local.config/certificates/apiserver.key", "ssl key path")

var els *elastic.Client
var sig chan os.Signal

func main() {
	initIniflags()

	if *versionFlag {
		runVersion()
	}

	initElastic()

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

// MaybeDefaultWithSelfSignedCerts is a minimal copy of the version in the kubernetes apiserver in order to avoid versioning issues.
func MaybeDefaultWithSelfSignedCerts(publicAddress string, alternateDNS []string, alternateIPs []net.IP) error {
	canReadCertAndKey, err := certutil.CanReadCertAndKey(*complianceServerCertPath, *complianceServerKeyPath)
	if err != nil {
		return err
	}
	if !canReadCertAndKey {
		// add localhost to the valid alternates
		alternateDNS = append(alternateDNS, "localhost")

		if cert, key, err := certutil.GenerateSelfSignedCertKey(publicAddress, alternateIPs, alternateDNS); err != nil {
			return fmt.Errorf("unable to generate self signed cert: %v", err)
		} else {
			if err := certutil.WriteCert(*complianceServerCertPath, cert); err != nil {
				return err
			}

			if err := certutil.WriteKey(*complianceServerKeyPath, key); err != nil {
				return err
			}
			log.Infof("Generated self-signed cert (%s, %s)", *complianceServerCertPath, *complianceServerKeyPath)
		}
	}

	return nil
}
