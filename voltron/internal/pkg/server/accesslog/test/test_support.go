package test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type AccessLogMessage struct {
	Time     time.Time         `json:"ts"`
	TLS      AccessLogTLS      `json:"tls"`
	Request  AccessLogRequest  `json:"req"`
	Response AccessLogResponse `json:"resp"`
}

type AccessLogTLS struct {
	Proto       string `json:"proto"`
	Version     int    `json:"version"`
	ServerName  string `json:"serverName"`
	CipherSuite string `json:"cipherSuite"`
}

type AccessLogRequest struct {
	Time             time.Time     `json:"time"`
	RemoteAddr       string        `json:"remoteAddr"`
	Proto            string        `json:"proto"`
	Method           string        `json:"method"`
	Host             string        `json:"host"`
	Path             string        `json:"path"`
	Query            string        `json:"query"`
	ClusterID        string        `json:"xClusterID"`
	UserAgent        string        `json:"userAgent"`
	Accept           string        `json:"accept"`
	ImpersonateGroup string        `json:"impersonateGroup"`
	Auth             AccessLogAuth `json:"auth"`
}

type AccessLogResponse struct {
	Status       int     `json:"status"`
	BytesWritten int     `json:"bytes"`
	Duration     float64 `json:"duration"`
	Body         string  `json:"body"`
}

type AccessLogAuth struct {
	Iss      string   `json:"iss"`
	Sub      string   `json:"sub"`
	Aud      string   `json:"aud"`
	Sid      string   `json:"sid"`
	Nonce    string   `json:"nonce"`
	Username string   `json:"username"`
	Groups   []string `json:"groups"`
	TenantID string   `json:"ccTenantID"`
}

func ReadAccessLogs(outputFile io.Reader) ([]AccessLogMessage, error) {
	var messages []AccessLogMessage

	scanner := bufio.NewScanner(outputFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		msg := AccessLogMessage{}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func ReadLastAccessLog(logFile *os.File) (AccessLogMessage, error) {
	logs, err := ReadAccessLogs(logFile)
	if err != nil {
		return AccessLogMessage{}, err
	}
	if len(logs) == 0 {
		return AccessLogMessage{}, fmt.Errorf("log file empty")
	}

	return logs[len(logs)-1], nil
}
