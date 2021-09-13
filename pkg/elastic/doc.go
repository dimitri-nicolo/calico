// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// This package creates an ElasticSearch client, indexes a given document with the given document ID.
// If there is a connection or authorization error while sending a document, it retries to send after an interval.
package elastic
