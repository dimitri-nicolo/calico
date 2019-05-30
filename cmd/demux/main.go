// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/targets"
	"net/http"
	"net/url"
	"sort"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
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
	targets *targets.Targets
}

func (dp demuxProxyHandler) Add(target string, destination *url.URL) {
	dp.targets.Add(target, destination)
}

func (dp demuxProxyHandler) List() []string {
	targets := make([]string, 0, len(dp.targets.List()))
	for target := range dp.targets.List() {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	return targets
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
	name := r.Form["name"][0]
	url, _ := url.Parse(r.Form["target"][0])
	dph.Add(name, url)
	returnJSON(w, r.Form)
}

func (dph *demuxProxyHandler) listTargets(w http.ResponseWriter, r *http.Request) {
	returnJSON(w, dph.List())
}

//-------------------------------------------------

func main() {
	cfg := config.Config{}
	if err := envconfig.Process("VOLTRON", &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting VOLTRON with configuration %v", cfg)

	targets := targets.CreateStaticTargetsForServer()
	log.Infof("Targets are: %v", targets)
	handler := demuxproxy.New(demuxproxy.NewHeaderMatcher(targets, "x-target"))

	http.Handle("/", handler)
	proxyHandler := demuxProxyHandler{targets: targets}
	http.HandleFunc("/targets", proxyHandler.handle)

	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	log.Infof("Starting web server on", url)
	if err := http.ListenAndServe(url, nil); err != nil {
		log.Fatal(err)
	}
}
