// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

var Logger logrus.Logger

func init() {
	logrus.Info("WAF logging initialization beginning.")
	defer logrus.Info("WAF logging initialization completed.")

	Logger = *logrus.New()
	Logger.Level = logrus.WarnLevel
	Logger.Formatter = &logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "@timestamp",
		},
	}

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
