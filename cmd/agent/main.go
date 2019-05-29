package main

import (
	"fmt"
	"github.com/caarlos0/env"
	"github.com/tigera/voltron/internal/pkg/config"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"net/http"
)

func main() {
	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}
	fmt.Println(fmt.Sprintf("Starting with configuration %v", cfg))

	handler := proxy.New(proxy.CreateStaticTargets(), proxy.Path())
	http.Handle("/",handler)

	fmt.Println(fmt.Sprintf("Targets are: %v", handler.Targets))

	fmt.Println(fmt.Sprintf("Starting web server on %v:%v", cfg.Host, cfg.Port))
	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	if err := http.ListenAndServe(url, nil); err != nil {
		panic(err)
	}
}
