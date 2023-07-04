// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates

import (
	"context"
	"fmt"
	"reflect"

	"github.com/olivere/elastic/v7"
	"github.com/projectcalico/go-json/json"
	"github.com/sirupsen/logrus"
)

// IndexBootstrapper creates an index template for the give log type and cluster information
// pairing and create a bootstrap index that uses that template
var IndexBootstrapper Load = func(ctx context.Context, client *elastic.Client, config *TemplateConfig) (*Template, error) {
	templateName := config.TemplateName()
	template, err := config.Template()
	if err != nil {
		return nil, err
	}

	// Create/Update the template in Elastic
	logrus.WithField("template", template).Info("Template to be created...")
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

	var aliasExists bool
	var aliasedIndex string
	for _, row := range response {
		if row.Alias == config.Alias() {
			aliasExists = true
			aliasedIndex = row.Index
			break
		}
	}

	if !aliasExists {
		indexExists, err := client.IndexExists(config.BootstrapIndexName()).Do(ctx)
		if err != nil {
			return nil, err
		}
		if !indexExists {
			logrus.WithField("name", config.BootstrapIndexName()).Infof("Creating bootstrap index")
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
			logrus.WithField("name", response.Index).Info("Bootstrap index created")
		} else {
			// Alias doesn't exist, but the index does.
			logrus.WithField("name", config.BootstrapIndexName()).Infof("Creating alias for index")
			_, err := client.Alias().Add(config.BootstrapIndexName(), config.Alias()).Do(ctx)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Alias exists. This means that the index was setup previously.
		logrus.Infof("alias %s exists for index %s", config.Alias(), aliasedIndex)

		// We now want to compare its mappings to our mappings.
		// If different, we want to rollover the index so that it uses the latest mappings.

		ir, err := client.IndexGet(aliasedIndex).Do(ctx)
		if err != nil {
			return nil, err
		}

		indexMappings := ir[aliasedIndex].Mappings
		if indexMappings == nil {
			return nil, fmt.Errorf("failed to get index mappings")
		}

		// Please note that we only compare the mappings.
		// One could argue that similar logic should be done to detect settings changes.
		// This is possible, but we would need to ignore the following fields: provided_name, creation_date, uuid, version.
		// To keep things simple, we'll ignore this and assume that we're unlikely to update the settings without updating the mappings...
		if reflect.DeepEqual(indexMappings, template.Mappings) {
			logrus.Info("Existing index already uses the latest mappings")
		} else {
			logrus.Info("Existing index does not use the latest mappings, let's rollover the index so that it uses the latest mappings")
			response, err := client.RolloverIndex(config.Alias()).Do(ctx)
			if err != nil {
				return nil, err
			}
			if !response.Acknowledged {
				return nil, fmt.Errorf("failed to acknowledge index rollover")
			}
			if response.RolledOver {
				logrus.Infof("Rolled over index %s to index %s", response.OldIndex, response.NewIndex)
			} else {
				logrus.Infof("Did not rollover index %s", response.OldIndex)
			}
		}
	}

	return template, nil
}
