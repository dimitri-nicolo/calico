// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/app-policy/waf"
)

type mockLogFormatter struct {
	entries []*logrus.Entry
}

func (f *mockLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	f.entries = append(f.entries, entry)
	return nil, nil
}

func TestLogRateLimiter(t *testing.T) {
	proxiedFormatter := new(mockLogFormatter)
	rateLimitedFormatter := waf.NewRateLimitedFormatter(
		proxiedFormatter,
		3,                    // 3 is the maximum number of log entries
		300*time.Millisecond, // per 300 milliseconds
	)

	for i := 0; i < 5; i++ { // generating 5 log entries:
		entry := logrus.NewEntry(nil)
		entry.Message = fmt.Sprintf("log entry [%d]", i)
		_, _ = rateLimitedFormatter.Format(entry)
	}

	time.Sleep(300 * time.Millisecond) // 300ms pause

	for i := 5; i < 10; i++ { // and generating another 5 log entries (10 in total):
		entry := logrus.NewEntry(nil)
		entry.Message = fmt.Sprintf("log entry [%d]", i)
		_, _ = rateLimitedFormatter.Format(entry)
	}

	expectedEntries := []string{
		"log entry [0]",
		"log entry [1]",
		"log entry [2]",
		// "log entry [3]", // these two entries should be discarded as
		// "log entry [4]", // the limit of 3 entries per 300ms was exceeded
		"log entry [5]",
		"log entry [6]",
		"log entry [7]",
		// "log entry [8]", // these two entries should also be discarded as
		// "log entry [9]", // the limit of 3 entries per 300ms was exceeded again
	}

	if len(expectedEntries) != len(proxiedFormatter.entries) {
		t.Fatalf(
			"number of log entries is incorrect - expected: %d, got: %d",
			len(expectedEntries),
			len(proxiedFormatter.entries),
		)
	}

	for i, entry := range proxiedFormatter.entries {
		expectedMessage := expectedEntries[i]
		if entry.Message != expectedMessage {
			t.Errorf(
				"log entries do not match - expected: %s, got: %s",
				expectedMessage,
				entry.Message,
			)
		}
	}
}
