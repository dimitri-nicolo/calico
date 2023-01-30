package testutils

import (
	"context"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/sirupsen/logrus"
)

func RefreshIndex(ctx context.Context, c lmaelastic.Client, index string) error {
	logrus.WithField("index", index).Info("[TEST] Refreshing index")
	_, err := c.Backend().Refresh(index).Do(ctx)
	return err
}
