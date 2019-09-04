package st_kind_test

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	v1 "k8s.io/api/rbac/v1"

	log "github.com/sirupsen/logrus"

	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/voltron/internal/pkg/clusters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var tokenType string

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
	flag.StringVar(&tokenType, "token-type", "bearer", "Can have one of the following values: bearer or basic")
	log.Infof("Using tokenType=%v", &tokenType)
}

var _ = Describe("ST tests", func() {

	http.DefaultClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}

	BeforeSuite(func() {
		run(fmt.Sprintf("install_cluster.sh %s", tokenType), false)

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

	Describe("Proxying requests through Guardian", func() {

		It("should install Guardian", func() {
			run("install_guardian.sh ", false)
		})

		Context("when binding Jane to read pods", func() {

			It("should define a role to read pods", func() {
				policy := v1.PolicyRule{
					APIGroups: []string{""},
					Verbs:     []string{"get", "watch", "list"},
					Resources: []string{"pods"},
				}
				role := &v1.ClusterRole{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "read-pods",
					},
					Rules: []v1.PolicyRule{policy},
				}

				data, err := json.Marshal(role)
				Expect(err).NotTo(HaveOccurred())
				add(data, "https://localhost:9443/apis/rbac.authorization.k8s.io/v1/clusterroles", token)
			})

			It("should bind Jane to read-pods role", func() {
				binding := &v1.ClusterRoleBinding{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "read-pods-jane-binding",
					},
					RoleRef: v1.RoleRef{
						Kind: "ClusterRole",
						Name: "read-pods",
					},
					Subjects: []v1.Subject{{Kind: "User", Name: "Jane"}},
				}

				data, err := json.Marshal(binding)
				Expect(err).NotTo(HaveOccurred())
				add(data, "https://localhost:9443/apis/rbac.authorization.k8s.io/v1/clusterrolebindings", token)
			})

			It("should send a request to test endpoint/target using Jane's credentials", func() {
				req, err := http.NewRequest("GET", podEndpoint, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Set("Authorization", genToken("Jane"))
				req.Header.Set("x-cluster-id", uid)
				resp, err := http.DefaultClient.Do(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
			})

			It("should deny send a request to test endpoint/target using Bob's credentials", func() {
				req, err := http.NewRequest("GET", podEndpoint, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Set("Authorization", genToken("Bob"))
				req.Header.Set("x-cluster-id", uid)
				resp, err := http.DefaultClient.Do(req)

				Expect(err).ToNot(HaveOccurred())
				if tokenType == "bearer" {
					Expect(resp.StatusCode).To(Equal(401))
				} else {
					Expect(resp.StatusCode).To(Equal(403))
				}
			})

			/* TODO: https://tigera.atlassian.net/browse/SAAS-283
			It("should send a request through a REWRITE target", func() {
				req, err := http.NewRequest("GET", "https://localhost:9443/tigera-elasticsearch/version", nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Set("Authorization", genToken("Jane"))
				req.Header.Set("x-cluster-id", uid)
				resp, err := http.DefaultClient.Do(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
			})*/

			It("should send a request to nonexistent endpoint/ using Jane's credentials", func() {
				req, err := http.NewRequest("GET", "https://localhost:9443/nonexistent", nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Set("Authorization", genToken("Jane"))
				req.Header.Set("x-cluster-id", uid)
				resp, err := http.DefaultClient.Do(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(404))
			})

			It("should send a request to endpoint/ without authentication", func() {
				req, err := http.NewRequest("GET", podEndpoint, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Set("x-cluster-id", uid)
				resp, err := http.DefaultClient.Do(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(401))
			})

			It("should send a request to wrong cluster id using Jane's credentials", func() {
				req, err := http.NewRequest("GET", podEndpoint, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("x-cluster-id", "wrong-cluster")
				req.Header.Add("Authorization", genToken("Jane"))
				resp, err := http.DefaultClient.Do(req)

				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(400))
			})

			It("should send a request to wrong cluster id without authentication", func() {
				req, err := http.NewRequest("GET", "https://localhost:9443/api/v1", nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("x-cluster-id", "wrong-cluster")
				resp, err := http.DefaultClient.Do(req)

				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(400))
			})

			/* TODO: https://tigera.atlassian.net/browse/SAAS-283
			It("should send a request to the health endpoint", func() {
				resp, err := http.Get("http://localhost:9080/health")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
			})*/

			It("should not allow Jane to access network policies", func() {
				req, err := http.NewRequest("GET", "https://localhost:9443/apis/networking.k8s.io/v1/networkpolicies", nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Set("x-cluster-id", uid)
				req.Header.Set("Authorization", genToken("Jane"))
				resp, err := http.DefaultClient.Do(req)

				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(403))
			})

			Context("when binding users to read network's policies", func() {
				It("Should define a role to read network policies", func() {
					policy := v1.PolicyRule{
						APIGroups: []string{"networking.k8s.io"},
						Verbs:     []string{"get", "watch", "list"},
						Resources: []string{"networkpolicies"},
					}
					role := &v1.ClusterRole{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ClusterRole",
							APIVersion: "rbac.authorization.k8s.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "read-network-policies",
						},
						Rules: []v1.PolicyRule{policy},
					}
					data, err := json.Marshal(role)
					Expect(err).NotTo(HaveOccurred())
					add(data, "https://localhost:9443/apis/rbac.authorization.k8s.io/v1/clusterroles", token)
				})

				It("Should bind Jane to read-network-policies role", func() {
					binding := &v1.ClusterRoleBinding{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ClusterRoleBinding",
							APIVersion: "rbac.authorization.k8s.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "read-network-policies-jane-binding",
						},
						RoleRef: v1.RoleRef{
							Kind: "ClusterRole",
							Name: "read-network-policies",
						},
						Subjects: []v1.Subject{{Kind: "User", Name: "Jane"}},
					}

					data, err := json.Marshal(binding)
					Expect(err).NotTo(HaveOccurred())
					add(data, "https://localhost:9443/apis/rbac.authorization.k8s.io/v1/clusterrolebindings", token)
				})

				It("should allow Jane to access network policies", func() {
					req, err := http.NewRequest("GET", "https://localhost:9443/apis/networking.k8s.io/v1/networkpolicies", nil)
					Expect(err).NotTo(HaveOccurred())

					req.Header.Set("x-cluster-id", uid)
					req.Header.Set("Authorization", genToken("Jane"))
					resp, err := http.DefaultClient.Do(req)

					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(200))
				})

				It("should delete Jane's binding to network-policies-reader role", func() {
					delete("https://localhost:9443/apis/rbac.authorization.k8s.io/v1/clusterrolebindings/read-network-policies-jane-binding", token)
				})

				It("should not allow Jane to access network policies", func() {
					req, err := http.NewRequest("GET", "https://localhost:9443/apis/networking.k8s.io/v1/networkpolicies", nil)
					Expect(err).NotTo(HaveOccurred())

					req.Header.Set("x-cluster-id", uid)
					req.Header.Set("Authorization", genToken("Jane"))
					resp, err := http.DefaultClient.Do(req)

					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(403))
				})
			})
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

func add(data []byte, endpoint string, token string) {
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(201))
}

func delete(endpoint string, token string) {
	req, err := http.NewRequest("DELETE", endpoint, nil)
	Expect(err).ToNot(HaveOccurred())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(200))
}

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

func genToken(user string) string {
	if strings.ToLower(tokenType) == "basic" {
		token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:password%s", user, user)))
		fmt.Println("Basic token is", token)
		return "Basic " + token
	}

	return "Bearer token" + user
}
