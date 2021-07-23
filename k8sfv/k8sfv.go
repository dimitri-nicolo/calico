// Copyright (c) 2017, 2018-2021 Tigera, Inc. All rights reserved.
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
// limitations under the License.

package main

// XXX staticcheck disabled for the whole import as golangci-lint ignores
// specific directives, still an open issue:
// https://github.com/golangci/golangci-lint/issues/741
//
// SA1019 prometheus is using that lib and so need we
// github.com/golang/protobuf/proto is deprecated and fails lint

//nolint:staticcheck
import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/options"
)

// Global config - these are set by arguments on the ginkgo command line.
var (
	k8sServerEndpoint string // e.g. "http://172.17.0.2:6443"
	felixIP           string // e.g. "172.17.0.3"
	felixHostname     string // e.g. "b6fc45dcc1cb"
	prometheusPushURL string // e.g. "http://172.17.0.3:9091"
	codeLevel         string // e.g. "master"
)

// Prometheus metrics.
var (
	gaugeVecHeapAllocBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8sfv_heap_alloc_bytes",
		Help: "Occupancy measurement",
	}, []string{"process", "test_name", "test_step", "code_level"})
	gaugeVecOccupancyMeanBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8sfv_occupancy_mean_bytes",
		Help: "Mean occupancy for a test",
	}, []string{"process", "test_name", "code_level"})
	gaugeVecOccupancyIncreasePercent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8sfv_occupancy_increase_percent",
		Help: "% occupancy increase during a test",
	}, []string{"process", "test_name", "code_level"})
	gaugeVecTestResult = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8sfv_test_result",
		Help: "Test result, i.e. pass (1) or failure (0)",
	}, []string{"test_name", "code_level"})
)

var _ = BeforeSuite(func() {
	log.Info(">>> BeforeSuite <<<")
	log.WithFields(log.Fields{
		"k8sServerEndpoint": k8sServerEndpoint,
		"felixIP":           felixIP,
		"felixHostname":     felixHostname,
		"prometheusPushURL": prometheusPushURL,
		"codeLevel":         codeLevel,
	}).Info("Args")

	// Register Prometheus metrics.
	prometheus.MustRegister(gaugeVecHeapAllocBytes)
	prometheus.MustRegister(gaugeVecOccupancyMeanBytes)
	prometheus.MustRegister(gaugeVecOccupancyIncreasePercent)
	prometheus.MustRegister(gaugeVecTestResult)
})

// State that is common to most tests.
var (
	testName             string
	d                    deployment
	localFelixConfigured bool
)

var _ = JustBeforeEach(func() {
	log.Info(">>> JustBeforeEach <<<")
	testName = CurrentGinkgoTestDescription().FullTestText
})

var _ = AfterEach(func() {
	log.Info(">>> AfterEach <<<")

	// If we got as far as fully configuring the local Felix, check that the test finishes with
	// no left-over endpoints.
	if localFelixConfigured {
		Eventually(getNumEndpointsDefault(-1), "10s", "1s").Should(BeNumerically("==", 0))
	}

	// Store the result of each test in a Prometheus metric.
	result := float64(1)
	if CurrentGinkgoTestDescription().Failed {
		result = 0
	}
	gaugeVecTestResult.WithLabelValues(testName, codeLevel).Set(result)
})

var _ = AfterSuite(func() {
	log.Info(">>> AfterSuite <<<")
	if prometheusPushURL != "" {
		// Push metrics to Prometheus push gateway.

		err := push.New(
			prometheusPushURL,
			"k8sfv").Gatherer(prometheus.DefaultGatherer).Push()
		panicIfError(err)
	}

	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	panicIfError(err)
	fmt.Println("")
	for _, family := range metricFamilies {
		if strings.HasPrefix(*family.Name, "k8sfv") {
			fmt.Println(proto.MarshalTextString(family))
		}
	}
})

