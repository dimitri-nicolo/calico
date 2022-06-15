// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/projectcalico/calico/voltron/pkg/version"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "VOLTRON"
)

var (
	versionFlag = flag.Bool("version", false, "Print version information")
)

// Config is a configuration used for Voltron
type Config struct {
	Port       int `default:"5555"`
	Host       string
	TunnelPort int    `default:"5566" split_words:"true"`
	TunnelHost string `split_words:"true"`
	TunnelCert string `default:"/certs/tunnel/cert" split_words:"true" json:"-"`
	TunnelKey  string `default:"/certs/tunnel/key" split_words:"true" json:"-"`
	LogLevel   string `default:"INFO"`

	// HTTPSCert, HTTPSKey - path to an x509 certificate and its private key used
	// for external communication (Tigera UI <-> Voltron)
	HTTPSCert string `default:"/certs/https/cert" split_words:"true" json:"-"`
	HTTPSKey  string `default:"/certs/https/key" split_words:"true" json:"-"`
	// InternalHTTPSCert, InternalHTTPSKey - path to an x509 certificate and its private key used
	//for internal communication within the K8S cluster
	InternalHTTPSCert string `default:"/certs/internal/cert" split_words:"true" json:"-"`
	InternalHTTPSKey  string `default:"/certs/internal/key" split_words:"true" json:"-"`

	K8sConfigPath                string `split_words:"true"`
	KeepAliveEnable              bool   `default:"true" split_words:"true"`
	KeepAliveInterval            int    `default:"100" split_words:"true"`
	K8sEndpoint                  string `default:"https://kubernetes.default" split_words:"true"`
	ComplianceEndpoint           string `default:"https://compliance.tigera-compliance.svc.cluster.local" split_words:"true"`
	ComplianceCABundlePath       string `default:"/certs/compliance/tls.crt" split_words:"true"`
	ComplianceInsecureTLS        bool   `default:"false" split_words:"true"`
	EnableCompliance             bool   `default:"true" split_words:"true"`
	ElasticEndpoint              string `default:"https://127.0.0.1:8443" split_words:"true"`
	NginxEndpoint                string `default:"http://127.0.0.1:8080" split_words:"true"`
	PProf                        bool   `default:"false"`
	EnableMultiClusterManagement bool   `default:"false" split_words:"true"`
	KibanaEndpoint               string `default:"https://tigera-secure-kb-http.tigera-kibana.svc:5601" split_words:"true"`
	KibanaBasePath               string `default:"/tigera-kibana" split_words:"true"`
	KibanaCABundlePath           string `default:"/certs/kibana/tls.crt" split_words:"true"`
	PacketCaptureCABundlePath    string `default:"/certs/packetcapture/tls.crt" split_words:"true"`
	PacketCaptureEndpoint        string `default:"https://tigera-packetcapture.tigera-packetcapture.svc" split_words:"true"`
	EnableImageAssurance         bool   `split_words:"true"`
	ImageAssuranceCABundlePath   string `split_words:"true"`
	ImageAssuranceEndpoint       string `split_words:"true"`
	PrometheusCABundlePath       string `default:"/certs/prometheus/tls.crt" split_words:"true"`
	PrometheusPath               string `default:"/api/v1/namespaces/tigera-prometheus/services/calico-node-prometheus:9090/proxy/" split_words:"true"`
	PrometheusEndpoint           string `default:"https://prometheus-http-api.tigera-prometheus.svc:9090" split_words:"true"`

	// Dex settings
	DexEnabled      bool   `default:"false" split_words:"true"`
	DexURL          string `default:"https://tigera-dex.tigera-dex.svc.cluster.local:5556/" split_words:"true"`
	DexBasePath     string `default:"/dex/" split_words:"true"`
	DexCABundlePath string `default:"/etc/ssl/certs/tls-dex.crt" split_words:"true"`

	// OIDC Authentication settings.
	OIDCAuthEnabled        bool   `default:"false" split_words:"true"`
	OIDCAuthJWKSURL        string `default:"https://tigera-dex.tigera-dex.svc.cluster.local:5556/dex/keys" split_words:"true"`
	OIDCAuthIssuer         string `default:"https://127.0.0.1:5556/dex" split_words:"true"`
	OIDCAuthClientID       string `default:"tigera-manager" split_words:"true"`
	OIDCAuthUsernameClaim  string `default:"email" split_words:"true"`
	OIDCAuthUsernamePrefix string `split_words:"true"`
	OIDCAuthGroupsClaim    string `default:"groups" split_words:"true"`
	OIDCAuthGroupsPrefix   string `split_words:"true"`

	// The DefaultForward parameters configure where connections from guardian should be forwarded to by default
	ForwardingEnabled               bool          `default:"true" split_words:"true"`
	DefaultForwardServer            string        `default:"tigera-secure-es-http.tigera-elasticsearch.svc:9200" split_words:"true"`
	DefaultForwardDialRetryAttempts int           `default:"5" split_words:"true"`
	DefaultForwardDialInterval      time.Duration `default:"2s" split_words:"true"`
}

func (cfg Config) String() string {
	// Parse all command-line flags
	flag.Parse()

	// For --version use case
	if *versionFlag {
		version.Version()
		os.Exit(0)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}
