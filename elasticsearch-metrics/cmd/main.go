// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus-community/elasticsearch_exporter/collector"
	"github.com/prometheus-community/elasticsearch_exporter/pkg/clusterinfo"
	"github.com/prometheus/client_golang/prometheus"
	promcollectorsversion "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

// This file is a modified version of main.go from the original repo with added and modified TLS settings.
// See README.md for more information.

const name = "elasticsearch_exporter"

type transportWithAPIKey struct {
	underlyingTransport http.RoundTripper
	apiKey              string
}

func (t *transportWithAPIKey) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("ApiKey %s", t.apiKey))
	return t.underlyingTransport.RoundTrip(req)
}

func main() {
	var (
		listenAddress = kingpin.Flag("web.listen-address",
			"Address to listen on for web interface and telemetry.").
			Default(":9114").String()
		metricsPath = kingpin.Flag("web.telemetry-path",
			"Path under which to expose metrics.").
			Default("/metrics").String()
		esURI = kingpin.Flag("es.uri",
			"HTTP API address of an Elasticsearch node.").
			Default("http://localhost:9200").String()
		esTimeout = kingpin.Flag("es.timeout",
			"Timeout for trying to get stats from Elasticsearch.").
			Default("5s").Duration()
		esAllNodes = kingpin.Flag("es.all",
			"Export stats for all nodes in the cluster. If used, this flag will override the flag es.node.").
			Default("false").Bool()
		esNode = kingpin.Flag("es.node",
			"Node's name of which metrics should be exposed.").
			Default("_local").String()
		esExportIndices = kingpin.Flag("es.indices",
			"Export stats for indices in the cluster.").
			Default("false").Bool()
		esExportIndicesSettings = kingpin.Flag("es.indices_settings",
			"Export stats for settings of all indices of the cluster.").
			Default("false").Bool()
		esExportIndicesMappings = kingpin.Flag("es.indices_mappings",
			"Export stats for mappings of all indices of the cluster.").
			Default("false").Bool()
		esExportClusterSettings = kingpin.Flag("es.cluster_settings",
			"Export stats for cluster settings.").
			Default("false").Bool()
		esExportAliases = kingpin.Flag("es.aliases",
			"Export informational aliases metrics.").
			Default("true").Bool()
		esExportShards = kingpin.Flag("es.shards",
			"Export stats for shards in the cluster (implies --es.indices).").
			Default("false").Bool()
		esExportSnapshots = kingpin.Flag("es.snapshots",
			"Export stats for the cluster snapshots.").
			Default("false").Bool()
		esClusterInfoInterval = kingpin.Flag("es.clusterinfo.interval",
			"Cluster info update interval for the cluster label").
			Default("5m").Duration()
		esCA = kingpin.Flag("es.ca",
			"Path to PEM file that contains trusted Certificate Authorities for the Elasticsearch connection.").
			Default("").String()
		esClientPrivateKey = kingpin.Flag("es.client-private-key",
			"Path to PEM file that contains the private key for client auth when connecting to Elasticsearch.").
			Default("").String()
		esClientCert = kingpin.Flag("es.client-cert",
			"Path to PEM file that contains the corresponding cert for the private key to connect to Elasticsearch.").
			Default("").String()
		esInsecureSkipVerify = kingpin.Flag("es.ssl-skip-verify",
			"Skip SSL verification when connecting to Elasticsearch.").
			Default("false").Bool()
		logLevel = kingpin.Flag("log.level",
			"Sets the loglevel. Valid levels are debug, info, warn, error").
			Default("info").String()
		logFormat = kingpin.Flag("log.format",
			"Sets the log format. Valid formats are json and logfmt").
			Default("logfmt").String()
		logOutput = kingpin.Flag("log.output",
			"Sets the log output. Valid outputs are stdout and stderr").
			Default("stdout").String()

		// BEGIN TIGERA CHANGES
		ca = kingpin.Flag("ca.crt",
			"Path to PEM file that contains trusted Certificate Authorities.").
			Default("/tls/ca.crt").String()
		privateKey = kingpin.Flag("tls.key",
			"Path to PEM file that contains the private key for server auth.").
			Default("/tls/tls.key").String()
		certificate = kingpin.Flag("tls.crt",
			"Path to PEM file that contains the corresponding cert for the private key.").
			Default("/tls/tls.crt").String()
		// END TIGERA CHANGES
	)

	kingpin.Version(version.Print(name))
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()

	logger := getLogger(*logLevel, *logOutput, *logFormat)

	esURL, err := url.Parse(*esURI)
	if err != nil {
		_ = level.Error(logger).Log(
			"msg", "failed to parse es.uri",
			"err", err,
		)
		os.Exit(1)
	}

	esUsername := os.Getenv("ES_USERNAME")
	esPassword := os.Getenv("ES_PASSWORD")

	if esUsername != "" && esPassword != "" {
		esURL.User = url.UserPassword(esUsername, esPassword)
	}

	// returns nil if not provided and falls back to simple TCP.
	// BEGIN TIGERA CHANGES
	fipsModeEnabled := os.Getenv("FIPS_MODE_ENABLED") == "true"
	tlsConfig := createTLSConfig(*esCA, *esClientCert, *esClientPrivateKey, *esInsecureSkipVerify, fipsModeEnabled)
	// END TIGERA CHANGES

	var httpTransport http.RoundTripper

	httpTransport = &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	esAPIKey := os.Getenv("ES_API_KEY")

	if esAPIKey != "" {
		httpTransport = &transportWithAPIKey{
			underlyingTransport: httpTransport,
			apiKey:              esAPIKey,
		}
	}

	httpClient := &http.Client{
		Timeout:   *esTimeout,
		Transport: httpTransport,
	}

	// version metric
	prometheus.MustRegister(promcollectorsversion.NewCollector(name))

	// cluster info retriever
	clusterInfoRetriever := clusterinfo.New(logger, httpClient, esURL, *esClusterInfoInterval)

	prometheus.MustRegister(collector.NewClusterHealth(logger, httpClient, esURL))
	prometheus.MustRegister(collector.NewNodes(logger, httpClient, esURL, *esAllNodes, *esNode))

	if *esExportIndices || *esExportShards {
		iC := collector.NewIndices(logger, httpClient, esURL, *esExportShards, *esExportAliases)
		prometheus.MustRegister(iC)
		if registerErr := clusterInfoRetriever.RegisterConsumer(iC); registerErr != nil {
			_ = level.Error(logger).Log("msg", "failed to register indices collector in cluster info")
			os.Exit(1)
		}
	}

	if *esExportSnapshots {
		prometheus.MustRegister(collector.NewSnapshots(logger, httpClient, esURL))
	}

	if *esExportClusterSettings {
		prometheus.MustRegister(collector.NewClusterSettings(logger, httpClient, esURL))
	}

	if *esExportIndicesSettings {
		prometheus.MustRegister(collector.NewIndicesSettings(logger, httpClient, esURL))
	}

	if *esExportIndicesMappings {
		prometheus.MustRegister(collector.NewIndicesMappings(logger, httpClient, esURL))
	}
	// BEGIN TIGERA CHANGES
	caCertFile, err := os.ReadFile(*ca)
	if err != nil {
		log.Fatalf("error reading CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertFile)

	serverTLSConfig := NewTLSConfig(fipsModeEnabled)
	serverTLSConfig.ClientCAs = caCertPool
	serverTLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	serverTLSConfig.MinVersion = tls.VersionTLS12
	server := &http.Server{
		TLSConfig: serverTLSConfig,
	}
	// END TIGERA CHANGES

	// Create a context that is cancelled on SIGKILL or SIGINT.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// start the cluster info retriever
	switch runErr := clusterInfoRetriever.Run(ctx); runErr {
	case nil:
		_ = level.Info(logger).Log(
			"msg", "started cluster info retriever",
			"interval", (*esClusterInfoInterval).String(),
		)
	case clusterinfo.ErrInitialCallTimeout:
		_ = level.Info(logger).Log("msg", "initial cluster info call timed out")
	default:
		_ = level.Error(logger).Log("msg", "failed to run cluster info retriever", "err", err)
		os.Exit(1)
	}

	// register cluster info retriever as prometheus collector
	prometheus.MustRegister(clusterInfoRetriever)

	mux := http.DefaultServeMux
	mux.Handle(*metricsPath, promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte(`<html>
			<head><title>Elasticsearch Exporter</title></head>
			<body>
			<h1>Elasticsearch Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			_ = level.Error(logger).Log(
				"msg", "failed handling writer",
				"err", err,
			)
		}
	})

	// health endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
	})

	server.Handler = mux
	server.Addr = *listenAddress

	_ = level.Info(logger).Log(
		"msg", "starting elasticsearch_exporter",
		"addr", *listenAddress,
	)

	go func() {
		// BEGIN TIGERA CHANGES
		if err := server.ListenAndServeTLS(*certificate, *privateKey); err != nil {
			// END TIGERA CHANGES
			_ = level.Error(logger).Log(
				"msg", "http server quit",
				"err", err,
			)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	_ = level.Info(logger).Log("msg", "shutting down")
	// create a context for graceful http server shutdown
	srvCtx, srvCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer srvCancel()
	_ = server.Shutdown(srvCtx)
}

// BEGIN TIGERA CHANGES

// NewTLSConfig returns a tls.Config with the recommended default settings for Calico Enterprise components.
// Read more recommendations here in Chapter 3:
// https://www.gsa.gov/cdnstatic/SSL_TLS_Implementation_%5BCIO_IT_Security_14-69_Rev_6%5D_04-06-2021docx.pdf
func NewTLSConfig(fipsMode bool) *tls.Config {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	if fipsMode {
		cfg.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		}
		cfg.CurvePreferences = []tls.CurveID{tls.CurveP384, tls.CurveP256}
		cfg.MinVersion = tls.VersionTLS12
		// Our certificate for FIPS validation not mention validation for v1.3.
		cfg.MaxVersion = tls.VersionTLS12
		cfg.Renegotiation = tls.RenegotiateNever
	}
	return cfg
}

// END TIGERA CHANGES

func createTLSConfig(pemFile, pemCertFile, pemPrivateKeyFile string, insecureSkipVerify, fipsModeEnabled bool) *tls.Config {
	// BEGIN TIGERA CHANGES
	tlsConfig := NewTLSConfig(fipsModeEnabled)
	// END TIGERA CHANGES
	if insecureSkipVerify {
		// pem settings are irrelevant if we're skipping verification anyway
		tlsConfig.InsecureSkipVerify = true
	}
	if len(pemFile) > 0 {
		rootCerts, err := loadCertificatesFrom(pemFile)
		if err != nil {
			log.Fatalf("Couldn't load root certificate from %s. Got %s.", pemFile, err)
			return nil
		}
		tlsConfig.RootCAs = rootCerts
	}
	if len(pemCertFile) > 0 && len(pemPrivateKeyFile) > 0 {
		// Load files once to catch configuration error early.
		_, err := loadPrivateKeyFrom(pemCertFile, pemPrivateKeyFile)
		if err != nil {
			log.Fatalf("Couldn't setup client authentication. Got %s.", err)
			return nil
		}
		// Define a function to load certificate and key lazily at TLS handshake to
		// ensure that the latest files are used in case they have been rotated.
		tlsConfig.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return loadPrivateKeyFrom(pemCertFile, pemPrivateKeyFile)
		}
	}
	return tlsConfig
}

func loadCertificatesFrom(pemFile string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, err
	}
	certificates := x509.NewCertPool()
	certificates.AppendCertsFromPEM(caCert)
	return certificates, nil
}

func loadPrivateKeyFrom(pemCertFile, pemPrivateKeyFile string) (*tls.Certificate, error) {
	privateKey, err := tls.LoadX509KeyPair(pemCertFile, pemPrivateKeyFile)
	if err != nil {
		return nil, err
	}
	return &privateKey, nil
}
