package fv_test

import (
	"context"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authzv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"github.com/projectcalico/calico/app-policy/server"
	fakepolicysync "github.com/projectcalico/calico/app-policy/test/fv/policysync"
	"github.com/projectcalico/calico/felix/proto"
)

type dikastesHarness struct {
	policySync *fakepolicysync.FakePolicySync
	dikastes   *server.Dikastes
	cc         *grpc.ClientConn
	client     authzv3.AuthorizationClient
}

type dikastesHarnessResultPair struct {
	response *authzv3.CheckResponse
	err      error
}

func NewDikastesTestHarness(tmpDir string) (*dikastesHarness, error) {
	policySyncListenPath := filepath.Join(tmpDir, "felix")
	dikastesListenPath := filepath.Join(tmpDir, "dikas")

	policySync, err := fakepolicysync.NewFakePolicySync(policySyncListenPath)
	if err != nil {
		return nil, err
	}

	dikastes := server.NewDikastesServer(
		server.WithDialAddress("unix", policySync.Addr()),
		server.WithListenArguments("unix", dikastesListenPath),
		server.WithSubscriptionType("per-host-policies"),
	)

	return &dikastesHarness{policySync: policySync, dikastes: dikastes}, nil
}

func (h *dikastesHarness) Start(ctx context.Context) error {
	go h.policySync.Serve(ctx)
	go h.dikastes.Serve(ctx)
	<-h.dikastes.Ready
	cc, err := grpc.NewClient(h.dikastes.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	h.cc = cc
	h.client = authzv3.NewAuthorizationClient(h.cc)
	return nil
}

func (h *dikastesHarness) Cleanup() error {
	return h.cc.Close()
}

func (h *dikastesHarness) Restart(delay time.Duration) {
	h.policySync.StopAndDisconnect()
	time.Sleep(delay)
	h.policySync.Resume()
}

func (h *dikastesHarness) Checks(ctx context.Context, reqs []*authzv3.CheckRequest) (res []*dikastesHarnessResultPair, err error) {
	for _, req := range reqs {
		response, err := h.client.Check(ctx, req)
		res = append(res, &dikastesHarnessResultPair{response, err})
	}
	return
}

func (h *dikastesHarness) SendUpdates(updates ...*proto.ToDataplane) {
	h.policySync.SendUpdates(updates...)
}
