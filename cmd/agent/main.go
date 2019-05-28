package main

import (
	"fmt"
	"net/http"
	"net/url"
	http2 "github.com/tigera/voltron/internal/pkg/http"
)

import (
	"github.com/caarlos0/env"
)

type Config struct {
	Home string `env:"HOME"`
	Port int    `env:"PORT" envDefault:"3000"`
	Host string `env:"HOST" envDefault:"localhost"`
}

func main() {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}
	fmt.Println(fmt.Sprintf("Starting with configuration %v", cfg))

	handler := newForwardHandler()
	http.Handle("/",handler)

	fmt.Println(fmt.Sprintf("Targets are: %v", handler.Targets))

	fmt.Println(fmt.Sprintf("Starting web server on %v:%v", cfg.Host, cfg.Port))
	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	if err := http.ListenAndServe(url, nil); err != nil {
		panic(err)
	}
}

func newForwardHandler() http2.ForwardHandler {
	return http2.ForwardHandler{Targets: createStaticTargets()}
}

func createStaticTargets() map[string]*url.URL {
	targets := make(map[string]*url.URL)

	targets["k8s"] = parse("http://localhost:8001")
	targets["esProxy"] = parse("http://localhost:8002")

	return targets
}

func parse(rawUrl string) *url.URL{
	url, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}

	return url
}