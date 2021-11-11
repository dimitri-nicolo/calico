// Package sync abstracts out the process of (re)connecting and pulling data
// from a gRPC socket connected to felix, using channels. Consumers of the data
// can then simply pull it from a pipeline rather than worrying about the network
package sync

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/egress-gateway/proto"
	"google.golang.org/grpc"
)

const reconnectSleepTime = 3 * time.Second

type Client struct {
	dialOpts   []grpc.DialOption
	grpcTarget string

	updates     chan *proto.ToDataplane
	updatesLock sync.Mutex

	// readiness reporting
	healthy bool
}

// NewClient builds a policy-sync client
func NewClient(grpcTarget string, dOps []grpc.DialOption) *Client {
	c := Client{
		grpcTarget: grpcTarget, // hostname:port to target
		dialOpts:   dOps,       // dial options for gRPC connections
	}
	c.refreshUpdatesPipeline()
	return &c
}

// GetUpdatesPipeline provides read access to the sync-client's updates pipeline
func (c *Client) GetUpdatesPipeline() <-chan *proto.ToDataplane {
	c.updatesLock.Lock()
	defer c.updatesLock.Unlock()

	return c.updates
}

// SyncForever continually re-connects to Felix sync server every {reconnectSleepTime} until ctx.Done, sending updates down client's updates channel
func (c *Client) SyncForever(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// prepare connection
			conn, err := grpc.Dial(c.grpcTarget, c.dialOpts...)
			if err != nil {
				log.Errorf("could not dial syncserver: %v", err)
				c.healthy = false
				break
			}

			// init a client and request data
			client := proto.NewPolicySyncClient(conn)
			stream, err := client.Sync(ctx, &proto.SyncRequest{
				SubscriptionType: "l3-routes",
			})
			if err != nil {
				log.Errorf("could not initiate sync: %v", err)
				c.healthy = false
				break
			}

			// stream in updates - this stream should stay open and continually feed new updates
			for {
				update, err := stream.Recv()
				if err != nil {
					log.Warnf("unexpected error during sync: %v", err)
					c.healthy = false
					// if the stream to Felix breaks, close the associated updates channel and restart the connection process
					c.refreshUpdatesPipeline()
					break
				}

				c.updates <- update
				c.healthy = true
			}

			conn.Close()
		}
		time.Sleep(reconnectSleepTime)
	}
}

// create a fresh channel to push updates over, and cleanup the old one if necessary
func (c *Client) refreshUpdatesPipeline() {
	// we replace the client's channel reference before closing the old one,
	// so that there is no chance of a downstream consumer fetching a closed channel
	updatesOld := c.updates

	c.updatesLock.Lock()
	c.updates = make(chan *proto.ToDataplane)
	if updatesOld != nil {
		close(updatesOld)
	}
	c.updatesLock.Unlock()
}
