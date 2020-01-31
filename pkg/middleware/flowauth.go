package middleware

import (
	"net/http"

	authzv1 "k8s.io/api/authorization/v1"

	lmaauth "github.com/tigera/lma/pkg/auth"
	lmautil "github.com/tigera/lma/pkg/util"
)

// userAuthorizer implements the flows.RBACAuthorizer interface. This should created on a per-request basis as it uses the
// request authentication contexts to perform authorization.  It is a wrapper around the lma K8sAuthInterface to
// encapsulate the originating user request and to handle adding the resource attributes to the request context.
type userAuthorizer struct {
	k8sAuth lmaauth.K8sAuthInterface
	userReq *http.Request
}

func (u *userAuthorizer) Authorize(res *authzv1.ResourceAttributes) (bool, error) {
	req := u.userReq.WithContext(lmautil.NewContextWithReviewResource(u.userReq.Context(), res))
	if status, err := u.k8sAuth.Authorize(req); err != nil {
		// A Forbidden error should just be treated as unauthorized. Any other treat as an error which will terminate
		// processing.
		if status == http.StatusForbidden {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
