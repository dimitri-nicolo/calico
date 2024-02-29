// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type LogHandler struct {
	logger *log.Logger
}

func New(writers ...io.Writer) *LogHandler {
	formatter := &log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: log.FieldMap{
			log.FieldKeyTime: "@timestamp",
		},
	}

	return &LogHandler{
		logger: &log.Logger{
			Level:     log.WarnLevel,
			Formatter: formatter,
			Out:       io.MultiWriter(writers...),
		},
	}
}

func (l *LogHandler) Process(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		// skip on un-marshalable value
		log.Warnf("cannot marshal log value (%T): %v", v, err)
		return
	}
	var buf log.Fields
	if err := json.Unmarshal(b, &buf); err != nil {
		// skip on un-unmarshalable value
		log.Warnf("cannot marshal log value (%T): %v", v, err)
		return
	}
	l.logger.Log(log.ErrorLevel, buf)
}

func FileWriter(logFilePath string) (io.Writer, error) {
	// Create the log file directory if it doesn't exist.
	dir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		err = fmt.Errorf("cannot create log path %s", logFilePath)
		return nil, err
	}

	// Create or open the log file for appending.
	lf, err := os.OpenFile(
		logFilePath,
		os.O_CREATE|os.O_APPEND|os.O_RDWR,
		0755,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create, append or r/w log path %s", logFilePath)
	}
	return lf, nil
}
