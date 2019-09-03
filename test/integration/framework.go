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
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	restclient "k8s.io/client-go/rest"

	"github.com/tigera/calico-k8sapiserver/cmd/apiserver/server"
	_ "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/install"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/calico-k8sapiserver/pkg/apiserver"
	calicoclient "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"
)

const defaultEtcdPathPrefix = ""

func init() {
	rand.Seed(time.Now().UnixNano())
}

type TestServerConfig struct {
	etcdServerList []string
	emptyObjFunc   func() runtime.Object
}

// NewTestServerConfig is a default constructor for the standard test-apiserver setup
func NewTestServerConfig() *TestServerConfig {
	return &TestServerConfig{
		etcdServerList: []string{"http://localhost:2379"},
	}
}

func withConfigGetFreshApiserverAndClient(
	t *testing.T,
	serverConfig *TestServerConfig,
) (calicoclient.Interface,
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
	// start the server in the background
	go func() {
		ro := genericoptions.NewRecommendedOptions(defaultEtcdPathPrefix, apiserver.Codecs.LegacyCodec(v3.SchemeGroupVersion))
		ro.Etcd.StorageConfig.ServerList = serverConfig.etcdServerList
		options := &server.CalicoServerOptions{
			RecommendedOptions: ro,
			DisableAuth:        true,
			StopCh:             stopCh,
		}
		options.RecommendedOptions.SecureServing.BindPort = securePort

		//options.RecommendedOptions.SecureServing.BindAddress=
		if err := server.RunServer(options); err != nil {
			close(serverFailed)
			t.Fatalf("Error in bringing up the server: %v", err)
		}
	}()

	if err := waitForApiserverUp(secureAddr, serverFailed); err != nil {
		t.Fatalf("%v", err)
	}

	config := &restclient.Config{}
	config.Host = secureAddr
	config.Insecure = true
	clientset, err := calicoclient.NewForConfig(config)
	if nil != err {
		t.Fatal("can't make the client from the config", err)
	}
	return clientset, config, shutdownServer
}

func getFreshApiserverAndClient(
	t *testing.T,
	newEmptyObj func() runtime.Object,
) (calicoclient.Interface, func()) {
	serverConfig := &TestServerConfig{
		etcdServerList: []string{"http://localhost:2379"},
		emptyObjFunc:   newEmptyObj,
	}
	client, _, shutdownFunc := withConfigGetFreshApiserverAndClient(t, serverConfig)
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
				glog.Infof("Waiting for : %#v", serverURL)
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				c := &http.Client{Transport: tr}
				_, err := c.Get(serverURL)
				if err == nil {
					glog.Infof("Found server after %v tries and duration %v",
						tries, time.Since(startWaiting))
					return true, nil
				}
				tries++
				return false, nil
			}
		},
	)
}
