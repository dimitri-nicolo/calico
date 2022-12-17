package main

import (
	"crypto/x509"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/intrusion-detection-controller/install/pkg/config"
)

var (
	//go:embed data/api-kibana-dashboard.json
	apiDashboard string
	//go:embed data/tor-vpn-dashboard.json
	torVpnDashBoard string
	//go:embed data/honeypod-dashboard.json
	honeypodDashboard string
	//go:embed data/dns-dashboard.json
	dnsDashboard string
	//go:embed data/kubernetes-api-dashboard.json
	k8sApiDashboard string
	//go:embed data/siem-index.json
	siemIndex string
	//go:embed data/l7-dashboard.json
	l7Dashboard string
)

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	// Attempt to load CA cert.
	caCert, err := os.ReadFile(cfg.KibanaCAPath)
	if err != nil {
		log.Panicf("unable to read certificate for Kibana: %v", err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCert)
	if !ok {
		log.Panicf("failed to add certificate to the pool: %v", err)
	}

	// Set up default HTTP transport config.
	tlsConfig := tls.NewTLSConfig(cfg.FIPSMode)
	tlsConfig.RootCAs = caCertPool

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	var kibanaURL string
	if cfg.KibanaSpaceID == "" {
		kibanaURL = fmt.Sprintf("%s://%s:%s/tigera-kibana/", cfg.KibanaScheme, cfg.KibanaHost, cfg.KibanaPort)
	} else {
		kibanaURL = fmt.Sprintf("%s://%s:%s/tigera-kibana/s/%s/", cfg.KibanaScheme, cfg.KibanaHost, cfg.KibanaPort, cfg.KibanaSpaceID)
	}
	bulkCreateURL := fmt.Sprintf("%sapi/saved_objects/_bulk_create", kibanaURL)
	postDashboard(client, bulkCreateURL, cfg.ElasticUsername, cfg.ElasticPassword, "apiDashboard", apiDashboard)
	postDashboard(client, bulkCreateURL, cfg.ElasticUsername, cfg.ElasticPassword, "torVpnDashBoard", torVpnDashBoard)
	postDashboard(client, bulkCreateURL, cfg.ElasticUsername, cfg.ElasticPassword, "honeypodDashboard", honeypodDashboard)
	postDashboard(client, bulkCreateURL, cfg.ElasticUsername, cfg.ElasticPassword, "dnsDashboard", dnsDashboard)
	postDashboard(client, bulkCreateURL, cfg.ElasticUsername, cfg.ElasticPassword, "k8sApiDashboard", k8sApiDashboard)
	postDashboard(client, bulkCreateURL, cfg.ElasticUsername, cfg.ElasticPassword, "l7Dashboard", l7Dashboard)

	configURL := fmt.Sprintf("%sapi/saved_objects/config/7.6.2", kibanaURL)
	postDashboard(client, configURL, cfg.ElasticUsername, cfg.ElasticPassword, "siemIndex", siemIndex)
}

func postDashboard(client *http.Client, url, username, password, dashboardName, dashboard string) {
	log.Infof("POST %v %v", url, dashboardName)
	req, err := http.NewRequest("POST", url, strings.NewReader(dashboard))
	if err != nil {
		log.Panicf("unable to setup dashboard %s, err=%v", dashboardName, err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-xsrf", "reporting")

	resp, err := client.Do(req)
	if err != nil {
		log.Panicf("unable to setup dashboard %s, err=%v", dashboardName, err)
	}
	if resp.StatusCode == http.StatusConflict {
		req, err = http.NewRequest("PUT", url, strings.NewReader(dashboard))
		if err != nil {
			log.Panicf("unable to setup dashboard %s, err=%v", dashboardName, err)
		}
		req.SetBasicAuth(username, password)
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("kbn-xsrf", "reporting")
		log.Infof("Resource exists, PUT %v %v", url, dashboardName)
		resp, err = client.Do(req)
		if err != nil {
			log.Panicf("unable to setup dashboard %s, err=%v", dashboardName, err)
		}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Panicf("unable to setup dashboard %s, status=%v", dashboardName, resp.Status)
	}
	log.Info(resp.Status)
}
