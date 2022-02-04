// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

const LogPath string = "/var/log/calico/waf"
const LogFile string = "waf.log"

const MaxLogCount = 5
const MaxLogCountDuration = 1 * time.Second

var Logger logrus.Logger

func init() {
	logrus.Info("WAF logging initialization beginning.")
	defer logrus.Info("WAF logging initialization completed.")

	Logger = *logrus.New()
	Logger.Level = logrus.WarnLevel
	Logger.Formatter = NewRateLimitedFormatter(
		&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime: "@timestamp",
			},
		},
		MaxLogCount,
		MaxLogCountDuration,
	)

	if err := os.MkdirAll(LogPath, 0755); err == nil {
		if logfile, err := os.OpenFile(
			filepath.Join(LogPath, LogFile),
			os.O_CREATE|os.O_APPEND|os.O_RDWR,
			0755,
		); err == nil {
			Logger.SetOutput(io.MultiWriter(os.Stdout, logfile))
			return
		}
	}

	logrus.Error("Unable to create WAF log file, logging to stdout only. Elasticsearch logs for WAF may be unavailable.")
	Logger.SetOutput(os.Stdout)
}

func NewRateLimitedFormatter(formatter logrus.Formatter, logCount uint, timeUnit time.Duration) logrus.Formatter {
	limiter := new(rateLimitedFormatter)
	limiter.queue = make(chan bool, logCount)
	limiter.queueSize = logCount
	limiter.queueTime = timeUnit
	limiter.formatter = formatter
	return limiter
}

type rateLimitedFormatter struct {
	queue     chan bool
	queueSize uint
	queueTime time.Duration
	formatter logrus.Formatter
	activated bool
}

func (r *rateLimitedFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	select {
	case r.queue <- true:
		r.activated = false
		go func() {
			<-time.NewTimer(r.queueTime).C
			<-r.queue
		}()
		return r.formatter.Format(entry)
	default:
		// exceeding set limits (# log entries / time period / per node) will drop log entries
		// this is to prevent WAF from flooding Elasticsearch
		if r.activated {
			return nil, nil
		}
		r.activated = true
		return nil, fmt.Errorf("reached log output limiting rate of %d per %s", r.queueSize, r.queueTime)
	}
}
