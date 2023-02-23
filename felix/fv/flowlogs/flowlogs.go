// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package flowlogs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/collector"
)

func ReadFlowLogs(flowLogDir, output string) ([]collector.FlowLog, error) {
	switch output {
	case "file":
		return ReadFlowLogsFile(flowLogDir)
	default:
		panic("unrecognized flow log output")
	}
}

func ReadFlowLogsFile(flowLogDir string) ([]collector.FlowLog, error) {
	var flowLogs []collector.FlowLog
	log.WithField("dir", flowLogDir).Info("Reading Flow Logs from file")
	filePath := filepath.Join(flowLogDir, collector.FlowLogFilename)
	logFile, err := os.Open(filePath)
	if err != nil {
		return flowLogs, err
	}
	defer logFile.Close()

	s := bufio.NewScanner(logFile)
	for s.Scan() {
		var fljo collector.FlowLogJSONOutput
		err = json.Unmarshal(s.Bytes(), &fljo)
		if err != nil {
			all, _ := os.ReadFile(filePath)
			return flowLogs, fmt.Errorf("Error unmarshaling flow log: %v\nLog:\n%s\nFile:\n%s", err, string(s.Bytes()), string(all))
		}
		fl, err := fljo.ToFlowLog()
		if err != nil {
			return flowLogs, fmt.Errorf("Error converting to flow log: %v\nLog: %s", err, string(s.Bytes()))
		}
		flowLogs = append(flowLogs, fl)
	}
	return flowLogs, nil
}
