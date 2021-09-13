// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// There is one dpiProcessor per DeepPacketInspection resource, for each interface matching the DPI selector
// it starts a snort process, updates the status of DeepPacketInspection resource and tracks the snort process.
//
// If snort fails, it update the status of DeepPacketInspection and restarts snort after interval.
package processor
