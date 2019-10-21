// Copyright (c) 2019 Tigera Inc. All rights reserved.

package job

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"golang.org/x/sync/semaphore"

	"github.com/tigera/intrusion-detection/controller/pkg/alert/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/alert/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
)

type Job interface {
	Run(context.Context)
	Close()
	SetAlert(*v3.GlobalAlert)
}

type job struct {
	alert       *v3.GlobalAlert
	enqueueFunc runloop.EnqueueFunc
	statser     statser.Statser
	controller  controller.Controller
	cancel      context.CancelFunc
	once        sync.Once
	sem         *semaphore.Weighted
}

func NewJob(alert *v3.GlobalAlert, st statser.Statser, controller controller.Controller) Job {
	return &job{
		alert:      alert,
		statser:    st,
		controller: controller,
		sem:        semaphore.NewWeighted(1),
	}
}

func (j *job) Run(ctx context.Context) {
	j.once.Do(func() {
		log.WithFields(log.Fields{
			"name": j.alert.Name,
		}).Debug("Running job")
		ctx, j.cancel = context.WithCancel(ctx)

		var run runloop.RunFunc
		run, j.enqueueFunc = runloop.OnDemand()
		j.SetAlert(j.alert)
		go run(ctx, j.updateAlert)
	})
}

func (j *job) Close() {
	log.WithFields(log.Fields{
		"name": j.alert.Name,
	}).Debug("Closing job")
	j.cancel()
}

func (j *job) SetAlert(alert *v3.GlobalAlert) {
	log.WithFields(log.Fields{
		"name": j.alert.Name,
	}).Debug("Setting alert")
	j.enqueueFunc(alert)
}

func (j *job) updateAlert(ctx context.Context, i interface{}) {
	log.WithFields(log.Fields{
		"name": j.alert.Name,
	}).Debug("Updating alert")
	j.alert = i.(*v3.GlobalAlert)

	body, err := elastic.Watch(*j.alert)
	if err != nil {
		j.statser.Error(statser.CompilationFailed, err)
		return
	}
	j.statser.ClearError(statser.CompilationFailed)

	j.controller.Add(ctx, j.alert.Name, body, func(err error) {
		j.statser.Error(statser.InstallationFailed, err)
	}, j.statser)
	j.statser.ClearError(statser.InstallationFailed)
}
