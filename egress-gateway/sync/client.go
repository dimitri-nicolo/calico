// Copyright 2021-2022 Tigera Inc. All rights reserved.

// Package sync abstracts out the process of (re)connecting and pulling data
// from a gRPC socket connected to felix, using channels. Consumers of the data
// can then simply pull it from a pipeline rather than worrying about the network
package sync

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"

	"github.com/projectcalico/calico/felix/proto"
)

const reconnectSleepTime = 3 * time.Second

// Connection encapsulates the client's connection functionality. In this instance, its primary use
// is to aid in facilitating testing, by making mocking easier.
type Connection interface {
	Dial() error
	Close() error
	Sync() (proto.PolicySync_SyncClient, error)
}

type connection struct {
	ctx        context.Context
	conn       *grpc.ClientConn
	dialOpts   []grpc.DialOption
	grpcTarget string
}

func init() {
	resolver.SetDefaultScheme("passthrough")
}

// Dial establishes a new client connection and returns its address. Closes the connection if an
// error occurs during the dial process. Logs the connection's address.
func (c *connection) Dial() error {
	var err error
	c.conn, err = grpc.NewClient(c.grpcTarget, c.dialOpts...)
	if err != nil {
		return err
	}
	log.Tracef("open connection: %p", c.conn)

	return nil
}

// Close closes the client connection. Logs the closing connection's address.
func (c *connection) Close() error {
	if c.conn != nil {
		log.Tracef("close connection: %p", c.conn)

		return c.conn.Close()
	}

	return nil
}

// Sync creates a new egress gateway proto client from a client connection and syncs a request.
// Subscribes to 'l3-routes' type payloads.
func (c *connection) Sync() (proto.PolicySync_SyncClient, error) {
	client := proto.NewPolicySyncClient(c.conn)
	if client == nil {
		return nil, errors.New("could not get new policy sync client")
	}

	return client.Sync(c.ctx, &proto.SyncRequest{
		SubscriptionType:         "l3-routes",
		SupportsIPv6RouteUpdates: false,
	})
}

type Client struct {
	updates     chan *proto.ToDataplane
	updatesLock sync.Mutex

	// client connection methods
	conn Connection

	// readiness reporting
	healthy bool
}

// NewClient builds a policy-sync client
func NewClient(ctx context.Context, grpcTarget string, dOps []grpc.DialOption) *Client {
	c := Client{
		conn: &connection{
			ctx:        ctx,
			grpcTarget: grpcTarget, // hostname:port to target
			dialOpts:   dOps,       // dial options for gRPC connections
		},
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
			connectAndSync(ctx, c)
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

// connectAndSync encapsulates connection and streaming mechanism.
func connectAndSync(ctx context.Context, c *Client) {
	// prepare connection.
	err := c.conn.Dial()
	defer func() {
		err := c.conn.Close()
		if err != nil {
			log.WithError(err).Info("Ignoring error while closing connection; will reconnect shortly.")
		}
	}()

	if err != nil {
		log.WithError(err).Error("could not dial syncserver")
		c.healthy = false
		return
	}

	// init a client and request data
	stream, err := c.conn.Sync()
	if err != nil {
		log.WithError(err).Error("could not initiate sync")
		c.healthy = false
		return
	}

	// stream in updates - this stream should stay open and continually feed new updates
	for {
		select {
		case <-ctx.Done():
			return
		default:
			update, err := stream.Recv()
			if err != nil {
				log.WithError(err).Warnf("Connection to calico-node failed (perhaps calico-node is being restarted?).")
				c.healthy = false
				// if the stream to Felix breaks, close the associated updates channel and restart the
				// connection process
				c.refreshUpdatesPipeline()
				return
			}
			c.updates <- update
			c.healthy = true
		}
	}
}
