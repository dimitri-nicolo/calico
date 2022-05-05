package bootstrap

import (
	"net"
	"net/http"
	"net/http/pprof"

	log "github.com/sirupsen/logrus"
)

// StartPprofAt starts a pprof server using the given listener
func StartPprofAt(l net.Listener) error {
	srv := new(http.Server)
	debugMux := http.NewServeMux()
	srv.Handler = debugMux
	debugMux.HandleFunc("/debug/pprof/", pprof.Index)
	debugMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	debugMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	debugMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	debugMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	log.Infof("Starting pprof server at %s", l.Addr())
	return srv.Serve(l)
}

// StartPprof starts a pprof server on localhost:56060
func StartPprof() error {
	l, err := net.Listen("tcp", "localhost:56060")
	if err != nil {
		return err
	}

	return StartPprofAt(l)
}
