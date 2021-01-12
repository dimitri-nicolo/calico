/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	restclient "k8s.io/client-go/rest"

	"github.com/tigera/apiserver/cmd/apiserver/server"
	_ "github.com/tigera/apiserver/pkg/apis/projectcalico/install"
	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/apiserver/pkg/apiserver"
	calicoclient "github.com/tigera/apiserver/pkg/client/clientset_generated/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultEtcdPathPrefix = ""

func init() {
	rand.Seed(time.Now().UnixNano())
}

type TestServerConfig struct {
	etcdServerList                []string
	emptyObjFunc                  func() runtime.Object
	enableManagedClusterCreateAPI bool
	managedClustersCACertPath     string
	managedClustersCAKeyPath      string
	managementClusterAddr         string
	applyTigeraLicense            bool
}

// NewTestServerConfig is a default constructor for the standard test-apiserver setup
func NewTestServerConfig() *TestServerConfig {
	return &TestServerConfig{
		etcdServerList: []string{"http://localhost:2379"},
	}
}

func withConfigGetFreshApiserverServerAndClient(
	t *testing.T,
	serverConfig *TestServerConfig,
) (*apiserver.ProjectCalicoServer,
	calicoclient.Interface,
	*restclient.Config,
	func(),
) {
	securePort := rand.Intn(31743) + 1024
	secureAddr := fmt.Sprintf("https://localhost:%d", securePort)
	stopCh := make(chan struct{})
	serverFailed := make(chan struct{})
	shutdownServer := func() {
		t.Logf("Shutting down server on port: %d", securePort)
		close(stopCh)
	}

	t.Logf("Starting server on port: %d", securePort)
	ro := genericoptions.NewRecommendedOptions(defaultEtcdPathPrefix, apiserver.Codecs.LegacyCodec(v3.SchemeGroupVersion),
		genericoptions.NewProcessInfo("tigera-apiserver", "tigera-system"))
	ro.Etcd.StorageConfig.Transport.ServerList = serverConfig.etcdServerList
	options := &server.CalicoServerOptions{
		RecommendedOptions: ro,
		DisableAuth:        true,
		StopCh:             stopCh,
	}
	options.RecommendedOptions.SecureServing.BindPort = securePort
	// Set this so that we avoid RecommendedOptions.CoreAPI's initialization from calling InClusterConfig()
	// and uses our fv kubeconfig instead.
	options.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath = "../test-apiserver-kubeconfig.conf"
	options.EnableManagedClustersCreateAPI = serverConfig.enableManagedClusterCreateAPI
	options.ManagedClustersCACertPath = serverConfig.managedClustersCACertPath
	options.ManagedClustersCAKeyPath = serverConfig.managedClustersCAKeyPath
	options.ManagementClusterAddr = serverConfig.managementClusterAddr
	//options.RecommendedOptions.SecureServing.BindAddress=

	var err error
	pcs, err := server.PrepareServer(options)
	if err != nil {
		close(serverFailed)
		t.Fatalf("Error preparing the server: %v", err)
	}

	// Run the server in the background
	go func() {
		err := server.RunServer(options, pcs)
		if err != nil {
			close(serverFailed)
			t.Fatalf("Error running the server: %v", err)
		}
	}()

	if err := waitForApiserverUp(secureAddr, serverFailed); err != nil {
		t.Fatalf("%v", err)
	}
	if pcs == nil {
		t.Fatal("Calico server is nil")
	}

	cfg := &restclient.Config{}
	cfg.Host = secureAddr
	cfg.Insecure = true
	clientset, err := calicoclient.NewForConfig(cfg)
	if nil != err {
		t.Fatal("can't make the client from the config", err)
	}

	var licenseClient = clientset.ProjectcalicoV3().LicenseKeys()
	_ = licenseClient.Delete(context.Background(), "default", metav1.DeleteOptions{})

	if serverConfig.applyTigeraLicense {
		validLicenseKey := getValidLicenseKey("default")
		_, err = licenseClient.Create(context.Background(), validLicenseKey, metav1.CreateOptions{})
		if err != nil {
			t.Fatal("License cannot be applied", err)
		}
	}

	return pcs, clientset, cfg, shutdownServer
}

func getFreshApiserverServerAndClient(
	t *testing.T,
	newEmptyObj func() runtime.Object,
) (*apiserver.ProjectCalicoServer, calicoclient.Interface, func()) {
	serverConfig := &TestServerConfig{
		etcdServerList:     []string{"http://localhost:2379"},
		emptyObjFunc:       newEmptyObj,
		applyTigeraLicense: true,
	}
	pcs, client, _, shutdownFunc := withConfigGetFreshApiserverServerAndClient(t, serverConfig)
	return pcs, client, shutdownFunc
}

