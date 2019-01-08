// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package testutils

import (
	"context"
	"os"
	"strings"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"
)

const K8S_TEST_NS = "test"
const TEST_DEFAULT_NS = "default"

// Delete everything under /calico from etcd.
func WipeEtcd() {
	be, err := backend.NewClient(apiconfig.CalicoAPIConfig{
		Spec: apiconfig.CalicoAPIConfigSpec{
			DatastoreType: apiconfig.EtcdV3,
			EtcdConfig: apiconfig.EtcdConfig{
				EtcdEndpoints: os.Getenv("ETCD_ENDPOINTS"),
			},
		},
	})
	if err != nil {
		panic(err)
	}
	_ = be.Clean()

	// Set the ready flag so calls to the CNI plugin can proceed
	calicoClient, _ := client.NewFromEnv()
	newClusterInfo := api.NewClusterInformation()
	newClusterInfo.Name = "default"
	datastoreReady := true
	newClusterInfo.Spec.DatastoreReady = &datastoreReady
	ci, err := calicoClient.ClusterInformation().Create(context.Background(), newClusterInfo, options.SetOptions{})
	if err != nil {
		panic(err)
	}
	log.Printf("Set ClusterInformation: %v %v\n", ci, *ci.Spec.DatastoreReady)
}

func MustDeleteIPPool(c client.Interface, cidr string) {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)

	name := strings.Replace(cidr, ".", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, "/", "-", -1)

	_, err := c.IPPools().Delete(context.Background(), name, options.DeleteOptions{})
	if err != nil {
		panic(err)
	}
}

// MustCreateNewIPPool creates a new Calico IPAM IP Pool.
func MustCreateNewIPPool(c client.Interface, cidr string, ipip, natOutgoing, ipam bool) string {
	return MustCreateNewIPPoolBlockSize(c, cidr, ipip, natOutgoing, ipam, 0)
}

// MustCreateNewIPPoolBlockSize creates a new Calico IPAM IP Pool with support for setting the block size.
func MustCreateNewIPPoolBlockSize(c client.Interface, cidr string, ipip, natOutgoing, ipam bool, blockSize int) string {
	log.SetLevel(log.DebugLevel)

	log.SetOutput(os.Stderr)

	name := strings.Replace(cidr, ".", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, "/", "-", -1)
	var mode api.IPIPMode
	if ipip {
		mode = api.IPIPModeAlways
	} else {
		mode = api.IPIPModeNever
	}

	pool := api.NewIPPool()
	pool.Name = name
	pool.Spec.CIDR = cidr
	pool.Spec.NATOutgoing = natOutgoing
	pool.Spec.Disabled = !ipam
	pool.Spec.IPIPMode = mode
	pool.Spec.BlockSize = blockSize

	_, err := c.IPPools().Create(context.Background(), pool, options.SetOptions{})
	if err != nil {
		panic(err)
	}
	return pool.Name
}

// Used for passing arguments to the CNI plugin.
type cniArgs struct {
	Env []string
}

func (c *cniArgs) AsEnv() []string {
	return c.Env
}
