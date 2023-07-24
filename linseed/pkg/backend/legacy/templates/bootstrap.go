// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

	"github.com/olivere/elastic/v7"
	"github.com/projectcalico/go-json/json"
	"github.com/sirupsen/logrus"
)

type IndexInfo struct {
	AliasExists  bool
	IndexExists  bool
	AliasedIndex string
	Mappings     map[string]interface{}
}

func (index IndexInfo) HasMappings(mappings map[string]interface{}) bool {
	// Please note that we only compare the mappings.
	// One could argue that similar logic should be done to detect settings changes.
	// This is possible, but we would need to ignore the following fields: provided_name, creation_date, uuid, version.
	// To keep things simple, we'll ignore this and assume that we're unlikely to update the settings without updating the mappings...
	return reflect.DeepEqual(index.Mappings, mappings)
}

// IndexBootstrapper creates an index template for the give log type and cluster information
// pairing and create a bootstrap index that uses that template
var IndexBootstrapper Load = func(ctx context.Context, client *elastic.Client, config *TemplateConfig) (*Template, error) {

	// Get some info about the index in ES
	indexInfo, err := GetIndexInfo(ctx, client, config)
	if err != nil {
		return nil, err
	}

	// Get template for the index
	templateName := config.TemplateName()
	template, err := config.Template()
	if err != nil {
		return nil, err
	}

	// Check if the index mappings are up to date.
	if indexInfo.HasMappings(template.Mappings) {
		logrus.Info("Existing index already uses the latest mappings")
	} else {
		// Create/Update the index template in Elastic
		logrus.WithField("name", templateName).Info("Creating index template")
		_, err = client.IndexPutTemplate(templateName).BodyJson(template).Do(ctx)
		if err != nil {
			return nil, err
		}

		if indexInfo.AliasExists {
			// Rollover index to get latest mappings
			err = RolloverIndex(ctx, client, config, indexInfo.AliasedIndex)
			if err != nil {
				return nil, err
			}
		} else {
			if !indexInfo.IndexExists {
				// Create index and alias
				err = CreateIndexAndAlias(ctx, client, config)
				if err != nil {
					return nil, err
				}
			} else {
				// Alias doesn't exist, but the index does
				err = CreateAliasForIndex(ctx, client, config)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return template, nil
}

func GetIndexInfo(ctx context.Context, client *elastic.Client, config *TemplateConfig) (index IndexInfo, err error) {
	// Check if the alias already exists
	logrus.WithField("name", config.Alias()).Debug("Checking if alias exists")
	response, err := client.CatAliases().Alias(config.Alias()).Do(ctx)
	if err != nil {
		return index, err
	}

	for _, row := range response {
		if row.Alias == config.Alias() && row.IsWriteIndex == "true" {
			index.AliasExists = true
			index.AliasedIndex = row.Index
			break
		}
	}

	if index.AliasExists {
		// Alias exists. This means that the index was setup previously.
		logrus.Infof("alias %s exists for index %s", config.Alias(), index.AliasedIndex)

		ir, err := client.IndexGet(index.AliasedIndex).Do(ctx)
		if err != nil {
			return index, err
		}

		// Get mappings
		index.Mappings = ir[index.AliasedIndex].Mappings
		if index.Mappings == nil {
			return index, fmt.Errorf("failed to get index mappings")
		}

		// Deal with odd "dynamic" property
		err = updateMappingsDynamicProperty(index.Mappings)
		if err != nil {
			return index, err
		}
	} else {
		// Check if index exists even though it's not aliased
		index.IndexExists, err = client.IndexExists(config.BootstrapIndexName()).Do(ctx)
		if err != nil {
			return index, err
		}
	}

	return index, nil
}

// The "dynamic" property is an odd one. We typically use `"dynamic": false` in our mapping files
// and the docs suggest that's correct: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/dynamic-field-mapping.html
// However when reading the mappings from the index, we get `"dynamic": "false"`, probably because
// the "dynamic" property can accept multiple types, and is just serialized as a string for some reason...
func updateMappingsDynamicProperty(mappings map[string]interface{}) error {
	if mappings["dynamic"] != nil {
		if reflect.TypeOf(mappings["dynamic"]) == reflect.TypeOf(string("")) {
			s, ok := mappings["dynamic"].(string)
			if !ok {
				return fmt.Errorf("dynamic property in not a string (%v)", mappings["dynamic"])
			}

			if s == "false" {
				mappings["dynamic"] = false
			}
			if s == "true" {
				mappings["dynamic"] = true
			}
		}
	}
	return nil
}

func CreateIndexAndAlias(ctx context.Context, client *elastic.Client, config *TemplateConfig) error {
	logrus.WithField("name", config.BootstrapIndexName()).Infof("Creating bootstrap index")
	aliasJson := fmt.Sprintf(`{"%s": {"is_write_index": true}}`, config.Alias())

	// Create the bootstrap index and mark it to be used for writes
	response, err := client.
		CreateIndex(config.BootstrapIndexName()).
		BodyJson(map[string]interface{}{"aliases": json.RawMessage(aliasJson)}).
		Do(ctx)
	if err != nil {
		return err
	}
	if !response.Acknowledged {
		return fmt.Errorf("failed to acknowledge index creation")
	}
	logrus.WithField("name", response.Index).Info("Bootstrap index created")
	return nil
}

func CreateAliasForIndex(ctx context.Context, client *elastic.Client, config *TemplateConfig) error {
	logrus.WithField("name", config.BootstrapIndexName()).Infof("Creating alias for index")
	_, err := client.Alias().Add(config.BootstrapIndexName(), config.Alias()).Do(ctx)
	return err
}

func RolloverIndex(ctx context.Context, client *elastic.Client, config *TemplateConfig, oldIndex string) error {
	logrus.Info("Existing index does not use the latest mappings, let's rollover the index so that it uses the latest mappings")
	rolloverReq := client.RolloverIndex(config.Alias())
	// Event indices prior to 3.17 were created to match the pattern tigera_secure_ee_events.{$managed_cluster}.lma
	// or tigera_secure_ee_events.{$tenant_id}.{$managed_cluster}.lma. Because the index does
	// not have a suffix like `-000000` or `-0`, it will result in an error when trying to perform a roll-over request
	// We need to specify an index that ends in a number as a target-index on the Elastic API calls
	match, err := regexp.MatchString("^(tigera_secure_ee_events.).+(.lma)$", oldIndex)
	if err != nil {
		return err
	}
	if match {
		logrus.Infof("Existing index %s does not end in an number. Will need to specify a index that ends with a number", oldIndex)
		rolloverReq.NewIndex(config.BootstrapIndexName())
	}
	response, err := rolloverReq.Do(ctx)
	if err != nil {
		return err
	}
	if !response.Acknowledged {
		return fmt.Errorf("failed to acknowledge index rollover")
	}
	if response.RolledOver {
		logrus.Infof("Rolled over index %s to index %s", response.OldIndex, response.NewIndex)
	} else {
		logrus.Infof("Did not rollover index %s", response.OldIndex)
	}
	return nil
}
