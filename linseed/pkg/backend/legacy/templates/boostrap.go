// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/projectcalico/go-json/json"
	"github.com/sirupsen/logrus"
)

// IndexBootstrapper creates an index template for the give log type and cluster information
// pairing and create a bootstrap index that uses that template
var IndexBootstrapper Load = func(ctx context.Context, client *elastic.Client, config *TemplateConfig) (*Template, error) {
	templateName := config.TemplateName()
	template, err := config.Build()
	if err != nil {
		return nil, err
	}

	// Create/Update the template in Elastic
	logrus.WithField("name", templateName).Info("Creating index template")
	_, err = client.IndexPutTemplate(templateName).BodyJson(template).Do(ctx)
	if err != nil {
		return nil, err
	}

	// Check if the alias already exists
	logrus.WithField("name", config.Alias()).Debug("Checking if alias exists")
	response, err := client.CatAliases().Alias(config.Alias()).Do(ctx)
	if err != nil {
		return nil, err
	}

	var exists bool
	for _, row := range response {
		if row.Alias == config.Alias() {
			exists = true
			break
		}
	}

	if !exists {
		logrus.Infof("Creating bootstrap index %s", config.BootstrapIndexName())
		aliasJson := fmt.Sprintf(`{"%s": {"is_write_index": true}}`, config.Alias())

		// Create the bootstrap index and mark it to be used for writes
		response, err := client.
			CreateIndex(config.BootstrapIndexName()).
			BodyJson(map[string]interface{}{"aliases": json.RawMessage(aliasJson)}).
			Do(ctx)
		if err != nil {
			return nil, err
		}
		if !response.Acknowledged {
			return nil, fmt.Errorf("failed to acknowledge index creation")
		}
	}
	return template, nil
}
