// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package mutator

import (
	"net/http"
)

type ResponseHook interface {
	ModifyResponse(*http.Response) error
}
