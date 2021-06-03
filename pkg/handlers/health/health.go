package health

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	httpUtils "github.com/tigera/es-gateway/pkg/handlers/internal/common/http"
)

func Health(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
	switch r.Method {
	case http.MethodGet:
		httpUtils.ReturnJSON(w, "OK")
	default:
		http.NotFound(w, r)
	}
}
