// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import _ "embed"

// from is the start interval to query flow logs
// ingested flow logs have 1674254798 as start time
const from = 1674254790

// from is the end interval to query flow logs
// ingested flow logs have 1674255110 as end time
const to = 1674255111

// flowID will set an ID when ingesting flows to make sure
// we are actually updating the same document and not creating
// new flow logs
const flowID = "1"

// flowLog is a sample flow log to be ingested for testing purposes
//
//go:embed testdata/backend/flow_log_legacy.json
var flowLog string

// flow is the expected response when querying flows
//
//go:embed testdata/output/flow.json
var flow string
