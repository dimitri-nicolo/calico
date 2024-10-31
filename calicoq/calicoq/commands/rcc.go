// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

func NewRemoteClusterHandler() *remoteClusterHandler {
	return &remoteClusterHandler{
		status: make(map[model.RemoteClusterStatusKey]model.RemoteClusterStatus),
	}
}

type remoteClusterHandler struct {
	status map[model.RemoteClusterStatusKey]model.RemoteClusterStatus
	lock   sync.Mutex
}

func (r *remoteClusterHandler) OnUpdate(update api.Update) (_ bool) {
	if update.UpdateType == api.UpdateTypeKVDeleted {
		return
	}
	key := update.Key.(model.RemoteClusterStatusKey)
	value := update.Value.(*model.RemoteClusterStatus)
	r.lock.Lock()
	defer r.lock.Unlock()
	r.status[key] = *value
	return
}

// CheckForErrorAndExit checks the status of the remote cluster connections and displays any connection failures
// that could result in the returned data being incomplete. If any errors are found, details are displayed to
// stderr and we exit with non-zero return code. Otherwise, this is a no-op.
func (r *remoteClusterHandler) CheckForErrorAndExit() {
	r.lock.Lock()
	defer r.lock.Unlock()
	var errors []string
	for k, s := range r.status {
		switch s.Status {
		case model.RemoteClusterInSync:
			continue
		case model.RemoteClusterConnecting:
			errors = append(errors, fmt.Sprintf("RemoteClusterConfiguration(%s): still connecting to remote cluster", k.Name))
		case model.RemoteClusterConnectionFailed:
			errors = append(errors, fmt.Sprintf("RemoteClusterConfiguration(%s): connection to remote cluster failed: %s", k.Name, s.Error))
		case model.RemoteClusterResyncInProgress:
			errors = append(errors, fmt.Sprintf("RemoteClusterConfiguration(%s): data synchronization did not complete", k.Name))
		}
	}

	if len(errors) > 0 {
		sort.Strings(errors)
		fmt.Fprintf(os.Stderr, "The following problems were encountered connecting to the remote clusters\n"+
			"which may have resulted in incomplete data:\n-  %s\n", strings.Join(errors, "\n-  "))
		os.Exit(1)
	}
}
