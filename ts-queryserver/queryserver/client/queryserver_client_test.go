package client

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	querycacheclient "github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

var _ = Describe("QuerysServerClient tests", func() {
	Context("test SearchEndpoints", func() {
		var server *httptest.Server
		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "" {
					w.WriteHeader(http.StatusForbidden)
				}
				if r.Header.Get("Accept") != "application/json" {
					w.WriteHeader(http.StatusBadRequest)
				}
				if r.Method != "POST" {
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"count": 0, "items": []}`))
				Expect(err).ShouldNot(HaveOccurred())
			}))
		})

		AfterEach(func() {
			server.Close()
		})
		It("managed cluster", func() {
			config := &QueryServerConfig{
				QueryServerTunnelURL: server.URL,
				QueryServerURL:       "",
				QueryServerCA:        "/etc/pki/tls/certs/tigera-ca-bundle.crt",
				QueryServerToken:     "test_data/token",
			}

			client := queryServerClient{
				client: &http.Client{},
			}

			body := &querycacheclient.QueryEndpointsReqBody{}
			resp, err := client.SearchEndpoints(config, body, "managed-cluster")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.Count).To(Equal(0))

		})

		It("management / standalone cluster", func() {
			config := &QueryServerConfig{
				QueryServerTunnelURL: "",
				QueryServerURL:       server.URL,
				QueryServerCA:        "/etc/pki/tls/certs/tigera-ca-bundle.crt",
				QueryServerToken:     "test_data/token",
			}

			client := queryServerClient{
				client: &http.Client{},
			}

			body := &querycacheclient.QueryEndpointsReqBody{}
			resp, err := client.SearchEndpoints(config, body, "cluster")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.Count).To(Equal(0))
		})

		It("multi-tenant management cluster making a request for a managed cluster", func() {
			// Create a temporary directory to store file dependencies
			tempDir, err := os.MkdirTemp("", "query-client")
			if err != nil {
				fmt.Println("Error creating temporary file:", err)
				return
			}
			defer func(path string) {
				_ = os.RemoveAll(path)
			}(tempDir)

			// Write CA
			err = writeCA(tempDir)
			Expect(err).ShouldNot(HaveOccurred())

			// Write token
			err = writeToken(tempDir)
			Expect(err).ShouldNot(HaveOccurred())

			// Create a mock server that enforces the checks for impersonation headers
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()

				Expect(r.Header.Get("Impersonate-User")).To(Equal("system:serviceaccount:tigera-manager:tigera-manager"))
				Expect(r.Header.Values("Impersonate-Group")).To(ContainElements(
					"system:authenticated",
					"system:serviceaccounts",
					"system:serviceaccounts:tigera-manager",
				))

				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"count": 0, "items": []}`))
				Expect(err).ShouldNot(HaveOccurred())
			}))

			defer server.Close()

			config := &QueryServerConfig{
				QueryServerTunnelURL:    server.URL,
				QueryServerURL:          "",
				QueryServerCA:           filepath.Join(tempDir, "tigera-ca-bundle.crt"),
				QueryServerToken:        filepath.Join(tempDir, "token"),
				AddImpersonationHeaders: true,
			}

			client, err := NewQueryServerClient(config)
			Expect(err).ShouldNot(HaveOccurred())

			body := &querycacheclient.QueryEndpointsReqBody{}
			resp, err := client.SearchEndpoints(config, body, "managedcluster")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.Count).To(Equal(0))
		})

		It("query server token is empty", func() {
			config := &QueryServerConfig{
				QueryServerTunnelURL: "",
				QueryServerURL:       server.URL,
				QueryServerCA:        "/etc/pki/tls/certs/tigera-ca-bundle.crt",
				QueryServerToken:     "",
			}

			client := queryServerClient{
				client: &http.Client{},
			}

			body := &querycacheclient.QueryEndpointsReqBody{}
			resp, err := client.SearchEndpoints(config, body, "cluster")
			Expect(err).To(Equal(errInvalidToken))
			Expect(resp).To(BeNil())
		})

		It("query server token is empty", func() {
			config := &QueryServerConfig{
				QueryServerTunnelURL: "",
				QueryServerURL:       server.URL,
				QueryServerCA:        "/etc/pki/tls/certs/tigera-ca-bundle.crt",
				QueryServerToken:     "",
			}

			client := queryServerClient{
				client: &http.Client{},
			}

			body := &querycacheclient.QueryEndpointsReqBody{}
			resp, err := client.SearchEndpoints(config, body, "cluster")
			Expect(err).Should(HaveOccurred())
			Expect(resp).To(BeNil())
		})
	})
})

func writeToken(tempDir string) error {
	tokenValue := []byte("any")
	tokenFilePath := filepath.Join(tempDir, "token")
	tokenFile, err := os.Create(tokenFilePath)
	if err != nil {
		return err
	}
	write, err := tokenFile.Write(tokenValue)
	if err != nil {
		return err
	}
	if write != len(tokenValue) {
		return fmt.Errorf("failed to write all token")
	}
	err = tokenFile.Close()
	if err != nil {
		return err
	}

	return nil
}

func writeCA(tempDir string) error {
	ca, caKey, err := createCAKeyPair()
	if err != nil {
		return err
	}
	caBytes, err := signAndEncodeCert(ca, caKey, ca, caKey)
	if err != nil {
		return err
	}
	caFilePath := filepath.Join(tempDir, "tigera-ca-bundle.crt")
	caFile, err := os.Create(caFilePath)
	if err != nil {
		return err
	}
	write, err := caFile.Write(caBytes)
	if err != nil {
		return err
	}
	if write != len(caBytes) {
		return fmt.Errorf("expected to write all bytes to the CA file")
	}
	err = caFile.Close()
	if err != nil {
		return err
	}
	return nil
}

func createCAKeyPair() (*x509.Certificate, *rsa.PrivateKey, error) {
	// Create a x509 template for the mustCreateCAKeyPair
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"Tigera"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Generate a private key
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	return template, key, nil
}

func signAndEncodeCert(ca *x509.Certificate, caPrivateKey *rsa.PrivateKey, cert *x509.Certificate, key *rsa.PrivateKey) ([]byte, error) {
	// Sign the certificate with the provided CA
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &key.PublicKey, caPrivateKey)
	if err != nil {
		return nil, err
	}

	// Encode the certificate
	certPEM := bytes.Buffer{}
	err = pem.Encode(&certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		return nil, err
	}

	return certPEM.Bytes(), nil
}
