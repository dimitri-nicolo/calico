-- Copyright (c) 2024 Tigera, Inc. All rights reserved.

function record_transformer(tag, timestamp, record)
    if tag == "flows" then
        local end_time = record["end_time"]
        if end_time then
            record["@timestamp"] = end_time * 1000
        end
    end
    return 1, timestamp, record
end
