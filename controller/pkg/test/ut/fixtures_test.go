// Copyright 2019 Tigera Inc. All rights reserved.

package ut

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	oElastic "github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

// This file contains fixtures for elasticsearch tests in this module.  It creates
// a containerized elasticsearch that the tests can use, then runs the tests, and
// finally tears down the container.

var uut *elastic.Elastic
var elasticClient *oElastic.Client

const ElasticsearchImage = "docker.elastic.co/elasticsearch/elasticsearch:7.3.2"

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
	var hCfg *container.HostConfig
	if runtime.GOOS == "darwin" {
		hCfg = &container.HostConfig{
			PublishAllPorts: true,
		}
		cfg.ExposedPorts = nat.PortSet{"9200/tcp": struct{}{}}
	}
	result, err := d.ContainerCreate(ctx, cfg, hCfg, nil, "")
	if err != nil {
		panic("could not create elastic container: " + err.Error())
	}
	defer func() {
		if err := d.ContainerRemove(ctx, result.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "ContainerRemove: %s", err)
		}
	}()

	err = d.ContainerStart(ctx, result.ID, types.ContainerStartOptions{})
	if err != nil {
		panic("could not start elastic: " + err.Error())
	}
	timeout := time.Second * 10
	defer func() {
		if err := d.ContainerStop(ctx, result.ID, &timeout); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "ContainerRemove: %s", err)
		}
	}()

	// get host
	var host string
	j, err := d.ContainerInspect(ctx, result.ID)
	if err != nil {
		panic("could not inspect elastic container: " + err.Error())
	}
	if runtime.GOOS == "darwin" {
		host = "localhost:" + j.NetworkSettings.Ports["9200/tcp"][0].HostPort
	} else {
		host = j.NetworkSettings.IPAddress + ":9200"
	}

	u := &url.URL{
		Scheme: "http",
		Host:   host,
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

	os.Exit(rc)
}
