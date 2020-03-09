package metrics

import (
	"context"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/security"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	licenseClient "github.com/tigera/licensing/client"
	"sync"
	"time"
)

//Declare Prometheus metrics variables
var (
	gaugeNumDays = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "license_number_of_days",
		Help: "Total number of days license in valid state.",
	})
	gaugeNumNodes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "calico_nodes_used",
		Help: "Total number of nodes currently in use.",
	})
	gaugeMaxNodes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "calico_maximum_licensed_node",
		Help: "Total number of Licensed nodes.",
	})
	gaugeValidLicense = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "calico_license_valid",
		Help: "Is valid calico Enterprise License.",
	})

	wg sync.WaitGroup
)

type LicenseReporter struct {
	port        int
	pollMinutes int
	host        string
	caFile      string
	keyFile     string
	certFile    string
	registry    *prometheus.Registry
	client      clientv3.Interface
}

func NewLicenseReporter(host, certFile, keyFile, caFile string, port, pollMinutes int) *LicenseReporter {

	return &LicenseReporter{
		port:        port,
		host:        host,
		caFile:      caFile,
		keyFile:     keyFile,
		certFile:    certFile,
		registry:    prometheus.NewRegistry(),
		pollMinutes: pollMinutes,
	}
}

// Start Prometheus server and data collecteion
func (lr *LicenseReporter) Start() {
	var err error
	lr.client, err = clientv3.NewFromEnv()
	if err != nil {
		log.Fatal("Unable to get client v3 handle")
		return
	}
	wg.Add(2)
	go lr.servePrometheusMetrics()
	go lr.startReporter()
	wg.Wait()
}

// Register Prometheus Metrics variable
func init() {
	prometheus.MustRegister(gaugeNumDays)
	prometheus.MustRegister(gaugeNumNodes)
	prometheus.MustRegister(gaugeMaxNodes)
	prometheus.MustRegister(gaugeValidLicense)
}

// servePrometheusMetrics starts a lightweight web server to server prometheus metrics.
func (lr *LicenseReporter) servePrometheusMetrics() {
	err := security.ServePrometheusMetrics(prometheus.DefaultGatherer, lr.host, lr.port, lr.certFile, lr.keyFile, lr.caFile)
	if err != nil {
		log.Errorf("Error from libcalico go: ", err)
	}
	wg.Done()
}

//Continously scrape License Validity, Number of days license valid
//Maximum number of nodes licensed and Number of nodes in Use
func (lr *LicenseReporter) startReporter() {
	for {
		//Get Licensekey from datastore, only if license exists scrape data
		lic, err := lr.client.LicenseKey().Get(context.Background(), "default", options.GetOptions{})
		if err != nil {
			switch err.(type) {
			case cerrors.ErrorResourceDoesNotExist:
				log.Infof("No valid License found in your Cluster")
			default:
				log.Infof("Error getting LicenseKey :%v", err)
			}
			time.Sleep(time.Duration(lr.pollMinutes) * time.Minute)
			continue
		}

		nodeList, _ := lr.client.Nodes().List(context.Background(), options.ListOptions{})
		isValid, daysToExpire, maxNodes := lr.licesneHandler(*lic)
		gaugeNumNodes.Set(float64(len(nodeList.Items)))
		gaugeNumDays.Set(float64(daysToExpire))
		gaugeMaxNodes.Set(float64(maxNodes))
		if isValid == true {
			gaugeValidLicense.Set(float64(1))
		} else {
			gaugeValidLicense.Set(float64(0))
		}
		time.Sleep(time.Duration(lr.pollMinutes) * time.Minute)
	}
	wg.Done()
}

//Decode License, get expiry date, maximum allowed nodes
func (lr *LicenseReporter) licesneHandler(lic api.LicenseKey) (isValid bool, daysToExpire, maxNodes int) {

	//Decode the LicenseKey
	claims, err := licenseClient.Decode(lic)
	if err != nil {
		log.Warnf("License is corrupted. Please Contact Tigera support")
		return false, 0, 0
	}

	//Check if License is Valid
	if licStatus := claims.Validate(); licStatus != licenseClient.Valid {
		log.Warnf("License has expired. Please Contact Tigera support")
		return false, 0, 0
	}

	//Find number of days license valid, Maximum nodes
	durationInHours := int(claims.Claims.Expiry.Time().Sub(time.Now()).Hours())
	maxNodes = *claims.Nodes
	return true, durationInHours / 24, maxNodes
}
