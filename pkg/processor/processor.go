package processor

import (
        "context"
	_ "fmt"
        _ "time"
	_ "net/url"
	_ "os"
	_ "path/filepath"

	//log "github.com/sirupsen/logrus"
	api "github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"

)

type HoneypodLogProcessor struct {
    Ctx context.Context
    LogHandler api.AlertLogReportHandler
    Client elastic.Client
}

/*
type HoneypodLogProcessorSpec struct {
    AlertLogsSelection *api.AlertLogsSelection `yaml:"alertLogsSelection,omitempty" validate:"omitempty"`
}*/

func NewHoneypodLogProcessor(c elastic.Client, ctx context.Context) (HoneypodLogProcessor, error) {
    return HoneypodLogProcessor{LogHandler: c, Ctx: ctx, Client: c}, nil
}
