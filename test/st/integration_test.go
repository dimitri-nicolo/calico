package st_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/clusters"
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
	// Disable SSL Certificate Verification
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

var _ = Describe("Integration Tests", func() {

	var (
		voltronCmd *exec.Cmd
	)

	It("Should change directory to bin folder", func() {
		err := os.Chdir("../../")
		Expect(err).ToNot(HaveOccurred())

	})

	It("should set env variables pointing to docker-image/ for certs", func() {
		err := os.Setenv("VOLTRON_CERT_PATH", "docker-image")
		Expect(err).ToNot(HaveOccurred())

		err = os.Setenv("VOLTRON_TEMPLATE_PATH", "manifests/guardian.yaml")
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should fail to ping cluster endpoint", func() {
		timeout := time.Duration(5 * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		_, err := client.Get("https://localhost:5555")
		Expect(err).To(HaveOccurred())
	})

	It("Should start up voltron binary", func() {
		var startErr error
		voltronCmd = exec.Command("./bin/voltron")

		// Prints logs to OS' Stdout and Stderr
		voltronCmd.Stdout = os.Stdout
		voltronCmd.Stderr = os.Stderr

		go func() {
			startErr = voltronCmd.Start()

			// Blocking
			voltronCmd.Wait()
		}()

		// Wait for Server to start up
		time.Sleep(2 * time.Second)

		// Check if startError
		Expect(startErr).ToNot(HaveOccurred())

	})

	clustersEndpoint := "https://localhost:5555/voltron/api/clusters"

	Context("While Voltron is running", func() {
		It("Should successfully ping cluster endpoint, no clusters added", func() {
			req, err := http.NewRequest("GET", clustersEndpoint, nil)
			Expect(err).ToNot(HaveOccurred())

			ExpectRequestResponse(req, "[]")
		})

		It("Should add a cluster", func() {
			cluster, err := json.Marshal(&clusters.Cluster{ID: "ClusterA", DisplayName: "A"})
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("PUT", clustersEndpoint,
				bytes.NewBuffer(cluster))

			Expect(err).ToNot(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

		})

		It("Should List one cluster", func() {
			req, err := http.NewRequest("GET", clustersEndpoint, nil)
			Expect(err).ToNot(HaveOccurred())

			ExpectRequestResponse(req, `[{"id":"ClusterA","displayName":"A"}]`)
		})

		It("Should delete ClusterA", func() {
			cluster, err := json.Marshal(&clusters.Cluster{ID: "ClusterA"})
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("DELETE", clustersEndpoint,
				bytes.NewBuffer(cluster))

			Expect(err).ToNot(HaveOccurred())

			ExpectRequestResponse(req, "Deleted")
		})

		It("Should fail to delete nonexistant cluster", func() {
			cluster, err := json.Marshal(&clusters.Cluster{ID: "ClusterZ"})
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("DELETE", clustersEndpoint,
				bytes.NewBuffer(cluster))

			ExpectRequestResponse(req, `Cluster id "ClusterZ" does not exist`)
		})
	})

	It("Should kill the process", func() {
		err := voltronCmd.Process.Kill()
		Expect(err).ToNot(HaveOccurred())
	})
})

func ExpectRequestResponse(request *http.Request, expected string) {
	resp, err := http.DefaultClient.Do(request)
	Expect(err).ToNot(HaveOccurred())
	message, err := ioutil.ReadAll(resp.Body)

	Expect(err).NotTo(HaveOccurred())
	resp.Body.Close()
	trimmedMsg := strings.TrimRight(string(message), "\n")
	Expect(trimmedMsg).To(Equal(expected))
}
