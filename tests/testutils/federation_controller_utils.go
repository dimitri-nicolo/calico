// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package testutils

import (
	"fmt"
	"os"

	"github.com/projectcalico/felix/fv/containers"
)

func RunFederationController(etcdIP string, lkconfig string, rkconfig []string, isCalicoEtcdDatastore bool) *containers.Container {
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
		"-e", "RECONCILER_PERIOD=10s",
		"-e", fmt.Sprintf("KUBECONFIG=%s", lkconfig),
		"-v", fmt.Sprintf("%s:%s", lkconfig, lkconfig),
	}...)

	for _, kcf := range rkconfig {
		args = append(args, []string{
			"-v", fmt.Sprintf("%s:%s", kcf, kcf),
		}...)
	}

	args = append(args, fmt.Sprintf("%s", os.Getenv("CONTAINER_NAME")))

	return containers.Run("calico-federation-controller",
		containers.RunOpts{AutoRemove: true},
		args...,
	)
}
