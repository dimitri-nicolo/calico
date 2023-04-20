// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package testutils

import (
	"context"
	"fmt"
	"math/rand"
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

func RandomClusterName() string {
	name := fmt.Sprintf("cluster-%s", RandStringRunes(8))
	logrus.WithField("name", name).Info("Using random cluster name for test")
	return name
}

func RandomTenantName() string {
	name := fmt.Sprintf("tenant-%s", RandStringRunes(8))
	logrus.WithField("name", name).Info("Using random tenant name for test")
	return name
}

func RandStringRunes(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
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

func CleanupIndices(ctx context.Context, client *elastic.Client, cluster string) error {
	indices, err := client.CatIndices().Do(ctx)
	if err != nil {
		return err
	}
	for _, idx := range indices {
		if !strings.Contains(idx.Index, cluster) {
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
		if !strings.Contains(a.Alias, cluster) {
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

	templates, err := client.IndexGetTemplate().Do(ctx)
	if err != nil {
		return err
	}

	for name := range templates {
		if strings.Contains(name, cluster) {
			_, err = client.IndexDeleteTemplate(name).Do(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
