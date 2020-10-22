// Copyright 2020 Tigera Inc. All rights reserved.

package forwarder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
)

const (
	// defaultPollingTimeRange is the default time interval to check for more data to forward.
	defaultPollingTimeRange = 300 * time.Second

	// defaultPollingInterval is the default time interval to check for more data to forward.
	defaultPollingInterval = 300 * time.Second

	// defaultNumForwardingAttempts is the default number of retry forwarding attempts to perform
	// (includes both querying for events and writing them to file).
	defaultNumForwardingAttempts = 10

	// defaultExportLogsDirectory is the default directory path for where logs will be exported.
	defaultExportLogsDirectory = "/var/log/calico/ids"

	// defaultExportLogsMaxFileSizeMB is the default size limit to maintain during log rotation
	// for each log file containing exported data.
	defaultExportLogsMaxFileSizeMB = 50

	// defaultExportLogsMaxFiles is the default max limit for number of files to keep during log
	// rotation for exported data.
	defaultExportLogsMaxFiles = 3

	// logDispatcherFilename specifies the filename to use for writing events to file
	// Note: If we move to 2 or more concurrent workers running, each one must write to a separate file
	logDispatcherFilename = "events.log"
)

var (
	// settings is a package level configuration object for the log forwarder.
	settings = forwarderSettings{}
)

// forwarderSettings contains configuration for how the log forwarder behaves.
type forwarderSettings struct {
	// pollingTimeRange specifies how large (in sec) the time window should be when polling for events.
	pollingTimeRange time.Duration

	// pollingInterval specifies how often (in sec) to check for more data to forward.
	pollingInterval time.Duration

	// numForwardingAttempts specifies how many retry attempts we should perform (for both querying
	// for events and writing events to file).
	numForwardingAttempts uint

	// exportLogsDirectory is the path location of the directory housing exported data.
	exportLogsDirectory string

	// exportLogsMaxFileSizeMB is the max size per file to keep during log rotation for exported data.
	exportLogsMaxFileSizeMB int

	// exportLogsMaxFiles is the max number of files to keep during log rotation for exported data.
	exportLogsMaxFiles int
}

func init() {
	setPollingTimeRange()
	setPollingInterval()
	setNumForwardingAttempts()
	setExportLogsDirectory()
	setExportLogsMaxFileSizeMB()
	setExportLogsMaxFiles()
}

// setPollingTimeRange sets the polling time range based on ENV variable or the default value.
func setPollingTimeRange() {
	settings.pollingTimeRange = defaultPollingTimeRange

	intervalStr := os.Getenv("IDS_FORWARDER_POLLING_TIMERANGE_SECS")
	if intervalStr != "" {
		if intervalInt, err := strconv.Atoi(intervalStr); err != nil {
			log.Panicf("Failed to parse value for polling time range for forwarder %s", intervalStr)
		} else {
			settings.pollingTimeRange = time.Duration(intervalInt) * time.Second
		}
	}
}

// setPollingInterval sets the polling interval based on ENV variable or the default value.
func setPollingInterval() {
	settings.pollingInterval = defaultPollingInterval

	intervalStr := os.Getenv("IDS_FORWARDER_POLLING_INTERVAL_SECS")
	if intervalStr != "" {
		if intervalInt, err := strconv.Atoi(intervalStr); err != nil {
			log.Panicf("Failed to parse value for polling interval for forwarder %s", intervalStr)
		} else {
			settings.pollingInterval = time.Duration(intervalInt) * time.Second
		}
	}
}

// setNumForwardingAttempts sets the polling interval based on ENV variable or the default value.
func setNumForwardingAttempts() {
	settings.numForwardingAttempts = defaultNumForwardingAttempts

	intervalStr := os.Getenv("IDS_FORWARDER_POLLING_NUM_RETRY")
	if intervalStr != "" {
		if intervalInt, err := strconv.ParseUint(intervalStr, 10, 0); err != nil {
			log.Panicf("Failed to parse value for number of polling retries for forwarder %s", intervalStr)
		} else {
			settings.numForwardingAttempts = uint(intervalInt)
		}
	}
}

