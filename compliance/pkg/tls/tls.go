package tls

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	certutil "k8s.io/client-go/util/cert"
	keyutil "k8s.io/client-go/util/keyutil"
)

// GenerateSelfSignedCertsIfNeeded is a minimal copy of the version in the kubernetes apiserver in order to avoid versioning issues.
func GenerateSelfSignedCertsIfNeeded(publicAddress string, alternateDNS []string, alternateIPs []net.IP, certpath string, keypath string) error {
	canReadCertAndKey, err := certutil.CanReadCertAndKey(certpath, keypath)
	if err != nil {
		return err
	}
	if !canReadCertAndKey {
		// add localhost to the valid alternates
		alternateDNS = append(alternateDNS, "localhost")

		if cert, key, err := certutil.GenerateSelfSignedCertKey(publicAddress, alternateIPs, alternateDNS); err != nil {
			return fmt.Errorf("unable to generate self signed cert: %v", err)
		} else {
			if err := certutil.WriteCert(certpath, cert); err != nil {
				return err
			}

			if err := keyutil.WriteKey(keypath, key); err != nil {
				return err
			}
			log.Infof("Generated self-signed cert (%s, %s)", certpath, keypath)
		}
	}

	return nil
}
