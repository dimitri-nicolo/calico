// Copyright (c) 2022 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	_ "embed"
	jsonenc "encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	authzv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/projectcalico/calico/app-policy/flags"
	"github.com/projectcalico/calico/app-policy/internal/testdata"
	"github.com/projectcalico/calico/app-policy/internal/util/testutils"
	fakepolicysync "github.com/projectcalico/calico/app-policy/test/fv/policysync"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/libcalico-go/lib/uds"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

//go:embed tigera.conf
var tigeraConfContents string
var tigeraConfName = "tigera.conf"

func TestRunServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	confPath := filepath.Join(tempDir, tigeraConfName)
	if err := os.WriteFile(confPath, []byte(tigeraConfContents), 0777); err != nil {
		t.Fatalf("Failed to write file %s: %s", tigeraConfName, err)
	}

	listenPath := filepath.Join(t.TempDir(), "dikastes.sock")
	policySyncPath := filepath.Join(t.TempDir(), "nodeagent.sock")
	wafLogFile := filepath.Join(t.TempDir(), "waf.log")

	fps, err := fakepolicysync.NewFakePolicySync(policySyncPath)
	if err != nil {
		t.Fatalf("cannot setup policysync fake %v", err)
		return
	}
	go fps.Serve(ctx)

	config := flags.New()
	args := []string{
		"dikastes", "server",
		"-log-level", "trace",
		"-dial", policySyncPath,
		"-listen", listenPath,
		"-waf-log-file", wafLogFile,
		"-waf-enabled",
		"-waf-ruleset-file", confPath,
		"-waf-directive", "SecRuleEngine On",
		"-waf-events-flush-interval", "500ms",
		"-subscription-type", "per-host-policies",
	}

	if err := config.Parse(args); err != nil {
		t.Fatalf("cannot parse config %v", err)
		return
	}

	ready := make(chan struct{}, 1)
	go runServer(ctx, config, ready)
	<-ready
	fps.SendUpdates(inSync())

	client, err := NewExtAuthzClient(ctx, listenPath)
	if err != nil {
		t.Fatal("cannot create client", err)
		return
	}

	requests := []struct {
		*testutils.CheckRequestBuilder
		expectedCode code.Code
		expectedErr  error
	}{
		{testutils.NewCheckRequestBuilder(), code.Code_OK, nil},
		{testutils.NewCheckRequestBuilder(
			testutils.WithDestinationHostPort("1.1.1.1", 443),
			testutils.WithMethod("GET"),
			testutils.WithHost("my.loadbalancer.address"),
			testutils.WithPath("/cart?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"),
		), code.Code_PERMISSION_DENIED, nil},
		{testutils.NewCheckRequestBuilder(
			testutils.WithDestinationHostPort("2.2.2.2", 443),
			testutils.WithMethod("POST"),
			testutils.WithHost("www.example.com"),
			testutils.WithPath("/vulnerable.php?id=1' waitfor delay '00:00:10'--"),
			testutils.WithScheme("https"),
		), code.Code_PERMISSION_DENIED, nil},
	}

	for _, req := range requests {
		resp, err := client.Check(ctx, req.Value())
		assert.Nil(t, err, "error must not have occurred")
		assert.Equal(t, req.expectedErr, err)
		assert.Equal(t, req.expectedCode, code.Code(resp.Status.Code))
	}
	<-time.After(500 * time.Millisecond)

	f, err := os.Open(wafLogFile)
	assert.Nil(t, err, "error must not have occurred")
	defer f.Close()

	sc := jsonenc.NewDecoder(f)
	sc.DisallowUnknownFields()
	entries := []v1.WAFLog{}
	for sc.More() {
		var log v1.WAFLog
		err := sc.Decode(&log)
		if err != nil {
			t.Error("cannot decode log", err)
			continue
		}
		entries = append(entries, log)
	}
	assert.Equal(t, 2, len(entries), "expected the correct number of logs")
}

func TestRunServeNoPolicySync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenPath := filepath.Join(t.TempDir(), "dikastes.sock")
	policySyncPath := filepath.Join(t.TempDir(), "nodeagent.sock")
	wafLogFile := filepath.Join(t.TempDir(), "waf.log")

	fps, err := fakepolicysync.NewFakePolicySync(policySyncPath)
	if err != nil {
		t.Fatalf("cannot setup policysync fake %v", err)
		return
	}
	go fps.Serve(ctx)

	config := flags.New()
	args := []string{
		"dikastes", "server",
		"-log-level", "trace",
		"-listen", listenPath,
		"-waf-log-file", wafLogFile,
		"-waf-enabled",
		"-waf-events-flush-interval", "500ms",
		"-subscription-type", "per-pod-policies",
	}
	// Add the default embedded directives.
	// e.g. -waf-directive="Include @coraza.conf-recommended", etc
	args = append(
		args,
		testdata.DirectivesToCLI(testdata.DefaultEmbeddedDirectives)...,
	)
	if err := config.Parse(args); err != nil {
		t.Fatalf("cannot parse config %v", err)
		return
	}

	ready := make(chan struct{}, 1)
	go runServer(ctx, config, ready)
	<-ready

	client, err := NewExtAuthzClient(ctx, listenPath)
	if err != nil {
		t.Fatal("cannot create client", err)
		return
	}

	assert.Equal(t, fps.ActiveConnections(), 0, "expected 0 active connection with fake policy sync server")
	requests := []struct {
		*testutils.CheckRequestBuilder
		expectedCode code.Code
		expectedErr  error
	}{
		{testutils.NewCheckRequestBuilder(), code.Code_OK, nil},
		{testutils.NewCheckRequestBuilder(
			testutils.WithDestinationHostPort("1.1.1.1", 443),
			testutils.WithMethod("GET"),
			testutils.WithHost("my.loadbalancer.address"),
			testutils.WithPath("/cart?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"),
		), code.Code_PERMISSION_DENIED, nil},
		{testutils.NewCheckRequestBuilder(
			testutils.WithDestinationHostPort("1.1.1.1", 443),
			testutils.WithMethod("POST"),
			testutils.WithHost("www.example.com"),
			testutils.WithPath("/vulnerable.php?id=1' waitfor delay '00:00:10'--"),
			testutils.WithScheme("https"),
		), code.Code_PERMISSION_DENIED, nil},
	}

	for _, req := range requests {
		resp, err := client.Check(ctx, req.Value())
		assert.Nil(t, err, "error must not have occurred")
		assert.Equal(t, req.expectedErr, err)
		assert.Equal(t, req.expectedCode, code.Code(resp.Status.Code))
	}
	<-time.After(500 * time.Millisecond)

	f, err := os.Open(wafLogFile)
	assert.Nil(t, err, "error must not have occurred")
	defer f.Close()

	sc := jsonenc.NewDecoder(f)
	entries := []v1.WAFLog{}
	for sc.More() {
		var log v1.WAFLog
		err := sc.Decode(&log)
		if err != nil {
			t.Error("cannot decode log", err)
			continue
		}
		entries = append(entries, log)
	}
	// same destination for both 2 requests, so we expect 1 log entry only
	assert.Equal(t, 1, len(entries), "expected 1 logs")
}

func NewExtAuthzClient(ctx context.Context, addr string) (authzv3.AuthorizationClient, error) {
	dialOpts := uds.GetDialOptionsWithNetwork("unix")
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	cc, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	return authzv3.NewAuthorizationClient(cc), nil
}

func inSync() *proto.ToDataplane {
	return &proto.ToDataplane{
		Payload: &proto.ToDataplane_InSync{
			InSync: &proto.InSync{},
		},
	}
}
