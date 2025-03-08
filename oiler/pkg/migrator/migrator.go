// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package migrator

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/oiler/pkg/config"
	"github.com/projectcalico/calico/oiler/pkg/metrics"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator"
)

// Config is the configuration of a migrator that
// tracks its actions via Prometheus metrics
type Config struct {
	primaryLabels   prometheus.Labels
	secondaryLabels prometheus.Labels
	jobLabels       prometheus.Labels
	pageSize        int
	sleepTime       time.Duration
	timeOut         time.Duration
	name            string
	cluster         string
}

func NewConfig(cluster string, cfg config.Config) *Config {
	return &Config{
		primaryLabels:   primaryLabels(cluster, cfg),
		secondaryLabels: secondaryLabels(cluster, cfg),
		jobLabels:       jobLabels(cluster, cfg),
		pageSize:        cfg.ElasticPageSize,
		sleepTime:       cfg.WaitForNewData,
		timeOut:         cfg.ElasticTimeOut,
		name:            cfg.JobName,
		cluster:         cluster,
	}
}

func secondaryLabels(cluster string, cfg config.Config) prometheus.Labels {
	return prometheus.Labels{
		metrics.LabelTenantID:  cfg.SecondaryTenantID,
		metrics.LabelClusterID: cluster,
		metrics.JobName:        cfg.JobName,
		metrics.Source:         "secondary",
	}
}

func primaryLabels(cluster string, cfg config.Config) prometheus.Labels {
	return prometheus.Labels{
		metrics.LabelTenantID:  cfg.PrimaryTenantID,
		metrics.LabelClusterID: cluster,
		metrics.JobName:        cfg.JobName,
		metrics.Source:         "primary",
	}
}

func jobLabels(cluster string, cfg config.Config) prometheus.Labels {
	return prometheus.Labels{
		metrics.LabelClusterID: cluster,
		metrics.JobName:        cfg.JobName,
	}
}

// Migrator will migrate data continuously by reading a time interval
// from primary and writing it to a secondary location, regardless of
// the type of the data
type Migrator[T any] struct {
	Primary   operator.Operator[T]
	Secondary operator.Operator[T]
	Cfg       *Config
}

func (m Migrator[T]) Run(ctx context.Context, current operator.TimeInterval, checkpoints chan operator.TimeInterval) {
	log := logrus.WithFields(logrus.Fields{"cluster": m.Cfg.cluster})

	for {
		select {
		case <-ctx.Done():
			log.Info("Context canceled. Will stop migration")
			return
		default:
			// Reading data from primary location
			list, next, err := m.Read(ctx, current, m.Cfg.pageSize, log)
			if err != nil {
				log.WithError(err).Fatalf("Failed to read data for interval %#v", current)
			}

			// Writing data to secondary location
			err = m.Write(ctx, list.Items, log)
			if err != nil {
				log.WithError(err).Fatal("Failed to write data")
			}

			// Tracking migration metrics
			m.trackMigrationMetrics(current, log)

			// Store periodical checkpoints in case of failure
			select {
			case checkpoints <- current:
				log.Infof("Store last known time interval as a checkpoint: %v", current)
			default:
				log.Info("Skipping storing checkpoint because channel is full")
			}

			// Advance to next interval
			if next != nil {
				current = *next
			}

			// Waiting for new data to be generated
			if next.HasReachedEnd() {
				log.Infof("Will sleep as we need to wait for more data to be generated")
				metrics.WaitForData.With(m.Cfg.jobLabels).Set(1)
				time.Sleep(m.Cfg.sleepTime)
			}
		}
	}
}

func (m Migrator[T]) trackMigrationMetrics(current operator.TimeInterval, log *logrus.Entry) {
	lag := current.Lag(time.Now().UTC())
	lastGeneratedTime := current.LastGeneratedTime()
	metrics.MigrationLag.With(m.Cfg.jobLabels).Set(lag.Round(time.Second).Seconds())
	metrics.LastReadGeneratedTimestamp.With(m.Cfg.jobLabels).Set(float64(lastGeneratedTime.UnixMilli()))
	log.Infof("Migration is behind current time with %s with %s from current %v", lag, lastGeneratedTime, current)
}

func (m Migrator[T]) Write(ctx context.Context, items []T, log *logrus.Entry) error {
	timeOutContext, cancel := context.WithTimeout(ctx, m.Cfg.timeOut)
	defer cancel()

	if len(items) == 0 {
		log.Infof("Will skip write to as there are no items to write")
		return nil
	}

	log.Infof("Writing %d items", len(items))
	startWrite := time.Now().UTC()
	response, err := m.Secondary.Write(timeOutContext, items)
	if err != nil {
		return err
	}

	endWrite := time.Since(startWrite).Seconds()
	metrics.WriteDurationPerClusterIDAndTenantID.With(m.Cfg.secondaryLabels).Observe(endWrite)
	metrics.DocsWrittenPerClusterIDAndTenantID.With(m.Cfg.secondaryLabels).Add(float64(response.Succeeded))
	metrics.FailedDocsWrittenPerClusterIDAndTenantID.With(m.Cfg.secondaryLabels).Add(float64(response.Failed))
	metrics.LastWrittenGeneratedTimestamp.With(m.Cfg.jobLabels).Set(float64(time.Now().UTC().UnixMilli()))

	log.Infof("Finished writing. total=%d, success=%d, failed=%d in %v seconds", response.Total, response.Succeeded, response.Failed, endWrite)

	return nil
}

func (m Migrator[T]) Read(ctx context.Context, current operator.TimeInterval, pageSize int, log *logrus.Entry) (*v1.List[T], *operator.TimeInterval, error) {
	timeOutContext, cancel := context.WithTimeout(ctx, m.Cfg.timeOut)
	defer cancel()

	startRead := time.Now().UTC()
	log.Infof("Reading data for current=%v", current)
	list, next, err := m.Primary.Read(timeOutContext, current, pageSize)

	if err != nil {
		return nil, nil, err
	}

	endReadTime := time.Since(startRead).Seconds()
	log.Infof("Read %d items in %v seconds", len(list.Items), endReadTime)
	log.Infof("Next interval is set to %v", next)
	metrics.ReadDurationPerClusterIDAndTenantID.With(m.Cfg.primaryLabels).Observe(endReadTime)
	metrics.DocsReadPerClusterIDAndTenantID.With(m.Cfg.primaryLabels).Add(float64(len(list.Items)))

	return list, next, err
}
