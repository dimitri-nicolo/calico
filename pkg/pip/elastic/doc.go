package elastic

// This package contains an extension to the compliance ES client that can be used to handle composite aggregation
// queries.

// response.json contains a sample JSON response that the UI might receive.
//
// Two new fields added:
// - flow_impacted:  bool, indicates if flow is impacted by policy change. This is added to before and after flows.
// - source_action: string. For destination flows, this indicates which source flow it should be linked to. If not
//                  specified, the default is "allow" (which is the original pre-PIP behavior).
//
// JSON contains the following data:
//
// Flow A ---> B
// Before                         After
// Allow at A, Allow at B         Allow at A, Allow at B
// Allow at A, Deny at B          Allow at A, Deny at B
//                                Allow at A, Unknown at B
//                                Unknown at A, Deny at B
//                                Unknown at A, Unknown at B
//                                Deny at A, *no flow at B*
//
// Any set of flows handled by PIP will always add the flow_impacted and source_action fields. However, the UI should
// be resilient to both fields not being present since these fields will *not* be returned when doing a proxied ES
// query that is not processed by PIP (which is the case for standard flow visualizer).
