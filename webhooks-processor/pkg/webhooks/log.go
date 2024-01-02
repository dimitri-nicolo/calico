// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func logEntry(webhook *v3.SecurityEventWebhook) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{"name": webhook.Name, "uid": webhook.UID})
}
