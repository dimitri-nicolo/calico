package main

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/server"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "VOLTRON"
)

// Config is a configuration used for Voltron
type config struct {
	LogLevel string `default:"INFO"`
	Port     string `default:"5555"`
}

func main() {
	cfg := config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)

	sender := server.NewSimpleServer("localhost:"+cfg.Port, "localhost:30000")
	sender.Start()
}
