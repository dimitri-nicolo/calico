package replay

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

type replayer struct {
	resources  map[metav1.TypeMeta]map[apiv3.ResourceID]resources.Resource
	start, end time.Time
	lister     list.Destination
	eventer    event.Fetcher
	cb         syncer.SyncerCallbacks
}

func New(start, end time.Time, lister list.Destination, eventer event.Fetcher, callbacks syncer.SyncerCallbacks) syncer.Starter {
	return &replayer{
		make(map[metav1.TypeMeta]map[apiv3.ResourceID]resources.Resource),
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
		kind := rh.TypeMeta()
		clog := log.WithField("kind", kind.String())
		clog.Debug("initializing replayer")

		// Allocate map for resource.
		r.resources[kind] = make(map[apiv3.ResourceID]resources.Resource)

		// Get list for resource.
		l, err := r.lister.RetrieveList(kind, nil, &r.start, false)
		if err != nil {
			return err
		}
		clog.Debug("retrieved list")

		// Extract the list into an array of runtime.Objects.
		objs, err := meta.ExtractList(l.ResourceList)
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
			id := resources.GetResourceID(res)
			r.resources[kind][id] = res

			//TODO(rlb): Why are we sending updates from before the start time? We should only be populating
			// our cache up to this point. When we have finished populating our cache we should be dumping the
			// entire contents as the "starting point".

			// Send Update to callbacks.
			r.cb.OnUpdate(syncer.Update{Type: syncer.UpdateTypeSet, ResourceID: id, Resource: res})
		}
		clog.WithField("length", len(r.resources[kind])).Debug("stored objects into internal cache")

		// Replay events into the internal cache from the list time to the desired start time.
		if err = r.replay(ctx, &kind, &l.RequestStartedTimestamp.Time, &r.start); err != nil {
			return err
		}
	}
	return nil
}

//TODO(rlb): For resources with multiple possible versions, we need to list all audit events across all versions
//TODO       and order those event by time (since all versions may be used simultaneously by different clients)

// replay fetches events for the given resource from the list's timestamp up until the specified start time.
func (r *replayer) replay(ctx context.Context, kind *metav1.TypeMeta, from, to *time.Time) error {
	for ev := range r.eventer.GetAuditEvents(ctx, kind, from, to) {
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
		id := resources.GetResourceID(res)

		kind2 := resources.GetTypeMeta(res)
		update := syncer.Update{ResourceID: id, Resource: res}
		switch ev.Event.Verb {
		case "create", "update", "patch":
			update.Type = syncer.UpdateTypeSet
			r.resources[kind2][id] = res
		case "delete":
			update.Type = syncer.UpdateTypeDeleted
			delete(r.resources[kind2], id)
		}
		clog.WithField("update", update).Debug("replayed event")
		r.cb.OnUpdate(update)
	}
	return nil
}
