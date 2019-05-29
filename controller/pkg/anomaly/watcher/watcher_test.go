// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestPing(t *testing.T) {
	g := NewWithT(t)

	w := NewWatcher(&db.MockEvents{}, &elastic.MockXPack{})
	g.Expect(w.Ready()).Should(BeFalse())

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()
	w.Run(ctx)
	g.Eventually(w.Ready).Should(BeTrue())
}
