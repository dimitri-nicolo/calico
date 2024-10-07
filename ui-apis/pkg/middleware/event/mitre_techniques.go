// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package event

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// MITRE data fetched from https://github.com/mitre-attack/attack-stix-data/blob/master/enterprise-attack/enterprise-attack-14.1.json
// then processed with the following command:
// jq '[.objects[] | select( .type == "attack-pattern" ) | {id: .id, name: .name, is_subtechnique: .x_mitre_is_subtechnique, data: .external_references[] | select( .external_id != null ) | select( .external_id | contains("T") )} | {mitre_id: .data.external_id, name: .name, url: .data.url, is_subtechnique: .is_subtechnique}]' enterprise-attack-14.1.json > mitre_techniques.json
//
//go:embed mitre_techniques.json
var mitreTechniquesJson []byte
var mitreTechniques []MitreTechnique

type MitreTechnique struct {
	MitreID        string `json:"mitre_id"`
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Url            string `json:"url"`
	IsSubtechnique bool   `json:"is_subtechnique"`
}

func init() {
	err := json.Unmarshal(mitreTechniquesJson, &mitreTechniques)
	if err != nil {
		panic(err)
	}
}

func GetMitreTechnique(mitreId string) (MitreTechnique, error) {
	mt := MitreTechnique{}
	err := fmt.Errorf("unknown MITRE technique with ID %s", mitreId)

	for _, mitreTechnique := range mitreTechniques {
		if mitreTechnique.MitreID == mitreId {
			mt = mitreTechnique
			err = nil
			break
		}
	}

	if mt.IsSubtechnique {
		parts := strings.Split(mitreId, ".")
		techniqueId := parts[0]
		technique, err := GetMitreTechnique(techniqueId)
		if err != nil {
			return mt, err
		}
		mt.Name = fmt.Sprintf("%s: %s", technique.Name, mt.Name)
	}

	mt.DisplayName = fmt.Sprintf("%s: %s", mitreId, mt.Name)

	return mt, err
}
