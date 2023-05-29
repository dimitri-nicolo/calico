// Copyright 2019 Tigera Inc. All rights reserved.

package storage

// forwarderConfigMapping contains properties that are internal state used by the event forwarder.
const forwarderConfigMapping = `{
    "properties": {
        "last_successful_event_time": {
            "type": "date",
            "format": "strict_date_optional_time||epoch_second"
        },
        "last_successful_event_id": {
            "type": "keyword"
        },
        "last_successful_run_endtime": {
            "type": "date",
            "format": "strict_date_optional_time||epoch_second"
        }
    }
}`
