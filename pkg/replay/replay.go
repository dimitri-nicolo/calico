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

const (
	VerbCreate = "create"
	VerbUpdate = "update"
	VerbPatch  = "patch"
	VerbDelete = "delete"
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
	log.Info("initializing replayer cache to start time")
	if err := r.initialize(ctx); err != nil {
		r.cb.OnStatusUpdate(syncer.NewStatusUpdateFailed(err))
		return
	}
	r.cb.OnStatusUpdate(syncer.NewStatusUpdateInSync())

	log.Info("replaying audit events to end time")
	if err := r.replay(ctx, nil, &r.start, &r.end, true); err != nil {
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
			res.GetObjectKind().SetGroupVersionKind((&kind).GroupVersionKind())
			id := resources.GetResourceID(res)
			r.resources[kind][id] = res
		}
		clog.Debug("stored objects into internal cache, replaying events to start time")

		// Replay events into the internal cache from the list time to the desired start time.
		if err = r.replay(ctx, &kind, &l.RequestStartedTimestamp.Time, &r.start, false); err != nil {
			return err
		}
		clog.Debug("internal cache replayed to start time, publishing syncer updates")
	}

	// Send Update to callbacks.
	for kind, resList := range r.resources {
		for _, res := range resList {
			log.WithFields(log.Fields{"kind": kind, "resID": resources.GetResourceID(res)}).Debug("publishing syncer updates")
			r.cb.OnUpdate(syncer.Update{Type: syncer.UpdateTypeSet, ResourceID: resources.GetResourceID(res), Resource: res})
		}
	}
	return nil
}

// replay fetches events for the given resource from the list's timestamp up until the specified start time.
func (r *replayer) replay(ctx context.Context, kind *metav1.TypeMeta, from, to *time.Time, notifyUpdates bool) error {
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
		clog = clog.WithFields(log.Fields{"resID": id, "kind": kind2})
		update := syncer.Update{ResourceID: id, Resource: res}

		switch ev.Event.Verb {
		case VerbCreate, VerbUpdate, VerbPatch:
			clog.Debug("setting event")
			update.Type = syncer.UpdateTypeSet

			// Refuse to apply audit event if resource version of old resource is higher
			//  than the new one.
			oldRes, ok := r.resources[kind2][id]
			if ok {
				oldResVer, err := resources.GetResourceVersion(oldRes)
				if err != nil {
					clog.Warn("Failed to convert resourceVersion to number. Refusing to process event.")
					continue
				}
				newResVer, err := resources.GetResourceVersion(res)
				if err != nil {
					clog.Warn("Failed to convert resourceVersion to number. Refusing to process event.")
					continue
				}
				if oldResVer > newResVer {
					clog.Info("Resource version conflict detected. Refusing to process event.")
					continue
				}
			}

			r.resources[kind2][id] = res
		case VerbDelete:
			clog.Debug("deleting event")
			update.Type = syncer.UpdateTypeDeleted
			delete(r.resources[kind2], id)
		default:
			clog.Warn("invalid verb")
		}
		if notifyUpdates {
			r.cb.OnUpdate(update)
		}
	}
	return nil
}
