package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// MuxProxy knows about different destination
// and proxies requests according to some "hook" -- currently hardcoded to query param "target"
type DemuxProxy struct {
	Targets         map[string]*url.URL
	TargetExtractor func(r *http.Request) string
}

// New returns an initialized MuxProxy
func New(targets map[string]*url.URL, targetExtractor func(r *http.Request) string) DemuxProxy {
	return DemuxProxy{Targets: targets, TargetExtractor: targetExtractor}
}

// ServeHTTP knows how to proxy HTTP requests to different named targets
func (mp DemuxProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := mp.TargetExtractor(r)
	if len(target) == 0 {
		http.Error(w, "Could not extract target from request", 400)
		return
	}
	url, ok := mp.Targets[target]
	if !ok {
		http.Error(w, fmt.Sprintf("Configuration missing for target %#v", target), 400)
		return
	}
	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = url.Host

	httputil.NewSingleHostReverseProxy(url).ServeHTTP(w, r)
}
