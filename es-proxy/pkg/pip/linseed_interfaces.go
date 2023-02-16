package pip

import (
	"context"
	"time"

	"github.com/projectcalico/calico/lma/pkg/api"
	"github.com/projectcalico/calico/lma/pkg/list"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LinseedLister struct{}

func (l *LinseedLister) RetrieveList(kind metav1.TypeMeta, from *time.Time, to *time.Time, sortAscendingTime bool) (*list.TimestampedResourceList, error) {
	// TODO: Implement this
	return nil, nil
}

func (l *LinseedLister) StoreList(meta metav1.TypeMeta, lst *list.TimestampedResourceList) error {
	// TODO: Implement this
	return nil
}

type LinseedEventer struct{}

func (l *LinseedEventer) GetAuditEvents(ctx context.Context, from *time.Time, to *time.Time) <-chan *api.AuditEventResult {
	// TODO: Implement this
	ch := make(chan *api.AuditEventResult)
	close(ch)
	return ch
}
