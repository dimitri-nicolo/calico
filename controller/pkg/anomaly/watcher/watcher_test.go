// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestReady(t *testing.T) {
	g := NewWithT(t)

	w := NewWatcher(&db.MockEvents{}, &db.MockAuditLog{}, &elastic.MockXPack{}).(*watcher)
	g.Expect(w.Ready()).Should(BeFalse())

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()
	w.Run(ctx)
	g.Eventually(w.Ready).Should(BeTrue())

	var jid string
	for jid, _ = range Jobs {
		break
	}
	g.Expect(jid).ShouldNot(Equal(""))

	w.jobWatchers[jid].statser.Error("test", errors.New("test"))
	g.Expect(w.Ready()).Should(BeFalse())
}
