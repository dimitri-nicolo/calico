package main

import (
	"encoding/json"
	"fmt"
	"github.com/tigera/voltron/internal/pkg/config"
	"net/http"
	"net/url"
	"sort"
	"github.com/caarlos0/env"

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

	proxy := demuxproxy.New(demuxproxy.CreateStaticTargetsForAgent(), demuxproxy.XTarget())
	http.Handle("/", proxy)
	proxyHandler := demuxProxyHandler{proxy: proxy}
	http.HandleFunc("/targets", proxyHandler.handle)

	fmt.Println(fmt.Sprintf("Starting web server on %v:%v", cfg.Host, cfg.Port))
	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	if err := http.ListenAndServe(url, nil); err != nil {
		panic(err)
	}
}
