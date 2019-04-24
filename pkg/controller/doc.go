// Copyright (c) 2019 Tigera, Inc. All rights reserved.
/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain reportRetriever copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package controller contains the controller for Tigera compliance. It is based (almost entirely) off the
// v1.14.1 Kubernetes CronJob controller, but modified with the following tweaks:
// -  Reads Calico GlobalReport resources instead of K8s CronJob resources.
// -  Automatically deletes successful jobs
// -  Passes the correct job start and finish time to the report job
// -  Updates the GlobalReport status instead of the CronJob status
// -  Triggers the jobs at reportRetriever configured interval after the start of the next job (since data is historical and we need
//    to allow the data to be fully sent to the archive.
package controller
