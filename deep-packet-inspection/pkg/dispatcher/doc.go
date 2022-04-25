// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// This package handles the updates to resources that dpisyncer watches.
// The syncer package receives and caches updates on resource change from either typha or local syncer client,
// the cached request is sent to the dispatcher for processing.

// The dispatcher handles changes to WorkLoadEndpoint and DeepPacketInspection resource,
// If WorkLoadEndpoint resource is
// 	- created/updated:
//     	- if interface has changed, update the cached WEP name to interface mapping
//    	- update the cached labels, if the new label is no longer part of selector in DPI resource, stop the corresponding
//		  snort process, similarly if the new label is part of selector in DPI resource, start a snort process.
// 	- deleted:
//		- delete the cached labels for that WEP and stop any snort processes on the WEP.
// If DeepPacketInspection resource is
//	- created/updated:
//		- update the cached selectors, start snort process on all matching WEP and stop snort process for WEP
//	      that no longer match the updated selector.
//	- deleted:
//		- delete the cached selectors and stop snort processes started for this DPI resource.
package dispatcher
