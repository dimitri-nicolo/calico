package middleware

import (
	"fmt"
	"net/http"

	"github.com/prometheus/common/log"
	authzv1 "k8s.io/api/authorization/v1"

	lmaauth "github.com/tigera/lma/pkg/auth"
)

// userAuthorizer implements the flows.RBACAuthorizer interface. This should created on a per-request basis as it uses the
// request authentication contexts to perform authorization.  It is a wrapper around the lma K8sAuthInterface to
// encapsulate the originating user request and to handle adding the resource attributes to the request context.
type userAuthorizer struct {
	mcmAuth MCMAuth
	userReq *http.Request
}

// The key type is unexported to prevent collisions
type key int

const (
	// clusterKey is the context key for the target cluster of a request.
	clusterKey key = iota
)

func (u *userAuthorizer) Authorize(res *authzv1.ResourceAttributes) (bool, error) {
	cluster := u.userReq.Context().Value(clusterKey)
	log.Debugf("Authorizing request for cluster %v", cluster)
	k8sauth := u.mcmAuth.K8sAuth(fmt.Sprintf("%v", cluster))

	req := u.userReq.WithContext(lmaauth.NewContextWithReviewResource(u.userReq.Context(), res))
	if status, err := k8sauth.Authorize(req); err != nil {
		// A Forbidden error should just be treated as unauthorized. Any other treat as an error which will terminate
		// processing.
		if status == http.StatusForbidden {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
