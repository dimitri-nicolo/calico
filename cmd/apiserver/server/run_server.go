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
	"github.com/golang/glog"
)

func RunServer(opts *CalicoServerOptions) error {
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

	// do we need to do any post api installation setup? We should have set up the api already?
	glog.Infoln("Running the API server")
	return server.GenericAPIServer.PrepareRun().Run(opts.StopCh)

	return nil
}