func getFreshApiserverAndClient(
	t *testing.T,
	newEmptyObj func() runtime.Object,
	applyTigeraLicense bool,
) (calicoclient.Interface, func()) {
	serverConfig := &TestServerConfig{
		etcdServerList:     []string{"http://localhost:2379"},
		emptyObjFunc:       newEmptyObj,
		applyTigeraLicense: applyTigeraLicense,
	}
	_, client, _, shutdownFunc := withConfigGetFreshApiserverServerAndClient(t, serverConfig)
	return client, shutdownFunc
}

func customizeFreshApiserverAndClient(
	t *testing.T,
	serverConfig *TestServerConfig,
) (calicoclient.Interface, func()) {
	_, client, _, shutdownFunc := withConfigGetFreshApiserverServerAndClient(t, serverConfig)
	return client, shutdownFunc
}

func waitForApiserverUp(serverURL string, stopCh <-chan struct{}) error {
	interval := 1 * time.Second
	timeout := 30 * time.Second
	startWaiting := time.Now()
	tries := 0
	return wait.PollImmediate(interval, timeout,
		func() (bool, error) {
			select {
			// we've been told to stop, so no reason to keep going
			case <-stopCh:
				return true, fmt.Errorf("apiserver failed")
			default:
				klog.Infof("Waiting for : %#v", serverURL)
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				c := &http.Client{Transport: tr}
				_, err := c.Get(serverURL)
				if err == nil {
					klog.Infof("Found server after %v tries and duration %v",
						tries, time.Since(startWaiting))
					return true, nil
				}
				tries++
				return false, nil
			}
		},
	)
}

func getValidLicenseKey(name string) *v3.LicenseKey {

	validLicenseKey := &v3.LicenseKey{ObjectMeta: metav1.ObjectMeta{Name: name}}

	validLicenseKey.Spec.Certificate = validLicenseCertificate
	validLicenseKey.Spec.Token = validLicenseToken

	return validLicenseKey
}

const validLicenseCertificate = `-----BEGIN CERTIFICATE-----
MIIFxjCCA66gAwIBAgIQVq3rz5D4nQF1fIgMEh71DzANBgkqhkiG9w0BAQsFADCB
tTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1
cml0eSA8c2lydEB0aWdlcmEuaW8+MT8wPQYDVQQDEzZUaWdlcmEgRW50aXRsZW1l
bnRzIEludGVybWVkaWF0ZSBDZXJ0aWZpY2F0ZSBBdXRob3JpdHkwHhcNMTgwNDA1
MjEzMDI5WhcNMjAxMDA2MjEzMDI5WjCBnjELMAkGA1UEBhMCVVMxEzARBgNVBAgT
CkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xFDASBgNVBAoTC1Rp
Z2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1cml0eSA8c2lydEB0aWdlcmEuaW8+MSgw
JgYDVQQDEx9UaWdlcmEgRW50aXRsZW1lbnRzIENlcnRpZmljYXRlMIIBojANBgkq
hkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAwg3LkeHTwMi651af/HEXi1tpM4K0LVqb
5oUxX5b5jjgi+LHMPzMI6oU+NoGPHNqirhAQqK/k7W7r0oaMe1APWzaCAZpHiMxE
MlsAXmLVUrKg/g+hgrqeije3JDQutnN9h5oZnsg1IneBArnE/AKIHH8XE79yMG49
LaKpPGhpF8NoG2yoWFp2ekihSohvqKxa3m6pxoBVdwNxN0AfWxb60p2SF0lOi6B3
hgK6+ILy08ZqXefiUs+GC1Af4qI1jRhPkjv3qv+H1aQVrq6BqKFXwWIlXCXF57CR
hvUaTOG3fGtlVyiPE4+wi7QDo0cU/+Gx4mNzvmc6lRjz1c5yKxdYvgwXajSBx2pw
kTP0iJxI64zv7u3BZEEII6ak9mgUU1CeGZ1KR2Xu80JiWHAYNOiUKCBYHNKDCUYl
RBErYcAWz2mBpkKyP6hbH16GjXHTTdq5xENmRDHabpHw5o+21LkWBY25EaxjwcZa
Y3qMIOllTZ2iRrXu7fSP6iDjtFCcE2bFAgMBAAGjZzBlMA4GA1UdDwEB/wQEAwIF
oDATBgNVHSUEDDAKBggrBgEFBQcDAjAdBgNVHQ4EFgQUIY7LzqNTzgyTBE5efHb5
kZ71BUEwHwYDVR0jBBgwFoAUxZA5kifzo4NniQfGKb+4wruTIFowDQYJKoZIhvcN
AQELBQADggIBAAK207LaqMrnphF6CFQnkMLbskSpDZsKfqqNB52poRvUrNVUOB1w
3dSEaBUjhFgUU6yzF+xnuH84XVbjD7qlM3YbdiKvJS9jrm71saCKMNc+b9HSeQAU
DGY7GPb7Y/LG0GKYawYJcPpvRCNnDLsSVn5N4J1foWAWnxuQ6k57ymWwcddibYHD
OPakOvO4beAnvax3+K5dqF0bh2Np79YolKdIgUVzf4KSBRN4ZE3AOKlBfiKUvWy6
nRGvu8O/8VaI0vGaOdXvWA5b61H0o5cm50A88tTm2LHxTXynE3AYriHxsWBbRpoM
oFnmDaQtGY67S6xGfQbwxrwCFd1l7rGsyBQ17cuusOvMNZEEWraLY/738yWKw3qX
U7KBxdPWPIPd6iDzVjcZrS8AehUEfNQ5yd26gDgW+rZYJoAFYv0vydMEyoI53xXs
cpY84qV37ZC8wYicugidg9cFtD+1E0nVgOLXPkHnmc7lIDHFiWQKfOieH+KoVCbb
zdFu3rhW31ygphRmgszkHwApllCTBBMOqMaBpS8eHCnetOITvyB4Kiu1/nKvVxhY
exit11KQv8F3kTIUQRm0qw00TSBjuQHKoG83yfimlQ8OazciT+aLpVaY8SOrrNnL
IJ8dHgTpF9WWHxx04DDzqrT7Xq99F9RzDzM7dSizGxIxonoWcBjiF6n5
-----END CERTIFICATE-----`

