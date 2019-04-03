// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/list"
)

// Run is the entrypoint to start running the reporter.
func Run(ctx context.Context, cfg *Config) error {
	r := &reporter{
		ctx: ctx,
		cfg: cfg,
		clog: logrus.WithFields(logrus.Fields{
			"name":  cfg.Name,
			"type":  cfg.ReportType,
			"start": cfg.Start,
			"end":   cfg.End,
		}),
	}
	return r.run()
}

type reporter struct {
	ctx      context.Context
	cfg      *Config
	clog     *logrus.Entry
	listDest list.Destination
}

func (r *reporter) run() error {
	return nil
}
