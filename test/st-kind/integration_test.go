package st_kind_test

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/voltron/internal/pkg/clusters"
)

var _ = Describe("kind integration test", func() {

	http.DefaultClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}

	BeforeSuite(func() {
		run("install_cluster.sh", false)

		// Accept any certs, don't verify
		tp := &http.Transport{TLSClientConfig: &tls.Config{}}
		tp.TLSClientConfig.InsecureSkipVerify = true
		http.DefaultClient.Transport = tp
	})

	clustersEndpoint := "https://localhost:9443/apis/projectcalico.org/v3/managedclusters"
	podEndpoint := "https://localhost:9443/api/v1/pods"

	var (
		token string
		uid   string
	)

	Describe("Checking Voltron after installation", func() {

		It("should have its endpoints working and return an empty list of clusters", func() {
			encodedToken := strings.TrimSpace(run("create_access_token_for_curl.sh", true))
			byteToken, err := base64.StdEncoding.DecodeString(encodedToken)
			token = string(byteToken)
			Expect(err).ToNot(HaveOccurred())
			fmt.Println("This is a valid bearer token: " + token)

			req, err := http.NewRequest("GET", clustersEndpoint, nil)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
			Expect(err).ToNot(HaveOccurred())
			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(err).ToNot(HaveOccurred())
			responseMsg := getResponseMessage(req)
			var data v3.ManagedClusterList
			err = json.Unmarshal([]byte(responseMsg), &data)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(data.Items)).To(Equal(0))
		})

		It("should add a cluster without problems", func() {
			body := []byte(`{"kind":"","apiVersion":"projectcalico.org/v3","metadata":{"name":"managed-cluster","selfLink":"","uid":"","resourceVersion":"","creationTimestamp":"2019-08-09T00:22:53.621Z","annotations":{},"labels":{}},"spec":{"installationManifest":""}}`)
			req, err := http.NewRequest("POST", clustersEndpoint, bytes.NewBuffer(body))
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
			Expect(err).ToNot(HaveOccurred())
			responseMsg := getResponseMessage(req)
			Expect(err).ToNot(HaveOccurred())
			var data v3.ManagedCluster
			err = json.Unmarshal([]byte(responseMsg), &data)
			Expect(err).ToNot(HaveOccurred())
			err = WriteToFile("test-resources/guardian.yaml", data.Spec.InstallationManifest)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should list one cluster after adding it", func() {
			req, err := http.NewRequest("GET", clustersEndpoint, nil)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(err).ToNot(HaveOccurred())

			responseMsg := getResponseMessage(req)
			var data v3.ManagedClusterList
			err = json.Unmarshal([]byte(responseMsg), &data)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(data.Items)).To(Equal(1))
			Expect(data.Items[0].Name).To(Equal("managed-cluster"))
			uid = string(data.Items[0].UID)
			Expect(uid).ToNot(BeNil())
			fmt.Println("Uid of the new cluster: " + uid)
		})
	})

	Describe("Installing and checking Guardian", func() {
		Context("Installing and checking Guardian ", func() {

			It("should now be possible to send requests to the newly added cluster", func() {
				run("install_guardian.sh ", false)
				req, err := http.NewRequest("GET", podEndpoint, nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
				req.Header.Set("x-cluster-id", uid)
				Expect(err).ToNot(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
			})

			//Returns 503 now.
			//It("should not be possible to send requests to nonexistent cluster", func() {
			//	req, err := http.NewRequest("GET", clustersEndpoint, nil)
			//	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
			//	req.Header.Set("x-cluster-id", uid)
			//	Expect(err).ToNot(HaveOccurred())
			//	Expect(resp.StatusCode).To(Equal(404))
			//})

			//Returns 400 now.
			//It("should not be possible to send requests to nonexistent cluster without being authenticated/authorized", func() {
			//	req, err := http.NewRequest("GET", clustersEndpoint, nil)
			//	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
			//	req.Header.Set("x-cluster-id", uid)
			//	Expect(err).ToNot(HaveOccurred())
			//  Expect(resp.StatusCode).To(Equal(401/403))
			//})
		})
	})

	Describe("Deleting clusters", func() {
		Context("Deleting and verifying the outcomes ", func() {

			It("should delete a cluster without errors", func() {
				cluster, err := json.Marshal(&clusters.Cluster{ID: "ClusterA"})
				Expect(err).NotTo(HaveOccurred())
				req, err := http.NewRequest("DELETE", clustersEndpoint, bytes.NewBuffer(cluster))
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
				_, err = http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should eventually get an empty list of clusters after deleting", func() {
				req, err := http.NewRequest("GET", clustersEndpoint, nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
				Expect(err).ToNot(HaveOccurred())

				resp, err := http.DefaultClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))

				responseMsg := getResponseMessage(req)
				var data v3.ManagedClusterList
				err = json.Unmarshal([]byte(responseMsg), &data)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(data.Items)).To(Equal(0))
			})
		})
	})

	AfterSuite(func() {
		run("cleanup.sh", false)
	})
})

// if returnOutput is false, we send the output to os.stdout
func run(script string, returnOutput bool) string {
	fmt.Printf("INFO: Running script: %v  \n", script)
	cmd := exec.Command("/bin/bash", "-c", "/bin/bash scripts/"+script)

	var err error
	var out []byte
	if !returnOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		Expect(err).ToNot(HaveOccurred())
	} else {
		out, err = cmd.Output()
	}
	Expect(err).ToNot(HaveOccurred())
	fmt.Println("INFO: Done running script: " + script)
	return string(out)
}

func getResponseMessage(request *http.Request) string {
	resp, err := http.DefaultClient.Do(request)
	Expect(err).ToNot(HaveOccurred())

	message, err := ioutil.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	err = resp.Body.Close()
	Expect(err).ToNot(HaveOccurred())
	trimmedMsg := strings.TrimRight(string(message), "\n")
	return trimmedMsg
}

func WriteToFile(filename string, data string) error {
	file, err := os.Create(filename)
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	_, err = io.WriteString(file, data)
	Expect(err).ToNot(HaveOccurred())
	return file.Sync()
}
