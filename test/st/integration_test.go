package st_test

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
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
		voltronCmd  *exec.Cmd
		guardianCmd *exec.Cmd
	)

	It("Should change directory to bin folder", func() {
		err := os.Chdir("../../")
		Expect(err).ToNot(HaveOccurred())

	})

	It("should set env variables pointing to docker-image/ for certs", func() {
		err := os.Setenv("VOLTRON_CERT_PATH", "test")
		Expect(err).ToNot(HaveOccurred())

		err = os.Setenv("VOLTRON_TEMPLATE_PATH", "docker-image/voltron/templates/guardian.yaml.tmpl")
		Expect(err).ToNot(HaveOccurred())

		// disable toggle for authentication
		err = os.Setenv("VOLTRON_AUTHN_ON", "false")
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

			ExpectRespMsg(req, "[]")
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

			ExpectRespMsg(req, `[{"id":"ClusterA","displayName":"A"}]`)
		})

		It("Should delete ClusterA", func() {
			cluster, err := json.Marshal(&clusters.Cluster{ID: "ClusterA"})
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("DELETE", clustersEndpoint,
				bytes.NewBuffer(cluster))

			Expect(err).ToNot(HaveOccurred())

			ExpectRespMsg(req, "Deleted")
		})

		It("Should fail to delete nonexistant cluster", func() {
			cluster, err := json.Marshal(&clusters.Cluster{ID: "ClusterZ"})
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("DELETE", clustersEndpoint,
				bytes.NewBuffer(cluster))

			ExpectRespMsg(req, `Cluster id "ClusterZ" does not exist`)
		})
	})

	It("should set up guardian environment variables", func() {
		err := os.Setenv("GUARDIAN_PORT", "6666")
		Expect(err).ToNot(HaveOccurred())

		err = os.Setenv("GUARDIAN_VOLTRON_URL", "localhost:5566")
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should add a new test cluster to Voltron", func() {
		cluster, err := json.Marshal(&clusters.Cluster{ID: "TestCluster", DisplayName: "TestCluster"})
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("PUT", clustersEndpoint,
			bytes.NewBuffer(cluster))

		Expect(err).ToNot(HaveOccurred())

		resp, err := http.DefaultClient.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(200))

		// Save results to a file
		message, err := ioutil.ReadAll(resp.Body)

		Expect(err).NotTo(HaveOccurred())
		resp.Body.Close()

		err = ioutil.WriteFile("./test/st/tmp/guardian.yaml", message, 0644)
		Expect(err).NotTo(HaveOccurred())

		cmd1 := exec.Command("sh", "./scripts/dev/yaml-extract-creds.sh", "./test/st/tmp/guardian.yaml")
		err = cmd1.Run()

		Expect(err).NotTo(HaveOccurred())
	})

	It("should check for existance of generated guardian files", func() {
		// Test
		out, err := exec.Command("ls", "/tmp/").Output()
		Expect(err).NotTo(HaveOccurred())

		Expect(string(out)).To(ContainSubstring("guardian.crt"))
		Expect(string(out)).To(ContainSubstring("guardian.key"))
	})

	It("should set guardian environment variables", func() {
		err := os.Setenv("GUARDIAN_CERT_PATH", "/tmp/")
		Expect(err).NotTo(HaveOccurred())

		proxyTarget := fmt.Sprintf(`[{"path": "/api/", "url": "https://localhost:6443", ` +
			`"tokenPath":"./test/st/tmp/token", "caBundlePath":"./test/st/k8s-api-certs/k8s.crt"}]`)
		err = os.Setenv("GUARDIAN_PROXY_TARGETS", proxyTarget)
		Expect(err).NotTo(HaveOccurred())

	})

	It("Should start up guardian binary", func() {
		var startErr error
		guardianCmd = exec.Command("./bin/guardian")

		// Prints logs to OS' Stdout and Stderr
		guardianCmd.Stderr = os.Stderr

		// Switch STDOut into a pipe
		r, w, err := os.Pipe()
		Expect(err).ToNot(HaveOccurred())

		guardianCmd.Stdout = w

		go func() {
			startErr = guardianCmd.Start()

			// Blocking
			guardianCmd.Wait()
		}()

		// Check if startError
		Expect(startErr).ToNot(HaveOccurred())

		// Wait for guardian tunnel to be established. Listen to Stdout until we see connection establish
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			if contain := strings.Contains(text, "Tunnel: Accepting connections"); contain == true {
				break
			}
		}

	})

	Context("While Guardian is running", func() {
		It("Should send a request to nonexistant endpoint/", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")

			ExpectRequestResponse(req, expResponseCode(404))
		})

		It("Should send a request to test endpoint/target", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/api/v1/namespaces", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
		})

		It("Should send a request to wrong cluster id", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/api/v1", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "ClusterZ")
			resp, err := http.DefaultClient.Do(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})
	})

	It("Should kill the voltron and guardian processes", func() {
		err := voltronCmd.Process.Kill()
		Expect(err).ToNot(HaveOccurred())

		err = guardianCmd.Process.Kill()
		Expect(err).ToNot(HaveOccurred())
	})
})

type responseChecker func(*http.Response)

func expResponseMessage(expected string) responseChecker {
	return func(resp *http.Response) {
		message, err := ioutil.ReadAll(resp.Body)

		Expect(err).NotTo(HaveOccurred())
		resp.Body.Close()
		trimmedMsg := strings.TrimRight(string(message), "\n")
		Expect(trimmedMsg).To(Equal(expected))
	}
}

func expResponseCode(code int) responseChecker {
	return func(resp *http.Response) {
		Expect(resp.StatusCode).To(Equal(code))
	}
}

func ExpectRequestResponse(request *http.Request, checks ...responseChecker) {
	resp, err := http.DefaultClient.Do(request)
	Expect(err).ToNot(HaveOccurred())

	for _, check := range checks {
		check(resp)
	}
}

func ExpectRespMsg(r *http.Request, expected string) {
	ExpectRequestResponse(r, expResponseMessage(expected))
}
