package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/mockdata/replayer"
	"github.com/tigera/compliance/pkg/elastic"
)

func main() {
	// Initialize elastic.
	client, err := elastic.NewFromEnv()
	if err != nil {
		log.WithError(err).Fatal("failed to initialize elastic client")
	}
	if err = client.EnsureIndices(); err != nil {
		log.WithError(err).Fatal("failed to initialize elastic indices")
	}

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
		res, err := client.Backend().Index().
			Index(hit.Index).
			Type(hit.Type).
			Id(hit.Id).
			BodyString(string(*hit.Source)).
			Do(context.Background())
		if err != nil {
			log.WithError(err).Fatal("failed to index document")
		}
		log.WithFields(log.Fields{"id": hit.Id, "result": res}).Info("successfully indexed document")
	}
	log.Info("success")
}
