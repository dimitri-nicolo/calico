package main

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	lsrest "github.com/projectcalico/calico/linseed/pkg/client/rest"
)

const (
	Timeout          = 30 * time.Second
	auditLimit       = 1
	epoch            = "1970-01-01T00:00:00Z" //RFC3339
	esTimestampField = "timestamp"
)

func GetLinseedClient(cfg *Config) (lsclient.Client, error) {

	var err error
	var linseedClient lsclient.Client
	lsConfig := lsrest.Config{
		URL:            cfg.LinseedURL,
		CACertPath:     cfg.LinseedCA,
		ClientKeyPath:  cfg.LinseedClientKey,
		ClientCertPath: cfg.LinseedClientCert,
	}
	if linseedClient, err = lsclient.NewClient(cfg.TenantID, lsConfig, lsrest.WithTokenPath(cfg.LinseedToken)); err == nil {
		return linseedClient, err
	}
	return nil, err
}

// Returns the last available log timestamp for audit log.
// We use thus retrieved timestamp to get the logstream not retrieved so far rather than starting from scratch.
func GetStartTime(cfg *Config, linseedClient lsclient.Client) (int64, error) {

	// We are trying to find the latest timestamp available in ES data.
	// To do this, we get the Top 1 (i.e. first) result from the data
	// sorted in descending order of timestamp using audit index.
	params := v1.AuditLogParams{
		QueryParams: v1.QueryParams{
			MaxPageSize: auditLimit,
		},
		Type: v1.AuditLogTypeKube,
		Sort: []v1.SearchRequestSortBy{{Field: esTimestampField, Descending: true}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	auditResult, err := linseedClient.AuditLogs("").List(ctx, &params)
	if err != nil {
		log.Error("error while fetching latest audit index data timestamp from linseed.")
		return 0, err
	}

	var auditLogs v1.AuditLog
	if auditResult.TotalHits > 0 {
		auditLogs = auditResult.Items[0]

		// Convert timestamp to number of milliseconds since Epoch.
		epochTime, err := time.Parse(time.RFC3339, epoch)
		if err != nil {
			log.Errorf("error parsing epoch timestamp")
			return 0, err
		}
		duration := auditLogs.StageTimestamp.Sub(epochTime)
		log.Infof("start-time: %v", duration.Milliseconds())
		return duration.Milliseconds(), nil
	}

	// Elastic doesn't have any matching entry at the very first run.
	// don't error out, silently log with 0 value.
	log.Info("linseed didn't return any result, assuming no logs exist yet.")
	return 0, nil
}
