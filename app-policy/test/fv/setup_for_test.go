package fv_test

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/stretchr/testify/suite"

	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authzv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"github.com/projectcalico/calico/app-policy/server"
	"github.com/projectcalico/calico/felix/proto"

	fakepolicysync "github.com/projectcalico/calico/app-policy/test/fv/policysync"
)

type dikastesTestSuite struct {
	suite.Suite

	uidAlloc   *UIDAllocator
	dikastes   *server.Dikastes
	policySync *fakepolicysync.FakePolicySync
	cancelFn   context.CancelFunc
}

func (s *dikastesTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel

	tmpdir := s.T().TempDir()
	policySyncListenPath := filepath.Join(tmpdir, "felix")
	dikastesListenPath := filepath.Join(tmpdir, "dikas")

	policySync, err := fakepolicysync.NewFakePolicySync(policySyncListenPath)
	if err != nil {
		s.FailNow("error creating policy sync server for test", err)
		return
	}
	go policySync.Serve(ctx)
	s.policySync = policySync

	dikastes := server.NewDikastesServer(
		server.WithDialAddress("unix", policySync.Addr()),
		server.WithListenArguments("unix", dikastesListenPath),
		server.WithSubscriptionType("per-host-policies"),
	)
	go dikastes.Serve(ctx)
	<-dikastes.Ready
	s.dikastes = dikastes
}

func (s *dikastesTestSuite) TearDownTest() {
	s.cancelFn()
}

type dikastesTestCaseStep struct {
	comment string
	updates []*proto.ToDataplane
	checks  []dikastesTestCaseData
}

type dikastesTestCaseData struct {
	comment      string
	inputReq     *authzv3.CheckRequest
	expectedResp *authzv3.CheckResponse
	expectedErr  error
}

func (d *dikastesTestCaseStep) runAssertions(s *dikastesTestSuite) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10) // tests have a hard limit of 10 secs
	defer cancel()
	// send updates to policy sync for this assertion set
	s.policySync.SendUpdates(d.updates...)

	// we create a grpc client from scratch..
	cc, err := grpc.NewClient(s.dikastes.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		s.FailNowf("failed to init grpc client", "dialing to %s errors with %v", s.dikastes.Addr(), err)
		return
	}
	// that accesses the ext-authz api
	client := authzv3.NewAuthorizationClient(cc)

	<-time.After(time.Millisecond * 1000)

	// iterate thru all request case
	for _, tc := range d.checks {

		// run the 'mock' traffic flowing through ext_authz
		resp, err := client.Check(ctx, tc.inputReq)

		// then see if we get the correct expected response.
		// some notes:
		//   this makes the test output a lot readable if
		//   - we expect a response
		//   - we get a response
		//   - then we only really care about the status
		if tc.expectedResp != nil && resp != nil {
			s.Equal(
				code.Code(tc.expectedResp.Status.Code).String(),
				code.Code(resp.Status.Code).String(),
				fmt.Sprintf("failure on: %s > %s", d.comment, tc.comment),
			)
		} else if tc.expectedResp != nil && resp == nil {
			s.Fail("expected a response, got none!")
		} else { // catch-all comparison
			s.Equal(tc.expectedResp, resp, "responses are different")
		}

		// finally, we assert for an expected error. in most cases, we expect nil (:
		s.Equal(tc.expectedErr, err, fmt.Sprintf("actual error returned: %v", err))
	}
}
