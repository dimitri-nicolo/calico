package fv_test

import (
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/projectcalico/calico/app-policy/server"
	fakepolicysync "github.com/projectcalico/calico/app-policy/test/fv/policysync"
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
		server.WithWAFConfig(true, "", []string{}, []string{
			"Include @coraza.conf-recommended",
			"Include @crs-setup.conf.example",
			"Include @owasp_crs/*.conf",
			"SecRuleEngine DetectionOnly",
		}),
	)

	return policySync, dikastes, dikastesLogfile, nil
}

func (s *dikastesSrcDstInfoTestSuite) TearDownTest() {
}

func TestSrcDstInfo(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	suite.Run(t, new(dikastesSrcDstInfoTestSuite))
}
