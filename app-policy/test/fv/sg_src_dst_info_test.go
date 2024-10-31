package fv_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	authzv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/projectcalico/calico/app-policy/server"
	fakepolicysync "github.com/projectcalico/calico/app-policy/test/fv/policysync"
	"github.com/projectcalico/calico/felix/proto"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type dikastesSrcDstInfoTestSuite struct {
	suite.Suite

	uidAllocator *UIDAllocator

	tmpDir,
	dstDikastesLogfile string

	policySyncNode1 *fakepolicysync.FakePolicySync
	dstDikastes     *server.Dikastes
}

func (s *dikastesSrcDstInfoTestSuite) SetupTest() {
	s.uidAllocator = NewUIDAllocator()

	s.tmpDir = s.T().TempDir()

	var err error
	s.policySyncNode1, s.dstDikastes, s.dstDikastesLogfile, err = s.setupPolicySyncAndDikastes("n1", "dst")
	if err != nil {
		s.FailNow("error creating policy SRC sync server for test", err)
		return
	}
}

func (s *dikastesSrcDstInfoTestSuite) setupPolicySyncAndDikastes(policySyncName, dikastesName string) (*fakepolicysync.FakePolicySync, *server.Dikastes, string, error) {
	const (
		listenNetwork    = "unix"
		subscriptionType = "per-host-policies"
	)

	policySyncListenPath := filepath.Join(s.tmpDir, policySyncName)
	dikastesListenPath := filepath.Join(s.tmpDir, dikastesName)

	policySync, err := fakepolicysync.NewFakePolicySync(policySyncListenPath)
	if err != nil {
		return nil, nil, "", err
	}

	dikastesLogfile := filepath.Join(s.tmpDir, dikastesName+".log")
	dikastes := server.NewDikastesServer(
		server.WithDialAddress("unix", policySync.Addr()),
		server.WithListenArguments(listenNetwork, dikastesListenPath),
		server.WithSubscriptionType(subscriptionType),
		server.WithWAFConfig(true, dikastesLogfile, []string{}, []string{
			"Include @coraza.conf-recommended",
			"Include @crs-setup.conf.example",
			"Include @owasp_crs/*.conf",
			"SecRuleEngine DetectionOnly",
		}),
		server.WithWAFFlushDuration(time.Millisecond*300),
	)

	return policySync, dikastes, dikastesLogfile, nil
}

func (s *dikastesSrcDstInfoTestSuite) createExtAuthzClientFromDikastes(d *server.Dikastes) (authzv3.AuthorizationClient, error) {
	cc, err := grpc.NewClient(d.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return authzv3.NewAuthorizationClient(cc), nil
}

func (s *dikastesSrcDstInfoTestSuite) TearDownTest() {
}

func (s *dikastesSrcDstInfoTestSuite) TestSameNode() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.policySyncNode1.Serve(ctx)
	go s.dstDikastes.Serve(ctx)
	<-s.dstDikastes.Ready

	srcIp := "10.0.0.24"
	dstIp := "10.0.0.48"

	s.policySyncNode1.SendUpdates([]*proto.ToDataplane{
		wepUpdate("default/src-pod", []string{srcIp}, []string{"default"}),
		wepUpdate("default/dst-pod", []string{dstIp}, []string{"default"}),
		inSync(),
	}...)

	expectedIPs := map[string]bool{
		srcIp: false,
		dstIp: false,
	}
	s.waitForEndpoints(s.dstDikastes, expectedIPs)

	ec, err := s.createExtAuthzClientFromDikastes(s.dstDikastes)
	s.NoError(err)

	_, err = ec.Check(
		ctx,
		newRequest(
			s.uidAllocator.NextUID(),
			"POST", "http://my.loadbalancer.info/cart?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user",
			map[string]string{},
			newPeer("10.0.0.24", "default", "default"),
			newPeer("10.0.0.48", "default", "default"),
		),
	)
	s.NoError(err)

	// read the log file
	var logs []*v1.WAFLog
	<-time.After(time.Millisecond * 1500)
	s.Eventually(func() bool {
		l, _ := readLogs(s.dstDikastesLogfile)
		logs = append(logs, l...)
		return len(logs) > 0
	}, time.Duration(time.Second*5), time.Millisecond*1000)
	s.NoError(err)
	s.Len(logs, 1)
	if len(logs) > 0 {
		srcEntry := logs[0].Source
		s.Equal(srcEntry.IP, "10.0.0.24")
		s.Equal(srcEntry.PodName, "src-pod")
		s.Equal(srcEntry.PodNameSpace, "default")

		dstEntry := logs[0].Destination
		s.Equal(dstEntry.IP, "10.0.0.48")
		s.Equal(dstEntry.PodName, "dst-pod")
		s.Equal(dstEntry.PodNameSpace, "default")
	}
}