func initialize(k8sServerEndpoint string) (clientset *kubernetes.Clientset) {

	config, err := clientcmd.NewNonInteractiveClientConfig(*api.NewConfig(),
		"",
		&clientcmd.ConfigOverrides{
			ClusterDefaults: api.Cluster{
				Server:                k8sServerEndpoint,
				InsecureSkipTLSVerify: true,
			},
		},
		clientcmd.NewDefaultClientConfigLoadingRules()).ClientConfig()
	if err != nil {
		panic(err)
	}

	config.QPS = 10000
	config.Burst = 20000
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	Eventually(func() (err error) {
		calicoClient, err := client.New(apiconfig.CalicoAPIConfig{
			Spec: apiconfig.CalicoAPIConfigSpec{
				DatastoreType: apiconfig.Kubernetes,
				KubeConfig: apiconfig.KubeConfig{
					K8sAPIEndpoint:           k8sServerEndpoint,
					K8sInsecureSkipTLSVerify: true,
				},
			},
		})
		if err != nil {
			log.WithError(err).Warn("Waiting to create Calico client")
			return
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		err = calicoClient.EnsureInitialized(
			ctx,
			"v3.0.0-test",
			"v2.0.0-test",
			"felix-fv,typha", // Including typha to prevent config churn
		)
		if err != nil {
			log.WithError(err).Warn("Failed to initialize datastore")
		}

		// Create a license key for the test.  When we add node enforcement
		// we may need to update this with a suitable entitlement.
		licenseKey := apiv3.NewLicenseKey()
		licenseKey.Name = "default"
		licenseKey.Spec.Token = `eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ1U0lXUkNJd3dwS1U3ZTlFIiwidGFnIjoiaHNvVDF2VG9KNjk0UktpdTBjTEltdyIsInR5cCI6IkpXVCJ9.f78MAwbJiOvFIPUpmYnVeQ.gu8Lk2C1x550ib32.Q9Qbog_J12XFDOYg_DN4qTzcqz5r7O8j-F6epi0YmlDQUjXH__tmE8ScHB1JiQzCG7NnGx3zLDrN1GKI4iubSZX33MNLET_qr42pz9DqlDGfbIsQexGu1UoC5pRdAgE4Wjht3hO9bUlEKXT-NDUr_urqNNsPWdKYx9FeJEIT4QRr8MBYGoDQi_bTBfku7gfZZ8WKudyBWK6giRclsGhdvcTXgzg8Gmt8fqiWto_qUn0BEoBuzN9_HciQgudYEsUHEX1RgQae5KseqsIECeRT0OQK7TvFHQkhJ82E6bU5_QOffV2O59XmNpRC8fGIIzoGvHvFIiAGq4Zfja1CYi5E3rFhTTML183CcXm2XFbtAJac0td742EUhLxAB-UCk_r0kY9n8pfMKzhxwZ7CMHXhrdUFo4M_0tyYGz5T9pnYq04szzekbIePx8IRDwz54wKUfAD9KfqjZoOZaiYH9Uds8z8Oix8MfSMx-2g4BZGEFxGLLu2ZDD3nHgzAv0AjJudOj63jsUmBrcc3UeQ_Z1r2MM4L--zhDFwqTk8OiZSGt4mtOUuLFvW167IXHpTwoEhxp9_eNrtAdRGNv9S_7wImNaYkmI5jL7-jCxgZIIIESE4k-XC-GE2mfMCNlyxqF0XzLcYBUMjdVxlhQwpOVD7aRoL7FSblpNGdUFJLYr7wQJynVRS3lBtkUlyIzE2Ic9oYdfmawCDmUqP9FsR0aDPkVgvm8UoUhCo62Xxb95Eb1PUqJocpT0C6rCp4sleK-wpU5dmY-9mkwYF_n1HcGs8SjnKTG5lGHIwMn7A-Y6-CfprUHD2egsjFqu6s0ME3V9bZ6KM3YMjZeKVJTU4UZ6LjnKp-Bms8jeEleayRVDdeKCejzxjv7m2lsN1kcFTnjTwvAm7QC-99xrgfK7cbEYjyGZRoecmKzK3YM6mI9SmdJIUvhzKieB2ACk37oaXSgQN5udsSBFoa5SGzbKhxjvaXx46_Sm2FRHPMO5A5cCE6nmjlGG4qbz25UlHdv3ojITOqKFHEj2VMJXgBawIHxGWSGjY8t4PxC8PHy9wcNKhQRZdAtzjQ6IKBSytFpxMYAg.Uw0iQ6VZYMtWjz6K26PbOw`
		licenseKey.Spec.Certificate = `-----BEGIN CERTIFICATE-----
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
		_, err = calicoClient.LicenseKey().Create(ctx, licenseKey, options.SetOptions{})
		if _, ok := err.(cerrors.ErrorResourceAlreadyExists); ok {
			// Fine; suppress this 'error'.
			err = nil
		}

		return
	}, "60s", "2s").ShouldNot(HaveOccurred())

	log.Info("Initialization is Done.")
	return
}

func create1000Pods(clientset *kubernetes.Clientset, nsPrefix string) error {

	d = NewDeployment(clientset, 49, true)
	nsName := nsPrefix + "test"

	// Create 1000 pods.
	createNamespace(clientset, nsName, nil)
	log.Info("Creating pods:")
	for i := 0; i < 1000; i++ {
		createPod(clientset, d, nsName, podSpec{})
	}
	log.Info("Done")

	Eventually(getNumEndpointsDefault(-1), "30s", "1s").Should(
		BeNumerically("==", 1000),
		"Addition of pods wasn't reflected in Felix metrics",
	)

	return nil
}

func cleanupAll(clientset *kubernetes.Clientset, nsPrefix string) {
	defer cleanupAllNamespaces(clientset, nsPrefix)
	defer cleanupAllNodes(clientset)
	cleanupAllPods(clientset, nsPrefix)
}

func panicIfError(err error) {
	if err != nil {
		log.WithError(err).Error("About to panic...")
		panic(err)
	}
}
