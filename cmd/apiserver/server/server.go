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

package server

import (
	"flag"
	"io"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/recognizer"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/client-go/pkg/api"
	"k8s.io/kubernetes/pkg/util/interrupt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/tigera/calico-k8sapiserver/legacy"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
)

const defaultEtcdPathPrefix = "/calico/v1"

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartCalicoServer(out io.Writer) (*cobra.Command, error) {
	//	o := NewCalicoServerOptions(out, errOut)

	// Create the command that runs the API server
	cmd := &cobra.Command{
		Short: "run a calico api server",
	}
	// We pass flags object to sub option structs to have them configure
	// themselves. Each options adds its own command line flags
	// in addition to the flags that are defined above.
	flags := cmd.Flags()
	flags.AddGoFlagSet(flag.CommandLine)

	stopCh := make(chan struct{})
	jsonSerializer := json.NewSerializer(json.DefaultMetaFactory, api.Scheme, api.Scheme, false)
	codec := api.Codecs.CodecForVersions(jsonSerializer, recognizer.NewDecoder(legacy.NewDecoder()),
		schema.GroupVersions(schema.GroupVersions{calico.SchemeGroupVersion}), nil)
	ro := genericoptions.NewRecommendedOptions(defaultEtcdPathPrefix, api.Scheme, codec)
	ro.Etcd.StorageConfig.Type = storagebackend.StorageTypeETCD2
	opts := &CalicoServerOptions{
		RecommendedOptions: ro,
		StopCh:             stopCh,
	}
	opts.addFlags(flags)

	cmd.Run = func(c *cobra.Command, args []string) {
		h := interrupt.New(nil, func() {
			close(stopCh)
		})
		if err := h.Run(func() error { return RunServer(opts) }); err != nil {
			glog.Fatalf("error running server (%s)", err)
			return
		}
	}

	return cmd, nil
}
