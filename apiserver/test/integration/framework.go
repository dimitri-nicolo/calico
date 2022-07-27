// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package util

package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/projectcalico/calico/libcalico-go/lib/seedrng"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	restclient "k8s.io/client-go/rest"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/projectcalico/calico/apiserver/cmd/apiserver/server"
	"github.com/projectcalico/calico/apiserver/pkg/apiserver"
)

const defaultEtcdPathPrefix = ""

func init() {
	seedrng.EnsureSeeded()
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
	ro := genericoptions.NewRecommendedOptions(defaultEtcdPathPrefix, apiserver.Codecs.LegacyCodec(v3.SchemeGroupVersion))
	ro.Etcd.StorageConfig.Transport.ServerList = serverConfig.etcdServerList
	options := &server.CalicoServerOptions{
		RecommendedOptions: ro,
		DisableAuth:        true,
		StopCh:             stopCh,
	}
	options.RecommendedOptions.SecureServing.BindPort = securePort
	// Set this so that we avoid RecommendedOptions.CoreAPI's initialization from calling InClusterConfig()
	// and uses our fv kubeconfig instead.
	options.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath = os.Getenv("KUBECONFIG")

	options.EnableManagedClustersCreateAPI = serverConfig.enableManagedClusterCreateAPI
	options.ManagedClustersCACertPath = serverConfig.managedClustersCACertPath
	options.ManagedClustersCAKeyPath = serverConfig.managedClustersCAKeyPath
	options.ManagementClusterAddr = serverConfig.managementClusterAddr

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

	licenseClient := clientset.ProjectcalicoV3().LicenseKeys()
	_ = licenseClient.Delete(context.Background(), "default", metav1.DeleteOptions{})

	if serverConfig.applyTigeraLicense {
		validLicenseKey := getLicenseKey("default", validLicenseCertificate, enterpriseToken)
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

func getLicenseKey(name, certificate, token string) *v3.LicenseKey {
	licenseKey := &v3.LicenseKey{ObjectMeta: metav1.ObjectMeta{Name: name}}

	licenseKey.Spec.Certificate = certificate
	licenseKey.Spec.Token = token

	return licenseKey
}

func sortedKeys(set map[string]bool) []string {
	var keys []string
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func createEnterprise(client calicoclient.Interface, ctx context.Context) error {
	enterpriseValidLicenseKey := getLicenseKey("default", validLicenseCertificate, enterpriseToken)
	_, err := client.ProjectcalicoV3().LicenseKeys().Create(ctx, enterpriseValidLicenseKey, metav1.CreateOptions{})
	return err
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

const enterpriseToken = `eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3VVZPcmpuczY0OEhDWnVtIiwidGFnIjoiTkJ2QllkLXI2X0NycWpfNVBXQm1wZyIsInR5cCI6IkpXVCJ9.rPfPoiuPMQGN-Ff2ybYqAA.RZ4vKYOgvU1D-MOQ.6S71luU6a9b7xD5aR4XHpHXwSrmSTBcuwGM0bq3l01A-Cif0nmJ7yFKfFaWa01Kts3vA4MibqT6NgxueA48oEQoDZqOdgeJgIUYezul7EioBNSjiZV87UIn3A3VOUwToatc1EaxlGs6KsI_E8wBrULYuKbMP8Fe6ir2hz1JZ6l67EwgBNYHDy1WTVYLapuJx2BYIXaxEEoaKUoYSFXQJ2hO-CjijC7gR10Y_raFJ_GPP0Bwo8iohLl42OocLPjK_JhZVm1FzGxZn_LHSnMdxWnBRXw6_Jt1K_39-p4eKfbV6zJ8vPz5eR1eSA06TY3MJljxnj5phuBsvqB5wsX5f0kaBVDwp0NQLtpuaTFqDp0hvG3rAwGQyqq-HjhXrYivN5QnbX70sL4fosFUwwwm4ZIpnPJoDUGfnrB3tvtVMMnP0I_ADxF75Dm_eEm3MEQI9IuhWO1JWxSJM4KV20BiL_UXfX17juxYmeQOkOQ9T2LX4nm0lnxpYksu0eUud2Ak8bUkFs4L7cmxrpfOV_gRPbW-38wl72dTNE58BsbpWrbu8fXcupvuLeGqYbb4bOXwsD7rPVIBFBJngSzPXmCCgZBm7_DYwcjzVMIkNm2zu6wTUd6FluuLgeyHJbR1WYKKMxpnn5__vc-lgPDcSqva36FQrQf3b5sef5uopM26NI3_SngU8bjAEFcMM73_-nLalKqVlq-MlwCCw_nBhK-_W5ZwIRzFfOAborYAgifpo6ckifk8i87HVRxUd96ynxjlZ8c4vm3I4bOvk65kXbarFxpD9V8TVdfpfXAMtlWG3DVAW4DqRxWbpEANbYCMW5K7YkEhwFw58sa6bKO_uCUEC0uX8bYmmAZThPjSLNdXf2cn__zCy3EgX6ePdvkyd12iMe8_a9TGT2EbrzipBxfIbDHc0fc8lO0Gku1UEuIlCmSEEEy7AmlZzhaN6ZkbdFf0QeAhDXPmRri0CUWekCQbcrIZNk97LZsjFpr_NICBNKbaLfh0Yw-cqKNahlhdzpoT1jvC_g8f_9LTafgj12xOLHSwTHAA06iBherrGb6UnRtPGDZ_RmzoPwVSCSSft.s49CPsULUtc_hD1R6RdcuQ`

const cloudProToken = `eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJOQURNdTRzUkM3WDNyaHBuIiwidGFnIjoiN0IyaW9vNDUtbFhFVnlIdEFydWo1dyIsInR5cCI6IkpXVCJ9.FWo32DjaHiyloaR5D77mKQ.hU7EyT6-sz07kv4W.9pqdkXXzBCbkfWOaPyox57s5q4N_ePjJiR3bmikN0TkZmE-ed67xItTw44yBXJaSpMB3ddRVumvQ8k0gcN2IfVhtWNjiI5eXe8UJ7pLG2Y-D0dG-4JJMlYYHh8Zn02_-J6MyjD1NCqqPHnkNvuBLITrNdLH923yRi6VhhqgxNiaxZPuU3dxVEDKuLowiDn660NSBtNg2VUfMWadDwCOJCWZIjOPfdLanQNecuIHcu-aHpwB5xp7cbeT3uD4547-gO5KOzpWji79AAkfrfS5ZPNN1gL7QEEXhHeS_eqVVoJO5p2XYsAYHwAhQQSWe4qkg6W-v27M5ZRYTK5BM9R4JtJJ22nfwfBSnGBFIy-3rVAZ2NkQSXADbtLB2pqpIDYZWU_v2_kyaNNvKwWgObWg53ARweqjeYW-O64tIpajdyhdXkALHVto-GQBKXsUm2jL3YkcVX8ytyO2ldo8_3KMuQHS-CzT6lGkyMe6xFAgwtCj_xfAZzjZBZHtjmGpa2CmIYpmT_TOuFgVTRxCU2UlB9vnut57334OT7DFDDia6DHHidKxEZxuSbQi5077Ao7Nh7n6h7tu6QXBew5nV3KFUWF8590IP5Ryr7MID_cQ2m1KAOuQhdU3sEz4hrYwrIjUsgev_Wd4ipQby51rL03cfvYpc0WQ3YF_axVmA4XwmiotVjxnBdiEQonS5JTsF42PakRUVtR97gTvrt479XU8EMeamz7glfYh9o0FScrE8b5kJrHP9Xu1WRyZH_EzAIGk9hBCsz4XjuHI2HdbPGOr7IYwzgEWxO-JgLP3VsuSOwEBW2wlSBuKiC-wjnJSmRnQzsz5a-rTvslgew3Gx9N1nqwzLs5VvKUf9Lj3SpyvamF2c_apAVvg6F70Hb9x2g8XvfHvbomZMKHjqugLeZXwzgmkQHHJzswp-7krIDHhEKszxTxOKEmGzAH4IJSx--1n8VfsjXSld7QFtLf4tiB5CDeaaMCnREpLkPo-GGbjd_DHPQoKqVKIpCP2HpIrPic_XCbFPi2amL7joHRUJS8UOvU2JZDkCglzOkDvfhidYHV4GqLKZIXUVOufNHxwGbwCwQGNSbDIKzi132pjJ1gU_eLdZzZpWzeU0w8CpTxwevksQwiTfx7EkiNlVVcUwzMqaZ9akoY8h2l53GqnjqwFnIYtWXe0LZpuqf0xYg6DUdE1XhTuqWjJAa3fhQH-clagbLUx_3LKgFC7nlhYRyheSDsVvhcH2hUoPpKljAv7lKeylYIBuNpRPNKuoI9n5HR8jNwX6lDsZVx84Lj8kccppMNDE3AZEHa0kEH2K9gOluIzaQTChidkbYOs8zRkg3sHVCZAmIfM2i5DrFLE_b8xLlgx9MrX4JgUAe3YOg-pYpEFJ4fz1nIpKaGfeVs9Tyy9YvTI_VLgt-zHpCR9LuG053UbfqqN7mvJP7hDuZGL_kW0Xdvv0WMsljO_g_yD1qPvSOOTdjIoOZbq2U7KL2dT0kZfSwWLdQfIWkPOc92qHrn16ZRbwHGQXhbcrWc88dDPpJHp1El2M4v76ieDiLXRPLeczLlq4HVC91pDzzwKrL6g2mt5nlEnlPmoBENAAJphOcWuCoNGfsYpJN7ttsEh96AZk7tCdi1F3LQ1-kRZH_C6LurXI2qU.9I81zcXLo78WfhoLpzgafQ`
