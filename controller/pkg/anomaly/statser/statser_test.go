// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

func TestStatser_SuccessfulSync(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-job"

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	st := NewStatser(name).(*statser)
	st.Run(ctx)
	defer st.Close()

	g.Expect(st.lastSuccessfulSync).Should(Equal(time.Time{}))
	st.SuccessfulSync()
	g.Expect(st.lastSuccessfulSync).Should(BeTemporally("~", time.Now(), time.Second))
	g.Expect(st.errorConditions).Should(HaveLen(0))
}

func TestStatser_Error_ClearError(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-job"
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	st := NewStatser(name).(*statser)
	st.Run(ctx)

	errStr1 := "test1"
	st.Error(XPackRecordsFailed, errors.New(errStr1))

	g.Expect(st.lastSuccessfulSync).Should(Equal(time.Time{}), "lastSuccessfulSync was not modified")
	g.Expect(st.lastSuccessfulSearch).Should(Equal(time.Time{}), "lastSuccessfulSearch was not modified")

	g.Expect(st.errorConditions).Should(HaveLen(1))
	g.Expect(st.errorConditions[XPackRecordsFailed]).Should(ConsistOf(v3.ErrorCondition{XPackRecordsFailed, errStr1}))

	errStr2 := "test2"
	st.Error(XPackRecordsFailed, errors.New(errStr2))

	g.Expect(st.lastSuccessfulSync).Should(Equal(time.Time{}), "lastSuccessfulSync was not modified")
	g.Expect(st.lastSuccessfulSearch).Should(Equal(time.Time{}), "lastSuccessfulSearch was not modified")

	g.Expect(st.errorConditions).Should(HaveLen(1))
	g.Expect(st.errorConditions[XPackRecordsFailed]).Should(ConsistOf(
		v3.ErrorCondition{XPackRecordsFailed, errStr1},
		v3.ErrorCondition{XPackRecordsFailed, errStr2},
	))

	errStr3 := "test3"
	st.Error(FilterFailed, errors.New(errStr3))

	g.Expect(st.errorConditions).Should(HaveLen(2))
	g.Expect(st.errorConditions[XPackRecordsFailed]).Should(ConsistOf(
		v3.ErrorCondition{XPackRecordsFailed, errStr1},
		v3.ErrorCondition{XPackRecordsFailed, errStr2},
	))
	g.Expect(st.errorConditions[FilterFailed]).Should(ConsistOf(
		v3.ErrorCondition{FilterFailed, errStr3},
	))

	st.ClearError(XPackRecordsFailed)
	g.Expect(st.errorConditions).Should(HaveLen(1))
	g.Expect(st.errorConditions[FilterFailed]).Should(ConsistOf(
		v3.ErrorCondition{FilterFailed, errStr3},
	))

	st.ClearError(FilterFailed)
	g.Expect(st.errorConditions).Should(HaveLen(0))
}

func TestStatser_status_deadlock(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-job"

	st := NewStatser(name).(*statser)

	ch := make(chan struct{})

	go func() {
		st.lock.Lock()
		defer st.lock.Unlock()
		_ = st.status()
		close(ch)
	}()

	g.Eventually(ch).Should(BeClosed(), "status does not deadlock")
}
