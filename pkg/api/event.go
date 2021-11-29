package api

import (
	"context"
	"time"

	"github.com/olivere/elastic/v7"
)

const (
	EventIndexWildCardPattern = "tigera_secure_ee_events.*"
	AutoBulkFlush             = 500
	PaginationSize            = 500
	EventTime                 = "time"
)

type EventBulkProcessor interface {
}

type EventsSearchFields struct {
	Time            int64   `json:"time"`
	Type            string  `json:"type"`
	Description     string  `json:"description"`
	Severity        int     `json:"severity"`
	Origin          string  `json:"origin"`
	SourceIP        *string `json:"source_ip"`
	SourcePort      *int64  `json:"source_port"`
	SourceNamespace string  `json:"source_namespace"`
	SourceName      string  `json:"source_name"`
	SourceNameAggr  string  `json:"source_name_aggr"`
	DestIP          *string `json:"dest_ip"`
	DestPort        *int64  `json:"dest_port"`
	DestNamespace   string  `json:"dest_namespace"`
	DestName        string  `json:"dest_name"`
	DestNameAggr    string  `json:"dest_name_aggr"`
	Host            string  `json:"host"`
}

type EventsData struct {
	EventsSearchFields
	Record interface{} `json:"record"`
}

type EventResult struct {
	*EventsData
	Err error
}

type EventHandler interface {
	// EventsIndexExists checks if index exists, all components sending data to elasticsearch should
	// check if Events index exists during startup.
	EventsIndexExists() (bool, error)

	// CreateEventsIndex is called by every component writing into events index if index doesn't exist.
	CreateEventsIndex() error

	PutSecurityEvent(ctx context.Context, data EventsData) (*elastic.IndexResponse, error)
	PutSecurityEventWithID(ctx context.Context, data EventsData, docId string) (*elastic.IndexResponse, error)
	PutBulkSecurityEvent(data EventsData) error

	SearchSecurityEvents(ctx context.Context, start, end *time.Time, filterData []EventsSearchFields, allClusters bool) <-chan *EventResult

	BulkProcessorInitialize(ctx context.Context, afterFn elastic.BulkAfterFunc) error
	BulkProcessorInitializeWithFlush(ctx context.Context, afterFn elastic.BulkAfterFunc, bulkActions int) error
	BulkProcessorFlush() error
	BulkProcessorClose() error
}
