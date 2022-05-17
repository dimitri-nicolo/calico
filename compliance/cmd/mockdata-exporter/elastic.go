package main

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/compliance/mockdata/replayer"
	"github.com/projectcalico/calico/lma/pkg/elastic"
)

const (
	maxRetriesPerIndex = 10
)

func main() {

	// Initialize elastic.
	client := elastic.MustGetElasticClient()

	// Retrieve the testdata.
	eeEvents, err := replayer.GetEEAuditEventsDoc()
	if err != nil {
		log.WithError(err).Fatal("failed to initialize elastic client")
	}
	kubeEvents, err := replayer.GetKubeAuditEventsDoc()
	if err != nil {
		log.WithError(err).Fatal("failed to initialize elastic client")
	}
	listEvents, err := replayer.GetListsDoc()
	if err != nil {
		log.WithError(err).Fatal("failed to initialize elastic client")
	}

	// Dump into elastic.
	for _, hit := range append(eeEvents.Hits.Hits, append(kubeEvents.Hits.Hits, listEvents.Hits.Hits...)...) {
		for r := 0; ; r++ {
			res, err := client.Backend().Index().
				Index(hit.Index).
				Id(hit.Id).
				BodyString(string(hit.Source)).
				Do(context.Background())
			if err == nil {
				log.WithFields(log.Fields{"id": hit.Id, "result": res}).Info("successfully indexed document")
				break
			}
			if r >= maxRetriesPerIndex {
				log.WithError(err).Fatal("failed to index document")
			}
			log.WithError(err).Info("failed to index document - retrying")
			time.Sleep(time.Second)
		}
	}
	log.Info("success")
}
