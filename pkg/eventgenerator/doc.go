// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Each snort writes alerts to a file named alert_fast.txt located in specific path,
// if the size of alert file reaches value set in its config, it rotates the `alert_fast.txt` and the old
// content are moved to a file named `alert_fast.txt.<unixtime>`.

// When a new DPI and WEP combination is added to EventGenerator, it reads all leftover files from previous run (if any),
// generates events, adds it to the ESForwarder's queue, and deletes the file.
// It tails the file `alert_fast.txt`, for each new line it transforms the alert,
// generates event document and unique document id, adds it to the ESForwarder's queue for processing.
package eventgenerator
