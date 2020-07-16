/*
Copyright 2017 The Kubernetes Authors.

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
	"os"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/klog"

	"github.com/tigera/apiserver/pkg/apiserver"
)

// PrepareServer prepares the server for execution. After invoking the caller should run RunServer.
func PrepareServer(opts *CalicoServerOptions) (*apiserver.ProjectCalicoServer, error) {
	if opts.StopCh == nil {
		/* the caller of RunServer should generate the stop channel
		if there is a need to stop the API server */
		opts.StopCh = make(chan struct{})
	}

	config, err := opts.Config()
	if err != nil {
		return nil, err
	}

	klog.V(4).Infoln("Completing API server configuration")
	return config.Complete().New()
}

// RunServer runs the Calico API server.  This blocks until stopped channel (passed in through options) is closed.
func RunServer(opts *CalicoServerOptions, server *apiserver.ProjectCalicoServer) error {
	path := "/tmp/ready"
	_ = os.Remove(path)

	allStop := make(chan struct{})
	go func() {
		klog.Infoln("Starting watch extension")
		changed, err := WatchExtensionAuth(allStop)
		if err != nil {
			klog.Errorln("Unable to watch the extension auth ConfigMap: ", err)
		}
		if changed {
			klog.Infoln("Detected change in extension-apiserver-authentication ConfigMap, exiting so apiserver can be restarted")
		}
	}()

	go func() {
		// do we need to do any post api installation setup? We should have set up the api already?
		klog.Infoln("Running the API server")
		server.GenericAPIServer.AddPostStartHook("tigera-apiserver-autoregistration",
			func(context genericapiserver.PostStartHookContext) error {
				f, err := os.Create(path)
				if err != nil {
					klog.Errorln(err)
					return err
				}
				klog.Info("apiserver is ready.")
				f.Close()
				return nil
			})
		if err := server.GenericAPIServer.PrepareRun().Run(allStop); err != nil {
			klog.Errorln("Error running API server: ", err)
		}
	}()

	select {
	case <-allStop:
	case <-opts.StopCh:
		close(allStop)
	}

	return nil
}
