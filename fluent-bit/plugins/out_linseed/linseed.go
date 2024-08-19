// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

import "C"

var cfg *Config

//export FLBPluginRegister
func FLBPluginRegister(def unsafe.Pointer) int {
	configureLogging()

	return output.FLBPluginRegister(def, "linseed", "Calico Enterprise linseed output plugin")
}

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	var err error

	cfg, err = NewConfig(plugin, output.FLBPluginConfigKey)
	if err != nil {
		logrus.WithError(err).Error("failed to create config")
		return output.FLB_ERROR
	}

	fields := logrus.Fields{
		"endpoint":       cfg.endpoint,
		"serviceaccount": cfg.serviceAccountName,
	}
	logrus.WithFields(fields).Info("linseed output plugin initialized")
	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	// process fluent-bit internal messagepack buffer
	processor := NewRecordProcessor()
	ndjsonBuffer, count, err := processor.Process(data, int(length))
	if err != nil {
		logrus.WithError(err).Error("failed to process record data")
		return output.FLB_ERROR
	}

	// post to ingestion endpoint
	t := C.GoString(tag)
	url := fmt.Sprintf("%s/ingestion/api/v1/%s/logs/bulk", cfg.endpoint, t)
	logrus.Debugf("sending %s logs to %q", t, url)
	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewBuffer(ndjsonBuffer.Bytes())))
	if err != nil {
		logrus.WithError(err).Error("failed to create http request")
		return output.FLB_ERROR
	}

	token, err := GetToken(cfg)
	if err != nil {
		logrus.WithError(err).Error("failed to get authorization token")
		return output.FLB_ERROR
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/x-ndjson")

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.insecureSkipVerify,
		},
	}}
	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("failed to send http request")
		return output.FLB_ERROR
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("error response from server %q", resp.Status)
		return output.FLB_ERROR
	}

	logrus.Infof("successfully sent %d logs", count)
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	return output.FLB_OK
}

func configureLogging() {
	logutils.ConfigureFormatter("linseed")
	logrus.SetOutput(os.Stdout)

	logLevel := logrus.InfoLevel
	rawLogLevel := os.Getenv("LOG_LEVEL")
	if rawLogLevel != "" {
		parsedLevel, err := logrus.ParseLevel(rawLogLevel)
		if err == nil {
			logLevel = parsedLevel
		} else {
			logrus.WithError(err).Warnf("failed to parse log level %q, defaulting to INFO.", parsedLevel)
		}
	}

	logrus.SetLevel(logLevel)
	logrus.Infof("log level set to %q", logLevel)
}

func main() {
}
