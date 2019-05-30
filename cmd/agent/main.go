package main

import (
	"fmt"
	"net/http"

	"github.com/caarlos0/env"
	"github.com/tigera/voltron/internal/pkg/config"
	"github.com/tigera/voltron/internal/pkg/proxy"
)

func main() {
	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}
	fmt.Printf("Starting with configuration %v\n", cfg)

	handler := proxy.New(proxy.CreateStaticTargets(), proxy.Path())
	http.Handle("/", handler)

	fmt.Printf("Targets are: %v\n", handler.Targets)

	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	fmt.Println("Starting web server on", url)
	if err := http.ListenAndServe(url, nil); err != nil {
		panic(err)
	}
}
