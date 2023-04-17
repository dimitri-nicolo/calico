// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package processes

import (
	"github.com/projectcalico/calico/linseed/pkg/handler"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

const (
	ProcPath = "/processes"
)

type procHandler struct {
	processes handler.ReadOnlyHandler[v1.ProcessInfo, v1.ProcessParams]
}

func New(procs bapi.ProcessBackend) handler.Handler {
	return &procHandler{
		processes: handler.NewReadOnlyHandler[v1.ProcessInfo, v1.ProcessParams](procs.List),
	}
}

func (h procHandler) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     ProcPath,
			Handler: h.processes.List(),
		},
	}
}
