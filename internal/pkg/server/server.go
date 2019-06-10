package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	demuxproxy "github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/targets"
)

// Server is the voltron server that accepts tunnels from the app clusters. It
// serves HTTP requests and proxies them to the tunnels.
type Server struct {
	http      *http.Server
	proxyMux  *http.ServeMux
	proxyTgts *targets.Targets

	certFile string
	keyFile  string
}

// New returns a new Server
func New(opts ...Option) (*Server, error) {
	srv := &Server{
		http:      new(http.Server),
		proxyTgts: targets.NewEmpty(),
	}

	for _, o := range opts {
		if err := o(srv); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	log.Infof("Targets are: %s", srv.proxyTgts)
	srv.proxyMux = http.NewServeMux()
	srv.http.Handler = srv.proxyMux

	srv.proxyMux.Handle("/", demuxproxy.New(demuxproxy.NewHeaderMatcher(srv.proxyTgts, "x-target")))
	proxyHandler := demuxProxyHandler{targets: srv.proxyTgts}
	srv.proxyMux.HandleFunc("/targets", proxyHandler.handle)

	return srv, nil
}

// ListenAndServeHTTP starts listening and serving HTTP requests
func (s *Server) ListenAndServeHTTP() error {
	return s.http.ListenAndServe()

}

// ServeHTTP starts serving HTTP requests
func (s *Server) ServeHTTP(lis net.Listener) error {
	return s.http.Serve(lis)
}

// ListenAndServeHTTPS starts listening and serving HTTPS requests
func (s *Server) ListenAndServeHTTPS() error {
	return s.http.ListenAndServeTLS(s.certFile, s.keyFile)
}

// ServeHTTPS starts serving HTTPS requests
func (s *Server) ServeHTTPS(lis net.Listener) error {
	return s.http.ServeTLS(lis, s.certFile, s.keyFile)
}

// Close stop the server
func (s *Server) Close() error {
	return s.http.Close()
}

func returnJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Error while encoding %#v", data), 500)
	}
}

type demuxProxyHandler struct {
	targets *targets.Targets
}

func (dph demuxProxyHandler) Add(target string, destination string) error {
	return dph.targets.Add(target, destination)
}

func (dph demuxProxyHandler) List() []string {
	targets := make([]string, 0, len(dph.targets.List()))
	for target := range dph.targets.List() {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	return targets
}

func (dph *demuxProxyHandler) handle(w http.ResponseWriter, r *http.Request) {
	log.Debugf("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
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
	url := r.Form["target"][0]
	dph.Add(name, url)
	log.Debugf("New target name=%s target=%s", name, url)
	returnJSON(w, r.Form)
}

func (dph *demuxProxyHandler) listTargets(w http.ResponseWriter, r *http.Request) {
	returnJSON(w, dph.List())
}
