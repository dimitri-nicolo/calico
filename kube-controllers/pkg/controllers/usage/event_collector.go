package usage

import (
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

const defaultReportsPerDay = 4

// newEventCollector collects input events relevant to report generation and outputs them on its event channels.
func newEventCollector(stopCh chan struct{}, informer cache.SharedIndexInformer, usageReportsPerDay int) eventCollector {
	return eventCollector{
		events: events{
			nodeUpdates:         make(chan nodeEvent),
			intervalComplete:    make(chan bool),
			initialSyncComplete: make(chan bool),
		},
		informer:           informer,
		usageReportsPerDay: defaultReportsPerDayIfNecessary(usageReportsPerDay),
		stopIssued:         stopCh,
	}
}

func (c *eventCollector) startCollectingEvents() {
	// Set up the tickers used by the core loop.
	completionTicker := time.NewTicker((24 * time.Hour) / time.Duration(c.usageReportsPerDay))
	checkSyncTicker := time.NewTicker(50 * time.Millisecond)
	defer completionTicker.Stop()
	defer checkSyncTicker.Stop()

	// Wire up the node event handler to the informer. This will feed the node update channel.
	nodeEventHandler := &nodeEventHandler{eventChannel: c.nodeUpdates}
	handlerRegistration, _ := c.informer.AddEventHandler(nodeEventHandler)

	// Watch for events on the tickers. These will feed the interval completion and initial sync channels.
	for {
		select {
		case <-completionTicker.C:
			log.Info("Interval completed")
			mustSend[bool](c.intervalComplete, true)

		case <-checkSyncTicker.C:
			if handlerRegistration.HasSynced() {
				log.Info("Sync received")
				mustSend[bool](c.initialSyncComplete, true)
				checkSyncTicker.Stop()
			}

		case <-c.stopIssued:
			return
		}
	}
}

type eventCollector struct {
	events
	informer           cache.SharedIndexInformer
	stopIssued         chan struct{}
	usageReportsPerDay int
}

func defaultReportsPerDayIfNecessary(reportsPerDay int) int {
	if reportsPerDay <= 0 {
		log.Warningf("Configured usage report per day value (%d) is <= 0. Defaulting to %d reports per day", reportsPerDay, defaultReportsPerDay)
		return defaultReportsPerDay
	}

	return reportsPerDay
}

// nodeEventHandler sends node update events to the provided eventChannel.
type nodeEventHandler struct {
	eventChannel chan nodeEvent
}

type nodeEvent struct {
	old *v1.Node
	new *v1.Node
}

func (n *nodeEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	log.Debugf("Node create event received for %s", obj.(*v1.Node).Name)
	mustSend[nodeEvent](n.eventChannel, nodeEvent{
		old: nil,
		new: obj.(*v1.Node),
	})
}
func (n *nodeEventHandler) OnUpdate(oldObj, newObj interface{}) {
	log.Debugf("Node update event received for %s", newObj.(*v1.Node).Name)
	mustSend[nodeEvent](n.eventChannel, nodeEvent{
		old: oldObj.(*v1.Node),
		new: newObj.(*v1.Node),
	})
}
func (n *nodeEventHandler) OnDelete(obj interface{}) {
	log.Debugf("Node delete event received for %s", obj.(*v1.Node).Name)
	mustSend[nodeEvent](n.eventChannel, nodeEvent{
		old: obj.(*v1.Node),
		new: nil,
	})
}
