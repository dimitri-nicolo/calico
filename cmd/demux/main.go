package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/caarlos0/env"
	"github.com/tigera/voltron/internal/pkg/config"

	demuxproxy "github.com/tigera/voltron/internal/pkg/proxy"
)

func returnJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Error while encoding %#v", data), 500)
	}
}

//-------------------------------------------------

type demuxProxyHandler struct {
	proxy demuxproxy.DemuxProxy
}

func (dph *demuxProxyHandler) handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		dph.updateTargets(w, r)
	case http.MethodGet:
		dph.listTargets(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (dph *demuxProxyHandler) updateTargets(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error while parsing body", 400)
		return
	}
	// no validations... for now
	// WARNING: there's a race condition in the write to Targets
	dph.proxy.Targets[r.Form["name"][0]], _ = url.Parse(r.Form["target"][0])
	returnJSON(w, r.Form)
}

func (dph *demuxProxyHandler) listTargets(w http.ResponseWriter, r *http.Request) {
	targets := make([]string, 0, len(dph.proxy.Targets))
	for target := range dph.proxy.Targets {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	returnJSON(w, targets)
}

//-------------------------------------------------

func main() {
	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}
	fmt.Printf("Starting with configuration %v\n", cfg)

	proxy := demuxproxy.New(demuxproxy.CreateStaticTargetsForAgent(), demuxproxy.XTarget())
	http.Handle("/", proxy)
	proxyHandler := demuxProxyHandler{proxy: proxy}
	http.HandleFunc("/targets", proxyHandler.handle)

	fmt.Printf("Targets are: %v\n", proxy.Targets)

	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	fmt.Println("Starting web server on", url)
	if err := http.ListenAndServe(url, nil); err != nil {
		panic(err)
	}
}
