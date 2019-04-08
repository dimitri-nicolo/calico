// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	"context"

	"github.com/sirupsen/logrus"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
)

// Run is the entrypoint to start running the reporter.
func Run(
	ctx context.Context, cfg *Config,
	lister list.Destination,
	eventer event.Fetcher,
) error {

	// Create the cross-reference cache that we use to monitor for changes in the relevant data.
	xc := xrefcache.NewXrefCache()
	replayer := replay.New(cfg.Start, cfg.End, lister, eventer, xc)

	r := &reporter{
		ctx: ctx,
		cfg: cfg,
		clog: logrus.WithFields(logrus.Fields{
			"name":  cfg.Name,
			"type":  cfg.ReportType,
			"start": cfg.Start,
			"end":   cfg.End,
		}),
		eventer:  eventer,
		xc:       xc,
		replayer: replayer,
	}
	return r.run()
}

type reporter struct {
	ctx      context.Context
	cfg      *Config
	clog     *logrus.Entry
	listDest list.Destination
	eventer  event.Fetcher
	xc       xrefcache.XrefCache
	replayer syncer.Starter
	data     apiv3.ReportData
}

func (r *reporter) run() error {
	if r.cfg.ReportType.Spec.IncludeEndpointData {
		// We need to include endpoint data in the report.
		r.clog.Debug("Including endpoint data in report")

		// Register the endpoint selectors to specify which endpoints we will receive notification for.
		if err := r.xc.RegisterInScopeEndpoints(r.cfg.Report.Spec.EndpointsSelection); err != nil {
			r.clog.WithError(err).Debug("Unable to register inscope endpoints selection")
			return nil
		}

		// Configure the x-ref cache to spit out the events that we care about (which is basically all the endpoints
		// flagged as "in-scope".
		for _, k := range xrefcache.KindsEndpoint {
			r.xc.RegisterOnUpdateHandler(k, xrefcache.EventInScope, r.onUpdate)
		}

		// Populate the report data from the replayer.
		r.replayer.Start(r.ctx)

		if r.cfg.ReportType.Spec.IncludeEndpointFlowLogData {
			// We also need to include flow logs data for the in-scope endpoints.
			r.clog.Debug("Including flow log data in report")
		}
	}

	if r.cfg.ReportType.Spec.AuditEventsSelection != nil {
		// We need to include audit log data in the report.
		r.clog.Debug("Including audit event data in report")
	}

	// Store report data

	return nil
}

func (r *reporter) onUpdate(update syncer.Update) {

}
