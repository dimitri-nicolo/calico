// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.

package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/voltron/internal/pkg/bootstrap"
	"github.com/projectcalico/calico/voltron/internal/pkg/config"
	"github.com/projectcalico/calico/voltron/internal/pkg/proxy"
	"github.com/projectcalico/calico/voltron/internal/pkg/regex"
	"github.com/projectcalico/calico/voltron/internal/pkg/server"
	"github.com/projectcalico/calico/voltron/internal/pkg/utils"
)

func main() {
	cfg := config.Config{}
	if err := envconfig.Process(config.EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting %s with %s", config.EnvConfigPrefix, cfg)

	if cfg.PProf {
		go func() {
			err := bootstrap.StartPprof()
			log.WithError(err).Fatal("PProf exited.")
		}()
	}

	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)

	kubernetesAPITargets, err := regex.CompileRegexStrings([]string{
		`^/api/?`,
		`^/apis/?`,
	})

	unauthenticatedTargets := []string{
		// Need this unauthenticated so a browswer can download the manager UI webcode.
		"/",
		// We need the es/version unauthenticated so the liveness probe for es-proxy
		// can be successful.
		"/tigera-elasticsearch/version",
		// We use special token authentication with Kibana so it doesn't need to be
		// authenticated.
		cfg.KibanaBasePath,
	}
	if cfg.DexEnabled {
		// Dex endpoints setup auth tokens so we can't authenticate access.
		unauthenticatedTargets = append(unauthenticatedTargets, cfg.DexBasePath)
	}

	if err != nil {
		log.WithError(err).Fatalf("Failed to parse tunnel target whitelist.")
	}

	opts := []server.Option{
		server.WithDefaultAddr(addr),
		server.WithKeepAliveSettings(cfg.KeepAliveEnable, cfg.KeepAliveInterval),
		server.WithExternalCredsFiles(cfg.HTTPSCert, cfg.HTTPSKey),
		server.WithKubernetesAPITargets(kubernetesAPITargets),
	}

	config := bootstrap.NewRestConfig(cfg.K8sConfigPath)
	k8s := bootstrap.NewK8sClientWithConfig(config)

	if cfg.EnableMultiClusterManagement {
		// the cert used to sign guardian certs is required no matter what to verify inbound connections
		tunnelSigningX509Cert, err := utils.LoadX509Cert(cfg.TunnelCert)
		if err != nil {
			log.WithError(err).Fatal("couldn't load tunnel X509 key pair")
		}

		if cfg.UseHTTPSCertOnTunnel {
			// if a tunnelCert and tunnelKey was specified, use those for the voltron server cert.
			// this uses a different certificate chain for guardian and voltron, but allows use of
			// a separate, public CA to verify voltron certificates instead of relying on self-signed certs.
			tlsCert, err := tls.LoadX509KeyPair(cfg.HTTPSCert, cfg.HTTPSKey)
			if err != nil {
				log.WithError(err).Fatal("couldn't load tunnel X509 key pair")
			}

			opts = append(opts, server.WithTunnelCert(tlsCert))
		} else if cfg.TunnelKey != "" {
			// otherwise, use the signing chain
			tlsCert, err := tls.LoadX509KeyPair(cfg.TunnelCert, cfg.TunnelKey)
			if err != nil {
				log.WithError(err).Fatal("couldn't load tunnel X509 key pair")
			}
			opts = append(opts, server.WithTunnelCert(tlsCert))
		} else {
			log.Fatal("must specify either a tunnel cert & key or a signing key")
		}

		// With the introduction of Centralized ElasticSearch for Multi-cluster Management,
		// certain categories of requests related to a specific cluster will be proxied
		// within the Management cluster (instead of being sent down a secure tunnel to the
		// actual Managed cluster).
		// In the setup below, we create a list of URI paths that should still go through the
		// tunnel down to a Managed cluster. Requests that do not match this whitelist, will
		// instead be proxied locally (within the Management cluster itself using the
		// defaultProxy that is set up later on in this function). The whitelist is used
		// within the server's clusterMuxer handler.
		tunnelTargetWhitelist, err := regex.CompileRegexStrings([]string{
			`^/api/?`,
			`^/apis/?`,
			`^/packet-capture/?`,
		})

		if err != nil {
			log.WithError(err).Fatalf("Failed to parse tunnel target whitelist.")
		}

		kibanaURL, err := url.Parse(cfg.KibanaEndpoint)
		if err != nil {
			log.WithError(err).Fatalf("failed to parse Kibana endpoint %s", cfg.KibanaEndpoint)
		}

		sniServiceMap := map[string]string{
			kibanaURL.Hostname(): kibanaURL.Host, // Host includes the port, Hostname does not
		}

		if cfg.EnableImageAssurance && cfg.ImageAssuranceEndpoint != "" && cfg.ImageAssuranceCABundlePath != "" {
			bastURL, err := url.Parse(cfg.ImageAssuranceEndpoint)
			if err != nil {
				log.WithError(err).Fatalf("failed to parse Bast API endpoint %s", cfg.KibanaEndpoint)
			}

			sniServiceMap[bastURL.Hostname()] = bastURL.Host
		}

		log.WithField("map", sniServiceMap).Info("SNI map")

		opts = append(opts,
			server.WithInternalCredFiles(cfg.InternalHTTPSCert, cfg.InternalHTTPSKey),
			server.WithTunnelSigningCreds(tunnelSigningX509Cert),
			server.WithForwardingEnabled(cfg.ForwardingEnabled),
			server.WithDefaultForwardServer(cfg.DefaultForwardServer, cfg.DefaultForwardDialRetryAttempts, cfg.DefaultForwardDialInterval),
			server.WithTunnelTargetWhitelist(tunnelTargetWhitelist),
			server.WithSNIServiceMap(sniServiceMap),
			server.WithFIPSModeEnabled(cfg.FIPSModeEnabled),
			server.WithCheckManagedClusterAuthorizationBeforeProxy(cfg.CheckManagedClusterAuthorizationBeforeProxy),
			server.WithUnauthenticatedTargets(unauthenticatedTargets),
		)
	}

	targetList := []bootstrap.Target{
		{
			Path:         "/api/",
			Dest:         cfg.K8sEndpoint,
			CABundlePath: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
		{
			Path:         "/apis/",
			Dest:         cfg.K8sEndpoint,
			CABundlePath: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
		{
			Path:             "/tigera-elasticsearch/",
			Dest:             cfg.ElasticEndpoint,
			PathRegexp:       []byte("^/tigera-elasticsearch/?"),
			PathReplace:      []byte("/"),
			AllowInsecureTLS: true,
		},
		{
			// Define this a separate path because the liveness probe for es-proxy goes through
			// voltron and does not need to be authenticated but the rest of tigera-elasticsearch
			// does. With this defined this Path will be matched specifically and we can serve
			// it unauthenticated.
			Path:             "/tigera-elasticsearch/version",
			Dest:             cfg.ElasticEndpoint,
			PathRegexp:       []byte("^/tigera-elasticsearch/version?"),
			PathReplace:      []byte("/version"),
			AllowInsecureTLS: true,
		},
		{
			Path:         "/packet-capture/",
			Dest:         cfg.PacketCaptureEndpoint,
			PathRegexp:   []byte("^/packet-capture/?"),
			PathReplace:  []byte("/"),
			CABundlePath: cfg.PacketCaptureCABundlePath,
		},
		{
			Path:         cfg.PrometheusPath,
			Dest:         cfg.PrometheusEndpoint,
			PathRegexp:   []byte(fmt.Sprintf("^%v/?", cfg.PrometheusPath)),
			PathReplace:  []byte("/"),
			CABundlePath: cfg.PrometheusCABundlePath,
		},
		{
			Path:         cfg.QueryserverPath,
			Dest:         cfg.QueryserverEndpoint,
			PathRegexp:   []byte(fmt.Sprintf("^%v/?", cfg.QueryserverPath)),
			PathReplace:  []byte("/"),
			CABundlePath: cfg.QueryserverCABundlePath,
		},
		{
			Path:         cfg.KibanaBasePath,
			Dest:         cfg.KibanaEndpoint,
			CABundlePath: cfg.KibanaCABundlePath,
		},
		{
			Path:             "/",
			Dest:             cfg.NginxEndpoint,
			AllowInsecureTLS: true,
		},
	}

	if cfg.EnableCompliance {
		targetList = append(targetList, bootstrap.Target{
			Path:             "/compliance/",
			Dest:             cfg.ComplianceEndpoint,
			CABundlePath:     cfg.ComplianceCABundlePath,
			AllowInsecureTLS: cfg.ComplianceInsecureTLS,
		})
	}

	if cfg.EnableImageAssurance && cfg.ImageAssuranceEndpoint != "" && cfg.ImageAssuranceCABundlePath != "" {
		targetList = append(targetList, bootstrap.Target{
			Path:         "/bast/",
			Dest:         cfg.ImageAssuranceEndpoint,
			PathRegexp:   []byte("^/bast/?"),
			PathReplace:  []byte("/"),
			CABundlePath: cfg.ImageAssuranceCABundlePath,
		})
	}

	if cfg.EnableCalicoCloudRbacApi && cfg.CalicoCloudRbacApiEndpoint != "" && cfg.CalicoCloudRbacApiCABundlePath != "" {
		targetList = append(targetList, bootstrap.Target{
			Path:         "/cloud-rbac/",
			Dest:         cfg.CalicoCloudRbacApiEndpoint,
			PathRegexp:   []byte("^/cloud-rbac/?"),
			PathReplace:  []byte("/"),
			CABundlePath: cfg.CalicoCloudRbacApiCABundlePath,
		})
	}

	var options []auth.JWTAuthOption
	if cfg.OIDCAuthEnabled {
		// If dex is enabled we need to add the CA Bundle, otherwise the default trusted certs from the image will
		// suffice.
		if cfg.DexEnabled {
			targetList = append(targetList, bootstrap.Target{
				Path:         cfg.DexBasePath,
				Dest:         cfg.DexURL,
				CABundlePath: cfg.DexCABundlePath,
			})
		}

		authOpts := []auth.DexOption{
			auth.WithGroupsClaim(cfg.OIDCAuthGroupsClaim),
			auth.WithJWKSURL(cfg.OIDCAuthJWKSURL),
			auth.WithUsernamePrefix(cfg.OIDCAuthUsernamePrefix),
			auth.WithGroupsPrefix(cfg.OIDCAuthGroupsPrefix),
		}
		if cfg.CalicoCloudRequireTenantClaim {
			if cfg.CalicoCloudTenantClaim == "" {
				log.Panic("Tenant claim not specified")
			}
			authOpts = append(authOpts, auth.WithCalicoCloudTenantClaim(cfg.CalicoCloudTenantClaim))
		}

		oidcAuth, err := auth.NewDexAuthenticator(
			cfg.OIDCAuthIssuer,
			cfg.OIDCAuthClientID,
			cfg.OIDCAuthUsernameClaim,
			authOpts...)
		if err != nil {
			log.WithError(err).Panic("Unable to create dex authenticator")
		}

		options = append(options, auth.WithAuthenticator(cfg.OIDCAuthIssuer, oidcAuth))
	}
	authn, err := auth.NewJWTAuth(config, k8s, options...)
	if err != nil {
		log.Fatal("Unable to create authenticator", err)
	}

	targets, err := bootstrap.ProxyTargets(targetList, cfg.FIPSModeEnabled)

	if err != nil {
		log.WithError(err).Fatal("Failed to parse default proxy targets.")
	}

	defaultProxy, err := proxy.New(targets)
	if err != nil {
		log.WithError(err).Fatalf("Failed to create a default k8s proxy.")
	}
	opts = append(opts, server.WithDefaultProxy(defaultProxy))

	srv, err := server.New(
		k8s,
		config,
		authn,
		opts...,
	)

	if err != nil {
		log.WithError(err).Fatal("Failed to create server.")
	}

	if cfg.EnableMultiClusterManagement {
		lisTun, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.TunnelHost, cfg.TunnelPort))
		if err != nil {
			log.WithError(err).Fatal("Failed to create tunnel listener.")
		}

		go func() {
			err := srv.ServeTunnelsTLS(lisTun)
			log.WithError(err).Fatal("Tunnel server exited.")
		}()

		go func() {
			err := srv.WatchK8s()
			log.WithError(err).Fatal("K8s watcher exited.")
		}()

		log.Infof("Voltron listens for tunnels at %s", lisTun.Addr().String())
	}

	log.Infof("Voltron listens for HTTP request at %s", addr)
	if err := srv.ListenAndServeHTTPS(); err != nil {
		log.Fatal(err)
	}
}
