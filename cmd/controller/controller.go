package main

import (
	"context"
	"fmt"
	"time"
	"net/url"

	//log "github.com/sirupsen/logrus"
	//rule "github.com/tigera/honeypod-recommendation/pkg/rule"
	//model "github.com/tigera/honeypod-recommendation/pkg/model"
	api "github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"
)

type AlertLogProcessor struct {
    Ctx context.Context
    logHandler api.AlertLogReportHandler
}

type AlertLogProcessorSpec struct {
    AlertLogsSelection *api.AlertLogsSelection `yaml:"alertLogsSelection,omitempty" validate:"omitempty"`
}

func NewAlertLogProcessor(c elastic.Client, ctx context.Context) (AlertLogProcessor, error) {
    return AlertLogProcessor{logHandler: c, Ctx: ctx}, nil
}

func main() {
    fmt.Println("hi")
    //c := elastic.MustGetElasticClient()
    cfg := elastic.MustLoadConfig()
    //cfg := elastic.Config{}
    cfg.ElasticURI = "https://tigera-secure-es-http.tigera-elasticsearch.svc:9200"
    //cfg.ElasticCA = "cert.crt"
    cfg.ParsedElasticURL, _ = url.Parse(cfg.ElasticURI)
    //if err != nil {
    //    fmt.Println("bad url")
    //}
    //cfg,_ = elastic.LoadConfig(cfg)
    c, err := elastic.NewFromConfig(cfg)
    index := "tigera_secure_ee_events.cluster"
    exists, err := c.Backend().IndexExists(index).Do(context.Background())
    if err != nil {
        fmt.Println("err")
        fmt.Println(exists)
    }
    fmt.Println("probly exist")
    fmt.Println(exists)

    ctx := context.Background()

    p, err := NewAlertLogProcessor(c, ctx)
    if err != nil {
       fmt.Println("processor fail")
    }

    //var spec AlertLogProcessorSpec

    fmt.Println("done")


    endTime := time.Now()
    startTime := endTime.Add(-10 * time.Minute)
    for e := range p.logHandler.SearchAlertLogs(ctx, nil, &startTime, &endTime) {
        if e.Err != nil {
		fmt.Println("search fial")
	}
	fmt.Println(e.Type)
	fmt.Println(e.SourceNamespace)
    }
    fmt.Println("done2")


    fmt.Println("get settings")
    settings, err := c.Backend().IndexGetSettings(index).Do(context.Background())
    if err != nil {
        fmt.Println("settings bad")
    }
    indexSettings := settings[index].Settings["index"].(map[string]interface{})
    for key,value := range indexSettings {
	    fmt.Println(key)
	    fmt.Println(value)
    }
}
