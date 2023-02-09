package testutils

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	elastic "github.com/olivere/elastic/v7"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func RefreshIndex(ctx context.Context, c lmaelastic.Client, index string) error {
	logrus.WithField("index", index).Info("[TEST] Refreshing index")
	_, err := c.Backend().Refresh(index).Do(ctx)
	return err
}

func LogIndicies(ctx context.Context, client *elastic.Client) error {
	indices, err := client.CatIndices().Do(ctx)
	if err != nil {
		return err
	}
	for _, idx := range indices {
		logrus.Infof("Index exists: %s", idx.Index)
	}
	aliases, err := client.CatAliases().Do(ctx)
	if err != nil {
		return err
	}
	for _, a := range aliases {
		logrus.Infof("Alias exists: %s -> %s", a.Alias, a.Index)
	}
	return nil
}

func CleanupIndices(ctx context.Context, client *elastic.Client, prefix string) error {
	indices, err := client.CatIndices().Do(ctx)
	if err != nil {
		return err
	}
	for _, idx := range indices {
		if !strings.HasPrefix(idx.Index, prefix) {
			// Skip indicies that don't match.
			continue
		}
		_, err = client.DeleteIndex(idx.Index).Do(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "not_found") {
				continue
			}
			return fmt.Errorf("error deleting index: %s", err)
		}
	}
	aliases, err := client.CatAliases().Do(ctx)
	if err != nil {
		return err
	}
	for _, a := range aliases {
		if !strings.HasPrefix(a.Alias, prefix) {
			// Skip aliases that don't match.
			continue
		}
		_, err = client.Alias().Remove(a.Index, a.Alias).Do(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "not_found") {
				continue
			}
			return fmt.Errorf("error removing alias: %s", err)
		}
	}

	return nil
}
