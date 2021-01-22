// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package winfv

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/projectcalico/felix/collector"
	"github.com/tigera/windows-networking/pkg/testutils"

	log "github.com/sirupsen/logrus"
)

type WinFV struct {
	rootDir    string
	flowLogDir string
	configFile string

	// The original content of config.ps1.
	originalConfig string
}

func NewWinFV(rootDir, flowLogDir string) (*WinFV, error) {
	configFile := filepath.Join(rootDir, "config.ps1")
	b, err := ioutil.ReadFile(configFile) // just pass the file name
	if err != nil {
		return nil, err
	}

	return &WinFV{
		rootDir:        rootDir,
		flowLogDir:     flowLogDir,
		configFile:     configFile,
		originalConfig: string(b),
	}, nil
}

func (f *WinFV) RestartFelix() {
	log.Infof("Restarting Felix...")
	testutils.Powershell(filepath.Join(f.rootDir, "restart-felix.ps1"))
	log.Infof("Felix Restarted.")
}

func (f *WinFV) RestoreConfig() error {
	err := ioutil.WriteFile(f.configFile, []byte(f.originalConfig), 0644)
	if err != nil {
		return err
	}
	return nil
}

// Add config items to config.ps1.
func (f *WinFV) AddConfigItems(configs map[string]interface{}) error {
	var entry, items string

	items = f.originalConfig
	// Convert config map to string
	for name, value := range configs {
		switch c := value.(type) {
		case int:
			entry = fmt.Sprintf("$env:FELIX_%s = %d", name, c)
		case string:
			entry = fmt.Sprintf("$env:FELIX_%s = %q", name, c)
		default:
			return fmt.Errorf("wrong config value type")
		}

		items = fmt.Sprintf("%s\n%s\n", items, entry)
	}

	err := ioutil.WriteFile(f.configFile, []byte(items), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (f *WinFV) ReadFlowLogs(output string) ([]collector.FlowLog, error) {
	switch output {
	case "file":
		return f.ReadFlowLogsFile()
	default:
		panic("unrecognized flow log output")
	}
}

func (f *WinFV) ReadFlowLogsFile() ([]collector.FlowLog, error) {
	var flowLogs []collector.FlowLog
	flowLogDir := f.flowLogDir
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
			all, _ := ioutil.ReadFile(filePath)
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
