package calico

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
)

func testCheckEventType(t *testing.T, expectEventType watch.EventType, w watch.Interface) {
	select {
	case res := <-w.ResultChan():
		if res.Type != expectEventType {
			t.Errorf("event type want=%v, get=%v", expectEventType, res.Type)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("time out after waiting %v on ResultChan", wait.ForeverTestTimeout)
	}
}
