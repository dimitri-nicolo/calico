package processor

import (
        "context"

	api "github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"

)

const (
    Index           = "tigera_secure_ee_events.cluster"
    PacketCapture   = "capture-honey"
    PcapPath        = "/pcap"
    SnortPath       = "/snort"
)

type HoneypodLogProcessor struct {
    Ctx context.Context
    LogHandler api.AlertLogReportHandler
    Client elastic.Client
}

func NewHoneypodLogProcessor(c elastic.Client, ctx context.Context) (HoneypodLogProcessor, error) {
    return HoneypodLogProcessor{LogHandler: c, Ctx: ctx, Client: c}, nil
}
