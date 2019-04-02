// Copyright 2019 Tigera Inc. All rights reserved.

package elasticipsets

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/olivere/elastic"
	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/mock"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

func TestController_Add_Success(t *testing.T) {
	g := NewWithT(t)
	dbm := &mock.IPSet{}
	tkr := mockNewTicker()
	defer tkr.restoreNewTicker()
	uut := NewController(dbm)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)

	name := "test"
	set := db.IPSetSpec{"1.2.3.4"}
	fail := func() { t.Error("controller called fail func unexpectedly") }
	stat := &mock.Statser{}
	uut.Add(ctx, name, set, fail, stat)

	uut.StartReconciliation(ctx)

	tkr.reconcile(t, ctx)

	g.Eventually(dbm.Calls).Should(ContainElement(mock.Call{Method: "PutIPSet", Name: name, Set: set}))
	g.Expect(countMethod(dbm, "PutIPSet")()).To(Equal(1))

	dbm.Metas = append(dbm.Metas, db.IPSetMeta{Name: name})

	tkr.reconcile(t, ctx)

	g.Consistently(countMethod(dbm, "PutIPSet")).
		Should(Equal(1), "should not add a second time")
}

func TestController_Delete_Success(t *testing.T) {
	g := NewWithT(t)
	name := "testdelete"
	dbm := &mock.IPSet{Metas: []db.IPSetMeta{{Name: name}}}
	tkr := mockNewTicker()
	defer tkr.restoreNewTicker()
	uut := NewController(dbm)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)

	uut.Delete(ctx, name)
	uut.StartReconciliation(ctx)
	// Test idempotency
	uut.Delete(ctx, name)
	uut.StartReconciliation(ctx)

	tkr.reconcile(t, ctx)

	g.Eventually(dbm.Calls).Should(ContainElement(mock.Call{Method: "DeleteIPSet", Name: name}))
	g.Expect(countMethod(dbm, "DeleteIPSet")()).To(Equal(1))

	dbm.Metas = nil

	tkr.reconcile(t, ctx)

	g.Consistently(countMethod(dbm, "DeleteIPSet")).
		Should(Equal(1), "should not delete a second time")
}

func TestController_GC_Success(t *testing.T) {
	g := NewWithT(t)
	dbm := &mock.IPSet{}
	tkr := mockNewTicker()
	defer tkr.restoreNewTicker()
	uut := NewController(dbm)

	gcName := "shouldGC"
	noGCName := "shouldNotGC"
	var gcVer int64 = 6
	dbm.Metas = append(dbm.Metas, db.IPSetMeta{Name: gcName, Version: &gcVer}, db.IPSetMeta{Name: noGCName})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)
	uut.NoGC(ctx, noGCName)
	uut.StartReconciliation(ctx)

	tkr.reconcile(t, ctx)

	g.Eventually(dbm.Calls).Should(ContainElement(mock.Call{Method: "DeleteIPSet", Name: gcName, Version: &gcVer}))
	g.Expect(countMethod(dbm, "DeleteIPSet")()).To(Equal(1), "should only GC one set")
}

func TestController_Update_Success(t *testing.T) {
	g := NewWithT(t)
	name := "test"
	var version int64 = 10
	dbm := &mock.IPSet{Metas: []db.IPSetMeta{{Name: name, Version: &version}}}
	tkr := mockNewTicker()
	defer tkr.restoreNewTicker()
	uut := NewController(dbm)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)

	set := db.IPSetSpec{"1.2.3.4"}
	fail := func() { t.Error("controller called fail func unexpectedly") }
	stat := &mock.Statser{}
	uut.Add(ctx, name, set, fail, stat)

	uut.StartReconciliation(ctx)

	tkr.reconcile(t, ctx)

	g.Eventually(dbm.Calls).Should(ContainElement(mock.Call{Method: "PutIPSet", Name: name, Set: set}))
	g.Expect(countMethod(dbm, "PutIPSet")()).To(Equal(1))

	tkr.reconcile(t, ctx)

	g.Consistently(countMethod(dbm, "PutIPSet")).
		Should(Equal(1), "should not update a second time")
}

func TestController_Reconcile_FailToList(t *testing.T) {
	g := NewWithT(t)
	dbm := &mock.IPSet{Error: errors.New("test")}
	tkr := mockNewTicker()
	defer tkr.restoreNewTicker()
	uut := NewController(dbm)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)

	aName := "added"
	aSet := db.IPSetSpec{"6.7.8.9"}
	var failed bool
	fail := func() { failed = true }
	stat := &mock.Statser{}
	uut.Add(ctx, aName, aSet, fail, stat)

	gName := "nogc"
	uut.NoGC(ctx, gName)

	uut.StartReconciliation(ctx)

	tkr.reconcile(t, ctx)

	g.Eventually(func() []statser.ErrorCondition { return stat.Status().ErrorConditions }).Should(HaveLen(1))
	g.Expect(failed).To(BeFalse())
}

