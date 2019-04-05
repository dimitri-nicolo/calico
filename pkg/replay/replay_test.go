// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package replay_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/elastic"
	. "github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/syncer"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type mockCallbacks struct {
	updates       []syncer.Update
	statusUpdates []syncer.StatusUpdate
}

func (cb *mockCallbacks) OnUpdate(u syncer.Update) {
	cb.updates = append(cb.updates, u)
}

func (cb *mockCallbacks) OnStatusUpdate(su syncer.StatusUpdate) {
	cb.statusUpdates = append(cb.statusUpdates, su)
}

//
// These tests are functional verification (fv) tests
// meaning that a standing elasticsearch database is
// required to run them.
//
// To run locally, you can spin one up quickly using
// `make run-elastic`
//
// TODO: hook up to ci properly with GINKGO_FOCUS filtering
//
var _ = Describe("Replay", func() {
	//
	// The mock data was generated using the testdata/demo.sh script
	// with each kubectl command separated by 10 second intervals.
	// It was exported using the cmd/testdata-exporter binary.
	// using data generated from 4/3/2019 2001 - 2006 UTC
	var (
		//ns  = "compliance-testing"
		ctx = context.Background()

		baseTime = time.Date(2019, 4, 3, 20, 01, 0, 0, time.UTC)
		start    = baseTime.Add(time.Minute)
		end      = start.Add(3 * time.Minute)

		elasticClient = elastic.MustGetElasticClient()
		cb            = new(mockCallbacks)
		replayer      = New(start, end, elasticClient, elasticClient, cb)
	)

	It("Replayer should send both an insync and a complete status update in a complete run through", func() {

		// Make the replay call.
		replayer.Start(ctx)
		Expect(cb.statusUpdates).To(ContainElement(syncer.NewStatusUpdateInSync()))
		Expect(cb.statusUpdates).To(ContainElement(syncer.NewStatusUpdateComplete()))
	})
})
