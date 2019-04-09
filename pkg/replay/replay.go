package replay

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

type replayer struct {
	resources  map[metav1.TypeMeta]map[string]resources.Resource
	start, end time.Time
	lister     list.Destination
	eventer    event.Fetcher
	cb         syncer.SyncerCallbacks
}

func New(start, end time.Time, lister list.Destination, eventer event.Fetcher, callbacks syncer.SyncerCallbacks) syncer.Starter {
	return &replayer{
		make(map[metav1.TypeMeta]map[string]resources.Resource),
		start, end, lister, eventer, callbacks,
	}
}

// Start will first initialize the replayer to a synced state
//   specified by the start Time, send an in-sync update,
//   replay all the audit events between the start and end Times,
//   and then send a complete update.
func (r *replayer) Start(ctx context.Context) {
	if err := r.initialize(ctx); err != nil {
		r.cb.OnStatusUpdate(syncer.NewStatusUpdateFailed(err))
		return
	}
	r.cb.OnStatusUpdate(syncer.NewStatusUpdateInSync())
	if err := r.replay(ctx, nil, &r.start, &r.end); err != nil {
		r.cb.OnStatusUpdate(syncer.NewStatusUpdateFailed(err))
		return
	}
	r.cb.OnStatusUpdate(syncer.NewStatusUpdateComplete())
}

// Initialize performs the following for all resource types:
// - Retrieve most recent list from before the specified start time.
// - Retrieve events from the list's timestamp up until the specified start time.
// - Replay the retrieve events on top of the list.
func (r *replayer) initialize(ctx context.Context) error {
	for _, rh := range resources.GetAllResourceHelpers() {
		gvk := rh.GroupVersionKind()
		clog := log.WithField("kind", gvk.String())
		clog.Debug("initializing replayer")

		// Allocate map for resource.
		r.resources[gvk] = make(map[string]resources.Resource)

		// Get list for resource.
		list, err := r.lister.RetrieveList(gvk, nil, &r.start, false)
		if err != nil {
			return err
		}
		clog.Debug("retrieved list")

		// Extract the list into an array of runtime.Objects.
		objs, err := meta.ExtractList(list.ResourceList)
		if err != nil {
			return err
		}
		clog.WithField("length", len(objs)).Debug("extracted list into array")

		// Iterate over objects and store into map.
		for i := 0; i < len(objs); i++ {
			res, ok := objs[i].(resources.Resource)
			if !ok {
				clog.WithField("obj", objs[i]).Warn("failed to type assert resource")
				continue
			}
			key := resources.GetNameNamespace(res).String()
			r.resources[gvk][key] = res

			// Send Update to callbacks.
			r.cb.OnUpdate(syncer.Update{Type: syncer.UpdateTypeSet, ResourceID: resources.GetResourceID(res), Resource: res})
		}
		clog.WithField("length", len(r.resources[gvk])).Debug("stored objects into internal cache")

		// Replay events into the internal cache from the list time to the desired start time.
		if err = r.replay(ctx, &gvk, &list.RequestStartedTimestamp.Time, &r.start); err != nil {
			return err
		}
	}
	return nil
}

// replay fetches events for the given resource from the list's timestamp up until the specified start time.
func (r *replayer) replay(ctx context.Context, gvk *metav1.TypeMeta, from, to *time.Time) error {
	for ev := range r.eventer.GetAuditEvents(ctx, gvk, from, to) {
		clog := log.WithFields(log.Fields{"auditID": ev.Event.AuditID, "verb": ev.Event.Verb})
		// Determine proper resource to update for internal cache.
		res, err := event.ExtractResourceFromAuditEvent(ev.Event)
		if err != nil {
			return err
		}

		// Nil resource and nil error means a status object.
		if res == nil {
			clog.Info("passing on a status event")
			continue
		}

		// Update the internal cache and send the appropriate Update to the callbacks.
		key := resources.GetNameNamespace(res).String()
		gvk2 := resources.GetTypeMeta(res)
		update := syncer.Update{ResourceID: resources.GetResourceID(res), Resource: res}
		switch ev.Event.Verb {
		case "create", "update", "patch":
			update.Type = syncer.UpdateTypeSet
			r.resources[gvk2][key] = res
		case "delete":
			update.Type = syncer.UpdateTypeDeleted
			delete(r.resources[gvk2], key)
		}
		clog.WithField("update", update).Debug("replayed event")
		r.cb.OnUpdate(update)
	}
	return nil
}
