package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	contentTypeApplicationCBOR  = "application/cbor"
	contentTypeApplicationSMILE = "application/smile"
	headerContentType           = "Content-Type"
)

// RejectUnacceptableContentTypeHandler returns an HTTP handler which rejects content-type application/cbor
// as a mitigation for CVE-2020-28491.
func RejectUnacceptableContentTypeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header != nil &&
			(strings.EqualFold(r.Header.Get(headerContentType), contentTypeApplicationCBOR) || strings.EqualFold(r.Header.Get(headerContentType), contentTypeApplicationSMILE)) {
			var user string
			usrCtx := r.Context().Value(ESUserKey)
			if usrCtx != nil {
				usr, ok := usrCtx.(*User)
				if ok {
					user = usr.Username
				}
			}
			var message string
			if strings.EqualFold(r.Header.Get(headerContentType), contentTypeApplicationCBOR) {
				message = fmt.Sprintf("A request with header application/cbor was made. This could be a possible attack. "+
					"See CVE-2020-28491 for more information. The request is associated with the user credential for %s.", user)
			} else {
				message = fmt.Sprintf("%s: %s is not supported", headerContentType, contentTypeApplicationCBOR)
			}
			log.Warn(message)
			http.Error(w, fmt.Sprintf("%s: %s is not supported", headerContentType, contentTypeApplicationCBOR), http.StatusUnsupportedMediaType)
			return
		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