func TestController_Add_FailToPut(t *testing.T) {
	g := NewWithT(t)
	dbm := &mock.IPSet{PutError: errors.New("test")}
	tkr := mockNewTicker()
	defer tkr.restoreNewTicker()
	uut := NewController(dbm)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)

	name := "test"
	set := db.IPSetSpec{"1.2.3.4"}
	var failed bool
	fail := func() { failed = true }
	stat := &mock.Statser{}
	uut.Add(ctx, name, set, fail, stat)

	uut.StartReconciliation(ctx)

	tkr.reconcile(t, ctx)

	g.Eventually(dbm.Calls).Should(ContainElement(mock.Call{Method: "PutIPSet", Name: name, Set: set}))
	g.Expect(countMethod(dbm, "PutIPSet")()).To(Equal(1))
	g.Expect(stat.Status().ErrorConditions).To(HaveLen(1))
	g.Expect(stat.Status().ErrorConditions[0].Type).To(Equal(statser.ElasticSyncFailed))
	g.Expect(failed).To(BeTrue())

	dbm.PutError = nil
	tkr.reconcile(t, ctx)

	g.Eventually(countMethod(dbm, "PutIPSet")).
		Should(Equal(2), "should retry put")
	g.Expect(stat.Status().ErrorConditions).To(HaveLen(0), "should clear error on success")
}

func TestController_GC_NotFound(t *testing.T) {
	g := NewWithT(t)
	dbm := &mock.IPSet{DeleteError: &elastic.Error{Status: http.StatusNotFound}}
	tkr := mockNewTicker()
	defer tkr.restoreNewTicker()
	uut := NewController(dbm)

	gcName := "shouldGC"
	var gcVer int64 = 6
	dbm.Metas = append(dbm.Metas, db.IPSetMeta{Name: gcName, Version: &gcVer})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)
	uut.StartReconciliation(ctx)

	tkr.reconcile(t, ctx)

	g.Eventually(dbm.Calls).Should(ContainElement(mock.Call{Method: "DeleteIPSet", Name: gcName, Version: &gcVer}))
	g.Expect(countMethod(dbm, "DeleteIPSet")()).To(Equal(1))

	dbm.Metas = nil
	tkr.reconcile(t, ctx)
	g.Consistently(countMethod(dbm, "DeleteIPSet")).
		Should(Equal(1), "should not retry delete")
}

func TestController_GC_Error(t *testing.T) {
	g := NewWithT(t)
	dbm := &mock.IPSet{DeleteError: errors.New("test")}
	tkr := mockNewTicker()
	defer tkr.restoreNewTicker()
	uut := NewController(dbm)

	gcName := "shouldGC"
	var gcVer int64 = 6
	dbm.Metas = append(dbm.Metas, db.IPSetMeta{Name: gcName, Version: &gcVer})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)
	uut.StartReconciliation(ctx)

	tkr.reconcile(t, ctx)

	g.Eventually(dbm.Calls).Should(ContainElement(mock.Call{Method: "DeleteIPSet", Name: gcName, Version: &gcVer}))
	g.Expect(countMethod(dbm, "DeleteIPSet")()).To(Equal(1))

	dbm.DeleteError = nil
	tkr.reconcile(t, ctx)
	g.Eventually(countMethod(dbm, "DeleteIPSet")).
		Should(Equal(2), "should retry delete")
}

func TestController_NewTicker(t *testing.T) {
	dbm := &mock.IPSet{}
	uut := NewController(dbm)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uut.Run(ctx)
	uut.StartReconciliation(ctx)

	// Second call ensures we exercise the "real" ticker code
	uut.StartReconciliation(ctx)
}

// Test Add, Delete, NoGC, StartReconciliation, and Run functions when their
// context expires.
func TestController_ContextExpiry(t *testing.T) {
	g := NewWithT(t)
	dbm := &mock.IPSet{}
	uut := NewController(dbm)

	// monkey patch a blocking update channel. This prevents Add, Delete, NoGC
	// and StartReconciliation from being queued when the controller is not
	// running.
	uut.(*controller).updates = make(chan update)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	aCtx, aCancel := context.WithCancel(ctx)
	var aDone bool
	go func() {
		uut.Add(aCtx, "add", db.IPSetSpec{}, func() {}, &mock.Statser{})
		aDone = true
	}()

	dCtx, dCancel := context.WithCancel(ctx)
	var dDone bool
	go func() {
		uut.Delete(dCtx, "delete")
		dDone = true
	}()

	gCtx, gCancel := context.WithCancel(ctx)
	var gDone bool
	go func() {
		uut.NoGC(gCtx, "nogc")
		gDone = true
	}()

	sCtx, sCancel := context.WithCancel(ctx)
	var sDone bool
	go func() {
		uut.StartReconciliation(sCtx)
		sDone = true
	}()

	aCancel()
	dCancel()
	gCancel()
	sCancel()

	g.Eventually(func() bool { return aDone }).Should(BeTrue())
	g.Eventually(func() bool { return dDone }).Should(BeTrue())
	g.Eventually(func() bool { return gDone }).Should(BeTrue())
	g.Eventually(func() bool { return sDone }).Should(BeTrue())

	// Fresh controller to test Run context cancel
	uut2 := NewController(dbm)
	rCtx, rCancel := context.WithCancel(ctx)
	uut2.Run(rCtx)
	rCancel()
}

type mockTicker struct {
	oldTicker func() *time.Ticker
	ticks     chan<- time.Time
}

func mockNewTicker() *mockTicker {
	ticks := make(chan time.Time)
	mt := &mockTicker{oldTicker: NewTicker, ticks: ticks}
	tkr := time.Ticker{C: ticks}
	NewTicker = func() *time.Ticker { return &tkr }
	return mt
}

func (m *mockTicker) restoreNewTicker() {
	NewTicker = m.oldTicker
}

func (m *mockTicker) reconcile(t *testing.T, ctx context.Context) {
	select {
	case <-ctx.Done():
		t.Error("reconcile hangs")
	case m.ticks <- time.Now():
	}
}

func countMethod(client *mock.IPSet, method string) func() int {
	return func() int {
		n := 0
		for _, c := range client.Calls() {
			if c.Method == method {
				n++
			}
		}
		return n
	}
}
