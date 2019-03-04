package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/detector"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
)

const (
	DefaultElasticScheme = "http"
	DefaultElasticHost   = "elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local"
	DefaultElasticPort   = 9200
	DefaultElasticUser   = "elastic"
)

func main() {
	var u *url.URL
	uri := os.Getenv("ELASTIC_URI")
	if uri != "" {
		var err error
		u, err = url.Parse(uri)
		if err != nil {
			panic(err)
		}
	} else {
		scheme := os.Getenv("ELASTIC_SCHEME")
		if scheme == "" {
			scheme = DefaultElasticScheme
		}

		host := os.Getenv("ELASTIC_HOST")
		if host == "" {
			host = DefaultElasticHost
		}

		portStr := os.Getenv("ELASTIC_PORT")
		var port int64
		if portStr == "" {
			port = DefaultElasticPort
		} else {
			var err error
			port, err = strconv.ParseInt(portStr, 10, 16)
			if err != nil {
				panic(err)
			}
		}

		u = &url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", host, port),
		}
	}

	//log.SetLevel(log.TraceLevel)
	user := os.Getenv("ELASTIC_USER")
	if user == "" {
		user = DefaultElasticUser
	}
	pass := os.Getenv("ELASTIC_PASSWORD")
	ca := os.Getenv("ELASTIC_CA")
	e := db.NewElastic(u, user, pass, ca)

	s := feed.NewSyncher(e)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Sync(ctx)
	log.Info("synching started")

	d := detector.NewDetector(e, e)
	d.RunIPSet(ctx, "abuseipdb", 1*time.Minute)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
