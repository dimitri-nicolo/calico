/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"

	"github.com/golang/glog"
	"github.com/tigera/calico-k8sapiserver/cmd/apiserver/server"
	// set up logging the k8s way
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/kubernetes/pkg/api"
	// The API groups for our API must be installed before we can use the
	// client to work with them.  This needs to be done once per process; this
	// is the point at which we handle this for the API server process.
	// Please do not remove.
	_ "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/install"
)

func init() {
	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(api.Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	kapiv1.Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd, err := server.NewCommandStartCalicoServer(os.Stdout)
	if err != nil {
		glog.Errorf("Error creating server: %v", err)
		logs.FlushLogs()
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		glog.Errorf("server exited unexpectedly (%s)", err)
		logs.FlushLogs()
		os.Exit(1)
	}
}
