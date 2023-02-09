// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import _ "embed"

// flowLogsLinux is a sample flow logs to be ingested for testing purposes
//
//go:embed testdata/backend/flow_logs_legacy_linux.txt
var flowLogsLinux string
