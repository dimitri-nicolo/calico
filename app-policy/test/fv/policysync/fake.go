package policysync

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/projectcalico/calico/felix/proto"
)

var _ proto.PolicySyncServer = (*FakePolicySync)(nil)

type FakePolicySync struct {
	server     *grpc.Server
	lis        net.Listener
	sr         *proto.SyncRequest
	updates    chan *proto.ToDataplane
	stopped    bool
	disconnect chan struct{}
	activeCon  int

	endpoints map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint
	profiles  map[proto.ProfileID]*proto.Profile
	policies  map[proto.PolicyID]*proto.Policy
}

func NewFakePolicySync(listenPath string) (*FakePolicySync, error) {

	s := grpc.NewServer()
	lis, err := net.Listen("unix", listenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	fakePolicySync := &FakePolicySync{
		server:     s,
		lis:        lis,
		sr:         nil,
		updates:    make(chan *proto.ToDataplane, 1024),
		disconnect: make(chan struct{}, 1),
		activeCon:  0,

		endpoints: make(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint),
		profiles:  make(map[proto.ProfileID]*proto.Profile),
		policies:  make(map[proto.PolicyID]*proto.Policy),
	}

	proto.RegisterPolicySyncServer(s, fakePolicySync)
	return fakePolicySync, nil
}

func (p *FakePolicySync) Sync(sr *proto.SyncRequest, srv proto.PolicySync_SyncServer) error {
	if p.stopped {
		return grpc.ErrServerStopped
	}

	p.sr = sr
	p.activeCon++
T:
	for {
		select {
		case <-srv.Context().Done():
			break T
		case <-p.disconnect:
			p.activeCon--
			return context.Canceled // apparently, this replaces 'grpc conn is closing'
		case upd := <-p.updates:
			if err := srv.Send(upd); err != nil {
				log.Error("fakePolicySync.Sync send error: ", err)
			}
		}
	}
	return nil
}

func (p *FakePolicySync) Report(context.Context, *proto.DataplaneStats) (*proto.ReportResult, error) {
	return &proto.ReportResult{Successful: true}, nil
}

func (p *FakePolicySync) StopAndDisconnect() {
	p.stopped = true
	p.disconnect <- struct{}{}
}

func (p *FakePolicySync) Resume() {
	p.stopped = false
}

func (p *FakePolicySync) SendUpdates(updates ...*proto.ToDataplane) {
	for _, update := range updates {
		p.updates <- update
	}
}

func (p *FakePolicySync) Teardown() {
	p.server.Stop()
	log.Error(p.lis.Close())
}

func (p *FakePolicySync) Serve(ctx context.Context) {
	log.Infof("policy sync serving at %v", p.lis.Addr())
	go func() {
		log.Error(p.server.Serve(p.lis))
	}()
	<-ctx.Done()
	p.Teardown()
}

func (p *FakePolicySync) Addr() (_ string) {
	if p.lis != nil {
		return p.lis.Addr().String()
	}
	return
}

func (p *FakePolicySync) ActiveConnections() int {
	return p.activeCon
}
