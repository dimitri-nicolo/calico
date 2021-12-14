// Copyright (c) 2018 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package federationsyncer

/*
federationsyncer implements an api.Syncer for use with the Federated Services Controller.

It consumes Calico RemoteClusterConfiguration, and also Kubernetes Service and Endpoints resources to
provide a global view of Services and Endpoints across clusters.

This implementation uses the watchersyncer and the remotecluster wrapper.
*/
