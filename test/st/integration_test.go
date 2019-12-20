package st_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/voltron/internal/pkg/bootstrap"
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
		voltronWg   sync.WaitGroup
		guardianCmd *exec.Cmd
		guardianWg  sync.WaitGroup
		k8s         *bootstrap.K8sClient
	)

	It("Should change directory to bin folder", func() {
		err := os.Chdir("../../")
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should configure the K8s client", func() {
		k8s, _ = bootstrap.ConfigureK8sClient("./test/st/k8s-api-certs/kube.config")
	})

	It("should set env variables pointing to docker-image/ for certs", func() {
		err := os.Setenv("VOLTRON_CERT_PATH", "test")
		Expect(err).ToNot(HaveOccurred())

		err = os.Setenv("VOLTRON_TEMPLATE_PATH", "docker-image/voltron/templates/guardian.yaml.tmpl")
		Expect(err).ToNot(HaveOccurred())

		err = os.Setenv("VOLTRON_K8S_CONFIG_PATH", "./test/st/k8s-api-certs/kube.config")
		Expect(err).ToNot(HaveOccurred())

		// do not use the default proxy, not needed
		err = os.Setenv("VOLTRON_DEFAULT_K8S_PROXY", "false")
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
		voltronCmd = exec.Command("./bin/voltron")

		// Prints logs to OS' Stdout and Stderr
		voltronCmd.Stdout = os.Stdout
		voltronCmd.Stderr = os.Stderr

		errC := make(chan error)

		voltronWg.Add(1)
		go func() {
			defer voltronWg.Done()
			errC <- voltronCmd.Start()

			// Blocking
			voltronCmd.Wait()
		}()

		Expect(<-errC).ToNot(HaveOccurred())

	})

	clustersEndpoint := "https://localhost:5555/voltron/api/clusters"

	Context("While Voltron is running", func() {
		It("Should eventually successfully ping cluster endpoint, no clusters added", func() {
			req, err := http.NewRequest("GET", clustersEndpoint, nil)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return false
				}
				return resp.StatusCode == 200
			}).Should(BeTrue())

			ExpectRespMsg(req, "[]")
		})

		It("Should add a cluster", func() {
			cluster, err := json.Marshal(&clusters.ManagedCluster{ID: "ClusterA"})
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
			cluster, err := json.Marshal(&clusters.ManagedCluster{ID: "ClusterA"})
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("DELETE", clustersEndpoint,
				bytes.NewBuffer(cluster))

			Expect(err).ToNot(HaveOccurred())

			ExpectRespMsg(req, "Deleted")
		})

		It("Should fail to delete nonexistent cluster", func() {
			cluster, err := json.Marshal(&clusters.ManagedCluster{ID: "ClusterZ"})
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
		cluster, err := json.Marshal(&clusters.ManagedCluster{ID: "TestCluster"})
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

	It("should check for existence of generated guardian files", func() {
		// Test
		out, err := exec.Command("ls", "/tmp/").Output()
		Expect(err).NotTo(HaveOccurred())

		Expect(string(out)).To(ContainSubstring("managed-cluster.crt"))
		Expect(string(out)).To(ContainSubstring("managed-cluster.key"))
	})

	It("should set guardian environment variables", func() {
		err := os.Setenv("GUARDIAN_CERT_PATH", "/tmp/")
		Expect(err).NotTo(HaveOccurred())

		proxyTarget := fmt.Sprintf(`[{"path": "/api/", "url": "https://localhost:6443", ` +
			`"tokenPath":"./test/st/tmp/token", "caBundlePath":"./test/st/k8s-api-certs/k8s.crt"},
			{"path": "/REWRITE/", "url": "https://localhost:6443", ` +
			`"pathRegexp":"REWRITE", "pathReplace":"api", ` +
			`"tokenPath":"./test/st/tmp/token", "caBundlePath":"./test/st/k8s-api-certs/k8s.crt"},
			{"path": "/apis/", "url": "https://localhost:6443", ` +
			`"tokenPath":"./test/st/tmp/token", "caBundlePath":"./test/st/k8s-api-certs/k8s.crt"}]`)
		err = os.Setenv("GUARDIAN_PROXY_TARGETS", proxyTarget)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should start up guardian binary", func() {
		var startErr error
		guardianCmd = exec.Command("./bin/guardian")

		// Prints logs to OS' Stdout and Stderr
		guardianCmd.Stdout = os.Stdout
		guardianCmd.Stderr = os.Stderr

		guardianWg.Add(1)
		go func() {
			defer guardianWg.Done()
			startErr = guardianCmd.Start()

			// Blocking
			guardianCmd.Wait()
		}()

		// Check if startError
		Expect(startErr).ToNot(HaveOccurred())

	})

	guardianTest := func() {
		It("Should eventually send a request to test endpoint/target using Jane's credentials", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/api/v1/namespaces", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenJane")
			Eventually(func() bool {
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return false
				}
				return resp.StatusCode == 200
			}, "1s", "200ms").Should(BeTrue())
		})

		It("Should send a request through a REWRITE target", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/REWRITE/v1/namespaces", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenJane")

			ExpectRequestResponse(req, expResponseCode(200))
		})

		It("Should send a request to nonexistent endpoint/ using Jane's credentials", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenJane")

			ExpectRequestResponse(req, expResponseCode(404))
		})

		It("Should send a request to nonexistent endpoint/ without authentication", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")

			ExpectRequestResponse(req, expResponseCode(401))
		})

		It("Should send a request to wrong cluster id using Jane's credentials", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/api/v1", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "ClusterZ")
			req.Header.Add("Authorization", "Bearer tokenJane")
			resp, err := http.DefaultClient.Do(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("Should send a request to wrong cluster id without authentication", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/api/v1", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "ClusterZ")
			resp, err := http.DefaultClient.Do(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("Should send a request to the health endpoint", func() {
			resp, err := http.Get("http://localhost:9080/health")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})

		It("Should not allow Jane to access network policies", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/apis/networking.k8s.io/v1/networkpolicies", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenJane")

			ExpectRequestResponse(req, expResponseCode(403))
		})

		It("Should define a role to read network policies", func() {
			policy := v1.PolicyRule{
				APIGroups: []string{"networking.k8s.io"},
				Verbs:     []string{"get", "watch", "list"},
				Resources: []string{"networkpolicies"},
			}
			role := &v1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "read-network-policies",
				},
				Rules: []v1.PolicyRule{policy},
			}
			_, err := k8s.RbacV1().ClusterRoles().Create(role)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should bind Jane to read-network-policies role", func() {
			binding := &v1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "read-network-policies-jane-binding",
				},
				RoleRef: v1.RoleRef{
					Kind: "ClusterRole",
					Name: "read-network-policies",
				},
				Subjects: []v1.Subject{{Kind: "User", Name: "Jane"}},
			}
			_, err := k8s.RbacV1().ClusterRoleBindings().Create(binding)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should allow Jane to access network polices", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/apis/networking.k8s.io/v1/networkpolicies", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenJane")

			ExpectRequestResponse(req, expResponseCode(200))
		})

		It("Should delete Jane's binding to read-network-policies role", func() {
			err := k8s.RbacV1().ClusterRoleBindings().Delete("read-network-policies-jane-binding", &metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should not allow Jane to access network polices after deleting role", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/apis/networking.k8s.io/v1/networkpolicies", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenJane")

			ExpectRequestResponse(req, expResponseCode(403))
		})

		It("Should bind developers to read-network-policies role", func() {
			binding := &v1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "read-network-policies-developers-binding",
				},
				RoleRef: v1.RoleRef{
					Kind: "ClusterRole",
					Name: "read-network-policies",
				},
				Subjects: []v1.Subject{{Kind: "Group", Name: "developers"}},
			}
			_, err := k8s.RbacV1().ClusterRoleBindings().Create(binding)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should allow group developers to access network policies", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/apis/networking.k8s.io/v1/networkpolicies", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenDev")

			ExpectRequestResponse(req, expResponseCode(200))
		})

		It("Should delete developers's binding to read-network-policies role", func() {
			err := k8s.RbacV1().ClusterRoleBindings().Delete("read-network-policies-developers-binding", &metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should not allow group developers to access network policies after deleting binding", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/apis/networking.k8s.io/v1/networkpolicies", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenDev")

			ExpectRequestResponse(req, expResponseCode(403))
		})

		It("Should not allow Bob to read namespaces", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/apis/v1/namespaces", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenBob")

			ExpectRequestResponse(req, expResponseCode(401))
		})

		It("Should authenticate using Bearer in favour of Basic tokens", func() {
			req, err := http.NewRequest("GET", "https://localhost:5555/apis/v1/namespaces", nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("x-cluster-id", "TestCluster")
			req.Header.Add("Authorization", "Bearer tokenBob")
			req.Header.Add("Authorization", "Basic tokenJane")

			ExpectRequestResponse(req, expResponseCode(401))
		})

		It("Should delete role read-network-policies ", func() {
			err := k8s.RbacV1().ClusterRoles().Delete("read-network-policies", &metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

	}

	Context("While Guardian is running", guardianTest)

	When("voltron dies (tunnel breaks)", func() {
		It("should make guardian exit", func(done Done) {
			err := voltronCmd.Process.Kill()
			Expect(err).ToNot(HaveOccurred())
			voltronWg.Wait()
			guardianWg.Wait()
			close(done)
		})
	})

	When("voltron restarts", func() {
		It("Should restart voltron", func() {
			voltronCmd = exec.Command("./bin/voltron")

			// Prints logs to OS' Stdout and Stderr
			voltronCmd.Stdout = os.Stdout
			voltronCmd.Stderr = os.Stderr

			errC := make(chan error)

			voltronWg.Add(1)
			go func() {
				defer voltronWg.Done()
				errC <- voltronCmd.Start()

				// Blocking
				voltronCmd.Wait()
			}()

			Expect(<-errC).ToNot(HaveOccurred())
		})

		It("Should eventually get a list of clusters", func() {
			// to make sure that voltron is up again
			req, err := http.NewRequest("GET", clustersEndpoint, nil)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return false
				}
				return resp.StatusCode == 200
			}).Should(BeTrue())

			// TODO when proper recovery is implemented, the list should
			// eventually contain all previously registered clusters
			ExpectRespMsg(req, "[]")
		})

		It("Should start up guardian again", func() {
			var startErr error
			guardianCmd = exec.Command("./bin/guardian")

			// Prints logs to OS' Stdout and Stderr
			guardianCmd.Stdout = os.Stdout
			guardianCmd.Stderr = os.Stderr

			guardianWg.Add(1)
			go func() {
				defer guardianWg.Done()
				startErr = guardianCmd.Start()

				// Blocking
				guardianCmd.Wait()
			}()

			// Check if startError
			Expect(startErr).ToNot(HaveOccurred())

		})

		Context("While Guardian is running", guardianTest)
	})

	It("Should kill the voltron and guardian processes", func(done Done) {
		err := voltronCmd.Process.Kill()
		Expect(err).ToNot(HaveOccurred())

		err = guardianCmd.Process.Kill()
		Expect(err).ToNot(HaveOccurred())

		voltronWg.Wait()
		guardianWg.Wait()
		close(done)
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