// setExportLogsDirectory sets the directory where log data will be exported based on ENV variable
// or the default value.
func setExportLogsDirectory() {
	settings.exportLogsDirectory = defaultExportLogsDirectory

	dirStr := os.Getenv("IDS_FORWARDER_LOG_DIR")
	if dirStr != "" {
		settings.exportLogsDirectory = dirStr
	}
}

// setExportLogsMaxFileSizeMB sets the limit for log file size to keep in log rotation for the exported data.
func setExportLogsMaxFileSizeMB() {
	settings.exportLogsMaxFileSizeMB = defaultExportLogsMaxFileSizeMB

	maxFileSizeStr := os.Getenv("IDS_FORWARDER_MAX_FILESIZE_MB")
	if maxFileSizeStr != "" {
		if maxFileSizeInt, err := strconv.Atoi(maxFileSizeStr); err != nil {
			log.Panicf("Failed to parse value for max log file size for forwarder %s", maxFileSizeStr)
		} else {
			settings.exportLogsMaxFileSizeMB = maxFileSizeInt
		}
	}
}

// setExportLogsMaxFiles sets the limit for number of log files to keep in log rotation for the exported data.
func setExportLogsMaxFiles() {
	settings.exportLogsMaxFiles = defaultExportLogsMaxFiles

	maxNumFilesStr := os.Getenv("IDS_FORWARDER_MAX_NUMFILES")
	if maxNumFilesStr != "" {
		if maxNumFilesInt, err := strconv.Atoi(maxNumFilesStr); err != nil {
			log.Panicf("Failed to parse value for max number of log files for forwarder %s", maxNumFilesStr)
		} else {
			settings.exportLogsMaxFiles = maxNumFilesInt
		}
	}
}

// EventForwarder attempts to transport logs (events) from a source data store (Elasticsearch) to destination data store
// by way of log dispatcher (writing to file).
type EventForwarder interface {
	// Run executes forwarding action.
	Run(ctx context.Context)
	// Clean up execution context.
	Close()
}

// eventForwarder queries the source data store for logs and then dispatches them for forwarding to a final
// destination.
type eventForwarder struct {
	// Provides a unique id for this log forward (usual for differentiating when there are multiple instances)
	id string

	// Use a decorated logger so we have some extra metadata.
	logger *logrus.Entry

	once   sync.Once
	cancel context.CancelFunc
	ctx    context.Context

	// Provides access to retrieve events from the source data store
	events db.Events

	// Writes data obtained by the forwarder to logs that will be taken to its final destination
	dispatcher LogDispatcher

	// Maintain state on forwarding process over time.
	config *db.ForwarderConfig
}

// NewEventForwarder sets up a new log forwarder instance and returns it.
// Note: Log forwarder does not currently support concurrency of multiple instances.
func NewEventForwarder(uid string, events db.Events) EventForwarder {
	logrus.WithFields(logrus.Fields{
		"exportLogsDirectory":     settings.exportLogsDirectory,
		"exportLogsMaxFileSizeMB": settings.exportLogsMaxFileSizeMB,
		"exportLogsMaxFiles":      settings.exportLogsMaxFiles,
		"pollingInterval":         settings.pollingInterval,
		"pollingTimeRange":        settings.pollingTimeRange,
		"numForwardingAttempts":   settings.numForwardingAttempts,
	}).Info("Creating new event forwarder")

	dispatcher := NewFileDispatcher(
		settings.exportLogsDirectory,
		logDispatcherFilename,
		settings.exportLogsMaxFileSizeMB,
		settings.exportLogsMaxFiles,
	)

	return &eventForwarder{
		id: uid,
		logger: logrus.WithFields(logrus.Fields{
			"context": "eventforwarder",
			"uid":     uid,
			"logfile": fmt.Sprintf("%s/%s", settings.exportLogsDirectory, logDispatcherFilename),
		}),
		events:     events,
		dispatcher: dispatcher,
	}
}

// QueryError represents an error encountered while querying for events from the data store.
type QueryError struct {
	Err error
}

