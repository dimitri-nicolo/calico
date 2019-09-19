// Copyright 2019 Tigera Inc. All rights reserved.

package ut

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	oElastic "github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

// This file contains fixtures for elasticsearch tests in this module.  It creates
// a containerized elasticsearch that the tests can use, then runs the tests, and
// finally tears down the container.

var uut *elastic.Elastic
var elasticClient *oElastic.Client

const ElasticsearchImage = "docker.elastic.co/elasticsearch/elasticsearch:7.3.0"

func TestMain(m *testing.M) {
	d, err := client.NewEnvClient()
	if err != nil {
		panic("could not create Docker client: " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Pull image
	r, err := d.ImagePull(ctx, ElasticsearchImage, types.ImagePullOptions{})
	if err != nil {
		panic("could not pull image " + ElasticsearchImage + " " + err.Error())
	}
	defer func() { _ = r.Close() }()
	_, err = io.Copy(os.Stdout, r)
	if err != nil {
		panic("could not read pull response: " + err.Error())
	}

	// Create Elastic
	cfg := &container.Config{
		Env:   []string{"discovery.type=single-node"},
		Image: ElasticsearchImage,
	}
	result, err := d.ContainerCreate(ctx, cfg, nil, nil, "")
	if err != nil {
		panic("could not create elastic container: " + err.Error())
	}

	err = d.ContainerStart(ctx, result.ID, types.ContainerStartOptions{})
	if err != nil {
		panic("could not start elastic: " + err.Error())
	}

	// get IP
	j, err := d.ContainerInspect(ctx, result.ID)
	if err != nil {
		panic("could not inspect elastic container: " + err.Error())
	}
	host := j.NetworkSettings.IPAddress

	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:9200", host),
	}

	// Wait for elastic to start responding
	c := http.Client{}
	to := time.After(1 * time.Minute)
	for {
		_, err := c.Get("http://" + u.Host)
		if err == nil {
			break
		}
		select {
		case <-to:
			panic("elasticsearch didn't come up")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	options := []oElastic.ClientOptionFunc{
		oElastic.SetURL(u.String()),
		oElastic.SetErrorLog(log.StandardLogger()),
		oElastic.SetSniff(false),
		oElastic.SetHealthcheck(false),
		//elastic.SetTraceLog(log.StandardLogger()),
	}
	elasticClient, err = oElastic.NewClient(options...)
	if err != nil {
		panic("could not create elasticClient: " + err.Error())
	}

	uut, err = elastic.NewElastic(&http.Client{}, u, "", "")
	if err != nil {
		panic("could not create unit under test: " + err.Error())
	}

	uut.Run(ctx)

	rc := m.Run()

	timeout := time.Second * 10
	_ = d.ContainerStop(ctx, result.ID, &timeout)
	_ = d.ContainerRemove(ctx, result.ID, types.ContainerRemoveOptions{Force: true})

	os.Exit(rc)
}
