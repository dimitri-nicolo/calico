// Copyright (c) 2019 Tigera Inc. All rights reserved.

package statser

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"

	"github.com/tigera/intrusion-detection/controller/pkg/calico"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestStatser_status_deadlock(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-alert"
	xpack := &elastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	ga := &calico.MockGlobalAlertInterface{
		GlobalAlert: &v3.GlobalAlert{},
	}

	st := NewStatser(name, xpack, ga).(*statser)

	ch := make(chan struct{})

	go func() {
		st.lock.Lock()
		defer st.lock.Unlock()
		_ = st.status()
		close(ch)
	}()

	g.Eventually(ch).Should(BeClosed(), "status does not deadlock")
}

func TestStatser_updateStatus(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-alert"
	xpack := &elastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	ga := &calico.MockGlobalAlertInterface{
		GlobalAlert: &v3.GlobalAlert{},
	}

	st := NewStatser(name, xpack, ga).(*statser)

	st.errorConditions.Add(ElasticSyncFailed, errors.New("test error"))

	st.updateStatus(st.status())

	g.Expect(ga.GlobalAlert.Status.ErrorConditions).Should(ConsistOf(st.errorConditions.TypedErrors(ElasticSyncFailed)))
}