const validLicenseToken = `eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJTZGUtWW1KV0pIcTNWREJ3IiwidGFnIjoiOGF2UVZEVHptME9mRGlVM2hfRlhYQSIsInR5cCI6IkpXVCJ9.nZK7QAqo3Jfa3LjUPtFHmw.Y_QN4NvAH0GmSMO9.bMxJ4AtoIF7uLShaSRXDL6cGXUq4kPVQjsh_dFndWud3fjSn1S7q09HcnTHKNmTupCsmStSB_lV363Ar9ShrV8WRebZeKZYqB4OOMzbj89fiTPPPA0AlqxrlEMnHyQYefyp_Kjy_eymHoaiZBzIiHZgKBDP4Dh6lhrMThUMaer6iKo_iMjtI-zRlAQ0_eMAcxRyiyFFIbUdUcy3uMz1UBQFLlm7YMslRBRzvf8gT__Ptihjll0KsxyGtivzYEwgOZ4lheWr1Af5nmslNQP9mR6MOF4TeSik3_yzq6TP3mgUol5HNCWyNB9-o-uqk9Wn0mQG3uy1ERJCMHNPKoUrvSTA5DiF7QeN8YR2h1C36ehcGLYi9L9jj1nT2JOO-uFagTdJeGH3lRQnF6RYkyfw-kitHuac8Ghte-YZNvXTmRBp7wT_L-X89-FcT4XveW5va0ChVOdl7aKAlkf8GDl3gZEkz22eVtZAnFEp6N-ApSasFA-3clqTulSlsLL4WkQ_Vin3lMEr11cYl2VFnQovLw3F30vrB2XEyjEiGRw86R4PRfxlYkHDgK7FhGgFb1UM4lmZUCycExzSYYpDd3oQBFEDR_fhZ0oq6Fp7SUeA6ypFL_Hph1NB0kf5emGnq4R2vr-T4BuM8YYe9Qa6OuVtf2U3o3ipCqdsAAHII0GhlLJWCs5ovNPOEbS_ky_0mLW8mvzfHnPqGL3HjZA2DZb0pZlqI7qbmwiO8N9iU5uZA0RsHJX_ClDF971m2LoUQAbe2I0rCtrhVhW5ljQPuJSTv0chLSDCPxk0-jEsTpA12dqK3eiyT-hWyTTXb2ZsivBdCIpOpVbZM2z2EvvEMvsN3lLCHGP61i0C0KPlze9DJE6vZVxAW1nzqRqi1IqU5mfZuoX8McbQiAEzBQ096hvypIygBmVTr17N8sXmHwJPNEdiLQ3pTLfyHBGZDyZlpy2Ej-4mG-Iegg8hjTkEm3q7QHzRL8hWTP0ff7MHT1NOXkSbN_bIpLmtjb75-we3Mc2cBPyyV96D89G16UUGkh0lzy0pLMMbz_ejSbKlULFkJJWRGn_58Hkw1ROBeREccg_F5B0wqLKY__jyq1OqrzcIZrxhUPLaWfoDKzSykDw.yeAEkIEd1wSwvuwgHs_6dw`