// Error returns the string representation of the given QueryError
func (e QueryError) Error() string {
	return fmt.Sprintf("Failed to retrieve events: %s", e.Err.Error())
}

// NewQueryError creates a new QueryError.
func NewQueryError(err error) QueryError {
	return QueryError{
		Err: err,
	}
}

// Run performs the log forwarding which includes querying for events from the data store and dispatching those events.
func (f *eventForwarder) Run(ctx context.Context) {
	l := f.logger.WithFields(logrus.Fields{"func": "Run"})

	f.once.Do(func() {
		f.ctx, f.cancel = context.WithCancel(ctx)
		l.Info("Starting alert forwarder ...")

		err := f.dispatcher.Initialize()
		if err != nil {
			log.Errorf("Could not initialize dispatcher (dispatcher) %s", err)
			return
		}

		// Use in-memory field to record progress (time from the last success event to be forwarder), in case save to datastore
		// fails.
		var lastSuccessfulEndTime *time.Time
		// Iterate forever, waiting for settings.pollingInterval seconds between iterations. On each iteration retrieve events
		// and export to logs using dispatcher.
		go runloop.RunLoop(
			f.ctx,
			func() {
				var start time.Time
				var err error

				// 1. Figure out the start time to use for retrieving the next batch of events to forward. We want to
				// continue forwarding events by starting where the last successful run finished.

				// If we have a successufl savepoint already, start from there
				if lastSuccessfulEndTime != nil {
					// Start the next run from the very next time tick (second), because the time range query includes
					// both start and end times.
					start = (*lastSuccessfulEndTime).Add(time.Second)
					l.Debugf("Continuing forwarder from time [%v]", start)
				} else {
					// Otherwise, let's try to recover a savepoint from the config in the datastore ...
					// If we have a savepoint for the last successful run, then use that to determine the time
					// range for the new run.
					f.config, err = f.events.GetForwarderConfig(f.ctx, f.id)
					if err == nil && f.config != nil {
						l.WithFields(log.Fields{
							"forwarderConfig": f.config,
						}).Debugf("Found forwarder config with events.GetForwarderConfig(...)")
						start = *f.config.LastSuccessfulRunEndTime
						l.Debugf("Continuing forwarder from time [%v]", start)
					} else {
						// In the case we don't have a savedpoint for where we left off on the last successful run,
						// we need to pick a starting point. In this special case, we will pick the time range that
						// ends at the current time (time.Now()) and starts -X seconds ago (where X = pollingTimeRange
						// from our setttings).
						start = time.Now().Add(-settings.pollingTimeRange)
						// Start with a blank slate for config
						f.config = &db.ForwarderConfig{}
						l.Debugf("No config detected for forwarder, start from time [%v]", start)
					}
				}

				// 2. Attempt to retrieve next batch of events (using settings time range & last successful end time)
				// and forward
				err = f.retrieveAndForward(start, settings.numForwardingAttempts, time.Second)

				// 3. If current run was successful, then persist the new last successful end time
				if err == nil {
					t := start.Add(settings.pollingTimeRange)
					lastSuccessfulEndTime = &t
					f.config.LastSuccessfulRunEndTime = &t

					l.Infof("Updated forwarder config after successful run [%+v]", f.config)

					// Attempt to back up forwarding progress (in case the forwarder crashes and we need to recover)
					err = retry.Do(
						func() error {
							return f.events.PutForwarderConfig(f.ctx, f.id, f.config)
						},
						retry.Attempts(settings.numForwardingAttempts),
						retry.Delay(500*time.Millisecond),
						retry.OnRetry(
							func(n uint, err error) {
								l.WithError(err).WithFields(log.Fields{
									"forwarderConfig": f.config,
								}).Infof("Retrying forwarder events.PutForwarderConfig(...)")
							},
						),
					)
					// If we were unable to persist the state after retries, we will continune onwards (since we have state
					// in memory). So long as we can get to the next run without crashing we can try to save again.
					if err != nil {
						l.Info("Failed to save forwarder config to datastore, even after retries")
					} else {
						l.Info("Successfully saved forwarder config to datastore")
					}
				} else {
					if _, ok := err.(QueryError); ok {
						// Do nothing ... we need to execute this time range again
					}
				}
			},
			settings.pollingInterval,
		)

	})
}

