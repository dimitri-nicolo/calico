// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package testutils

import (
	"fmt"
	"os"

	"github.com/projectcalico/felix/fv/containers"
)

func RunFederationController(etcdIP string, localKubeconfig string, remoteKubeconfigs []string, isCalicoEtcdDatastore bool) *containers.Container {
	args := []string{"--privileged"}
	if isCalicoEtcdDatastore {
		args = append(args, []string{
			"-e", "DATASTORE_TYPE=etcdv3",
			"-e", fmt.Sprintf("ETCD_ENDPOINTS=http://%s:2379", etcdIP),
		}...)
	} else {
		args = append(args, []string{
			"-e", "DATASTORE_TYPE=kubernetes",
		}...)
	}
	args = append(args, []string{
		"-e", "ENABLED_CONTROLLERS=federatedservices",
		"-e", "LOG_LEVEL=debug",
		"-e", "DO_NOT_INITIALIZE_CALICO=true",
		"-e", "COMPACTION_PERIOD=0s",
		"-e", "RECONCILER_PERIOD=10s",
		"-e", "DEBUG_USE_SHORT_POLL_INTERVALS=true",
		"-e", fmt.Sprintf("KUBECONFIG=%s", localKubeconfig),
		"-v", fmt.Sprintf("%s:%s", localKubeconfig, localKubeconfig),
	}...)

	for _, rkc := range remoteKubeconfigs {
		args = append(args, []string{
			"-v", fmt.Sprintf("%s:%s", rkc, rkc),
		}...)
	}

	args = append(args, os.Getenv("CONTAINER_NAME"))

	return containers.Run("calico-federation-controller",
		containers.RunOpts{AutoRemove: true},
		args...,
	)
}
