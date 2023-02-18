package pip

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/audit"

	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/list"
)

func NewLister(client client.Client, cluster string) api.ListDestination {
	return &linseedLister{
		client:  client,
		cluster: cluster,
	}
}

type linseedLister struct {
	client  client.Client
	cluster string
}

func (l *linseedLister) RetrieveList(kind metav1.TypeMeta, from *time.Time, to *time.Time, sortAscendingTime bool) (*list.TimestampedResourceList, error) {
	// TODO: Implement this once tigera_secure_ee_snapshots are implemented in Linseed
	return nil, fmt.Errorf("RetrieveList not implemented in Linseed yet")
}

func (l *linseedLister) StoreList(meta metav1.TypeMeta, lst *list.TimestampedResourceList) error {
	// TODO: Implement this once tigera_secure_ee_snapshots are implemented in Linseed
	return fmt.Errorf("StoreList not implemented in Linseed yet")
}

func NewEventer(client client.Client, cluster string) api.ReportEventFetcher {
	return &linseedEventer{
		client:  client,
		cluster: cluster,
	}
}

type linseedEventer struct {
	client  client.Client
	cluster string
}

func (l *linseedEventer) GetAuditEvents(ctx context.Context, from *time.Time, to *time.Time) <-chan *api.AuditEventResult {
	// Result channel.
	ch := make(chan *api.AuditEventResult)

	go func() {
		defer func() {
			close(ch)
		}()

		// Get Audit logs, paginated.
		params := lapi.AuditLogParams{}
		params.TimeRange = &lmav1.TimeRange{}
		if from != nil {
			params.TimeRange.From = *from
		}
		if to != nil {
			params.TimeRange.From = *to
		}

		pager := client.NewListPager[audit.Event](&params)
		pages, errors := pager.Stream(ctx, l.client.AuditLogs(l.cluster).List)
		for page := range pages {
			for _, audit := range page.Items {
				cp := audit
				ch <- &api.AuditEventResult{Event: &cp}
			}
		}

		// Check for errors before returning.
		err := <-errors
		if err != nil {
			ch <- &api.AuditEventResult{Err: err}
		}
	}()

	return ch
}