// Close ensures we handle cleaning up the forwarder context.
func (f *eventForwarder) Close() {
	f.cancel()
}

// retrieveAndForward handles the actual querying for events and forwarding to file.
func (f *eventForwarder) retrieveAndForward(start time.Time, numAttempts uint, delay time.Duration) error {
	l := f.logger.WithFields(logrus.Fields{"func": "retrieveAndForward"})

	// Set the end point of the time range query
	end := start.Add(settings.pollingTimeRange)

	// 1. Attempt to query for security events
	rawEvents := []db.SecurityEvent{}
	var err error
	err = retry.Do(
		func() error {
			var err error
			rawEvents, err = f.events.GetSecurityEvents(f.ctx, start, end)
			return err
		},
		retry.Attempts(numAttempts),
		retry.Delay(delay),
		retry.OnRetry(
			func(n uint, err error) {
				l.WithError(err).WithFields(log.Fields{
					"start": start,
					"end":   end,
				}).Infof("Retrying forwarder events.GetSecurityEvents(...)")
			},
		),
	)
	if err != nil {
		l.WithError(err).WithFields(log.Fields{
			"start": start,
			"end":   end,
		}).Error("Failed events.GetSecurityEvents(...) after %d attempts", numAttempts)
		return NewQueryError(err)
	}

	// We successfully retrieved events (if any) for the given time range
	numEvents := len(rawEvents)
	if numEvents > 0 {
		l.WithFields(log.Fields{"start": start, "end": end}).Debugf("Succesfully retrieved %d events", numEvents)
	} else {
		l.WithFields(log.Fields{"start": start, "end": end}).Debugf("Retrieved no events for this time range")
	}

	// 2. Attempt to write all retrieved events to file
	for i, rawEvent := range rawEvents {
		var err error

		l.WithFields(log.Fields{
			"eventId": rawEvent.ID,
			"event":   string(rawEvent.Data),
		}).Debug("Attempting dispatcher.Dispatch(rawEvent) for event ...")

		// Attempt to write current raw event to file
		err = retry.Do(
			func() error {
				// Calling MarshalJSON should never return an error
				b, _ := rawEvent.Data.MarshalJSON()
				payload := map[string]interface{}{}
				err := json.Unmarshal(b, &payload)
				if err != nil {
					return err
				}
				l.WithFields(log.Fields{
					"payload": payload,
				}).Debug("attempting dispatcher.Dispatch(rawEvent) for event ...")

				if v, ok := payload["time"]; !ok {
					return errors.New("error unmarshalling event payload from datastore")
				} else {
					if t, ok := (v.(string)); ok {
						convertedT, err := strconv.ParseInt(t, 10, 0)
						if err == nil {
							epoch := time.Unix(convertedT, 0)
							f.config.LastSuccessfulEventTime = &epoch
						}
					}
					f.config.LastSuccessfulEventID = &rawEvent.ID
				}
				return f.dispatcher.Dispatch(b)
			},
			retry.Attempts(numAttempts),
			retry.Delay(500*time.Millisecond),
			retry.OnRetry(
				func(n uint, err error) {
					l.WithError(err).WithFields(log.Fields{
						"eventId": rawEvent.ID,
					}).Infof("Retrying forwarder dispatcher.Dispatch(rawEvent) on event %d of %d", i, numEvents)
				},
			),
		)
		// Encountering an error at this point means that we could not write to file for some reason
		// after multiple attempts.
		// Note: We could get fancier here and track exactly which events could not be written to file and
		// retry those.
		if err != nil {
			l.Logger.WithError(err).WithFields(log.Fields{
				"rawEvent": rawEvent,
			}).Error("Forwader failed dispatcher.Dispatch(rawEvent) after %d attempts", numAttempts)
		}
	}
	return nil
}
