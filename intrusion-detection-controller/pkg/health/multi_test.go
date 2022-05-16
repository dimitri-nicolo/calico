// Copyright 2019 Tigera Inc. All rights reserved.

package health

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

func TestPingers(t *testing.T) {
	g := NewWithT(t)

	ctx := context.TODO()
	g.Expect(Pingers{}.Ping(ctx)).ShouldNot(HaveOccurred())
	g.Expect(Pingers{MockPinger{}}.Ping(ctx)).ShouldNot(HaveOccurred())
	g.Expect(Pingers{MockPinger{}, MockPinger{}}.Ping(ctx)).ShouldNot(HaveOccurred())
	g.Expect(Pingers{MockPinger{errors.New("error")}}.Ping(ctx)).Should(HaveOccurred())
	g.Expect(Pingers{MockPinger{}, MockPinger{errors.New("error")}}.Ping(ctx)).Should(HaveOccurred())
	g.Expect(Pingers{MockPinger{errors.New("error")}, MockPinger{}}.Ping(ctx)).Should(HaveOccurred())
}

func TestReadiers(t *testing.T) {
	g := NewWithT(t)

	g.Expect(Readiers{}.Ready()).Should(BeTrue())
	g.Expect(Readiers{MockReadier{}}.Ready()).Should(BeFalse())
	g.Expect(Readiers{MockReadier{}, MockReadier{}}.Ready()).Should(BeFalse())
	g.Expect(Readiers{MockReadier{}, MockReadier{true}}.Ready()).Should(BeFalse())
	g.Expect(Readiers{MockReadier{true}, MockReadier{}}.Ready()).Should(BeFalse())
	g.Expect(Readiers{MockReadier{true}}.Ready()).Should(BeTrue())
	g.Expect(Readiers{MockReadier{true}, MockReadier{true}}.Ready()).Should(BeTrue())
}
