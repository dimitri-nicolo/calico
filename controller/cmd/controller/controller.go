package main

import (
	"context"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/detector"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
)

func main() {
	//log.SetLevel(log.TraceLevel)
	u, err := url.Parse("https://spike-xpack-kadm-es-ms:9200")
	if err != nil {
		panic(err)
	}
	user := "elastic"
	pass := "fQwZr34FNpJbYTyKTI9rEgMai5pq"
	ca := "/home/spike/clusters/spike-xpack/kubeadm/1.6/elastic.ca.pem"
	e := db.NewElastic(u, user, pass, ca)

	s := feed.NewSyncher(e)
	ctx, cancel := context.WithCancel(context.Background())
	s.Sync(ctx)
	log.Info("synching started")

	d := detector.NewDetector(e, e)
	d.RunIPSet(ctx, "abuseipdb", 1*time.Minute)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
}
