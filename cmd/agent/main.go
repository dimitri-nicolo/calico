package main

import (
	"fmt"
	"net/http"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/config"
	"github.com/tigera/voltron/internal/pkg/proxy"
)

func main() {
	cfg := config.Config{}
	if err := envconfig.Process("VOLTRON_AGENT", &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Error("Starting with configuration ", cfg)

	handler := proxy.New(proxy.CreateStaticTargets(), proxy.Path())
	http.Handle("/", handler)

	log.Infof("Targets are: %v", handler.Targets)

	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	log.Infof("Starting web server on %v", url)
	if err := http.ListenAndServe(url, nil); err != nil {
		panic(err)
	}
}
