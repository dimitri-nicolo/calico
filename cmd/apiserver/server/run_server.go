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

	"github.com/golang/glog"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

func RunServer(opts *CalicoServerOptions) error {
	path := "/tmp/ready"
	_ = os.Remove(path)

	if opts.StopCh == nil {
		/* the caller of RunServer should generate the stop channel
		if there is a need to stop the API server */
		opts.StopCh = make(chan struct{})
	}

	config, err := opts.Config()
	if err != nil {
		return err
	}

	glog.V(4).Infoln("Completing API server configuration")
	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	allStop := make(chan struct{})
	go func() {
		glog.Infoln("Starting watch extension")
		changed, err := WatchExtensionAuth(server.GenericAPIServer.LoopbackClientConfig, allStop)
		if err != nil {
			glog.Errorln("Unable to watch the extension auth ConfigMap: ", err)
		}
		if changed {
			glog.Infoln("Detected change in extension-apiserver-authentication ConfigMap, exiting so apiserver can be restarted")
		}
	}()

	go func() {
		// do we need to do any post api installation setup? We should have set up the api already?
		glog.Infoln("Running the API server")
		server.GenericAPIServer.AddPostStartHook("tigera-apiserver-autoregistration",
			func(context genericapiserver.PostStartHookContext) error {
				f, err := os.Create(path)
				if err != nil {
					glog.Errorln(err)
					return err
				}
				glog.Info("apiserver is ready.")
				f.Close()
				return nil
			})
		err = server.GenericAPIServer.PrepareRun().Run(allStop)
	}()

	select {
	case <-allStop:
	case <-opts.StopCh:
		close(allStop)
	}

	return err
}
