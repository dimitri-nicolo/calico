// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf

import (
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
	Logger.Formatter = newRateLimitedFormatter(
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

func newRateLimitedFormatter(formatter logrus.Formatter, logCount int, timeUnit time.Duration) logrus.Formatter {
	limiter := new(rateLimitedFormatter)
	limiter.formatter = formatter
	limiter.queueSize = make(chan bool, logCount)
	limiter.queueTime = timeUnit
	return limiter
}

type rateLimitedFormatter struct {
	formatter logrus.Formatter
	queueSize chan bool
	queueTime time.Duration
}

func (r *rateLimitedFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	select {
	case r.queueSize <- true:
		go func() {
			<-time.NewTimer(r.queueTime).C
			<-r.queueSize
		}()
		return r.formatter.Format(entry)
	default:
		// exceeding set limits (# log entries / time period / per node)
		// will drop log entries; this is to prevent Elasticsearch flooding by WAF
		return nil, nil
	}
}