func (s *dikastesSrcDstInfoTestSuite) TestDiffNode() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node1ps, srcDikastes, _, err := s.setupPolicySyncAndDikastes("n0", "src")
	if err != nil {
		s.FailNow("error creating policy SRC sync server for test", err)
		return
	}

	go node1ps.Serve(ctx)
	go srcDikastes.Serve(ctx)
	<-srcDikastes.Ready

	srcIp := "10.0.0.24"
	node1ps.SendUpdates([]*proto.ToDataplane{
		wepUpdate("default/src-pod", []string{srcIp}, []string{"default"}),
		inSync(),
	}...)

	expectedSrc := map[string]bool{srcIp: false}
	s.waitForEndpoints(srcDikastes, expectedSrc)

	sec, err := s.createExtAuthzClientFromDikastes(srcDikastes)
	s.NoError(err)
	s.NotNil(sec)

	resp, err := sec.Check(
		ctx,
		newRequest(
			s.uidAllocator.NextUID(),
			"POST", "http://my.loadbalancer.info/cart?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user",
			map[string]string{},
			newPeer("10.0.0.24", "default", "default"),
			newPeer("10.0.0.48", "default", "default"),
		),
	)
	s.NoError(err)
	okResponse, ok := resp.HttpResponse.(*authzv3.CheckResponse_OkResponse)
	s.True(ok)
	s.Len(okResponse.OkResponse.Headers, 2)
	foundHeaders := map[string]string{}
	for _, h := range okResponse.OkResponse.Headers {
		foundHeaders[h.Header.Key] = h.Header.Value
	}
	s.Equal(foundHeaders, map[string]string{
		"x-source-workload-name":      "src-pod",
		"x-source-workload-namespace": "default",
	})

	go s.policySyncNode1.Serve(ctx)
	go s.dstDikastes.Serve(ctx)
	<-s.dstDikastes.Ready

	destIp := "10.0.0.48"
	s.policySyncNode1.SendUpdates([]*proto.ToDataplane{
		wepUpdate("default/dst-pod", []string{destIp}, []string{"default"}),
		inSync(),
	}...)

	expectedDest := map[string]bool{destIp: false}
	s.waitForEndpoints(s.dstDikastes, expectedDest)

	ec, err := s.createExtAuthzClientFromDikastes(s.dstDikastes)
	s.NoError(err)

	_, err = ec.Check(
		ctx,
		newRequest(
			s.uidAllocator.NextUID(),
			"POST", "http://my.loadbalancer.info/cart?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user",
			foundHeaders, // inject headers to simulate previous leg of the request
			newPeer("10.0.0.24", "default", "default"),
			newPeer("10.0.0.48", "default", "default"),
		),
	)
	s.NoError(err)

	// read the log file
	var logs []*v1.WAFLog
	<-time.After(time.Millisecond * 1500)
	s.Eventually(func() bool {
		l, _ := readLogs(s.dstDikastesLogfile)
		logs = append(logs, l...)
		return len(logs) > 0
	}, time.Duration(time.Second*5), time.Millisecond*1000)
	s.NoError(err)
	s.Len(logs, 1)
	if len(logs) > 0 {
		srcEntry := logs[0].Source
		s.Equal(srcEntry.IP, "10.0.0.24")
		s.Equal(srcEntry.PodName, "src-pod")
		s.Equal(srcEntry.PodNameSpace, "default")

		dstEntry := logs[0].Destination
		s.Equal(dstEntry.IP, "10.0.0.48")
		s.Equal(dstEntry.PodName, "dst-pod")
		s.Equal(dstEntry.PodNameSpace, "default")
	}
}

func readLogs(logfile string) ([]*v1.WAFLog, error) {
	f, err := os.Open(logfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)

	res := []*v1.WAFLog{}
	for s.Scan() {
		// read line
		b := s.Bytes()
		var entry v1.WAFLog
		if err := json.Unmarshal(b, &entry); err != nil {
			continue
		}
		res = append(res, &entry)
	}
	return res, nil
}

func (s *dikastesSrcDstInfoTestSuite) waitForEndpoints(dikastes *server.Dikastes, expectedIPs map[string]bool) {
	s.Eventually(func() bool {
		endpoints := dikastes.GetWorkloadEndpoints()
		for _, ep := range endpoints {
			if len(ep.Ipv4Nets) > 0 {
				if _, ok := expectedIPs[ep.Ipv4Nets[0]]; ok {
					expectedIPs[ep.Ipv4Nets[0]] = true
				}
			}
		}
		for _, found := range expectedIPs {
			if !found {
				return false
			}
		}
		return true
	}, time.Duration(time.Second*5), time.Millisecond*1000)
}

func TestSrcDstInfo(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	suite.Run(t, new(dikastesSrcDstInfoTestSuite))
}
