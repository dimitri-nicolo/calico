// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package slack

var SlackErrors = map[string]bool{
	"invalid_payload":                   true,
	"user_not_found":                    true,
	"channel_not_found":                 true,
	"channel_is_archived":               true,
	"action_prohibited":                 true,
	"posting_to_general_channel_denied": true,
	"too_many_attachments":              true,
	"no_service":                        true,
	"no_service_id":                     true,
	"no_team":                           true,
	"team_disabled":                     true,
	"invalid_token":                     true,
}
