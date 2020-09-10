package model

import "fmt"

type ViewMap []Mapping

type Mapping struct {
	TargetField string `yaml:"targetField"`
	Type        string `yaml:"type"`
	SourceField string `yaml:"sourceField"`
}

func (vm *ViewMap) append(m Mapping) {
	*vm = append(*vm, m)
}

func (vm *ViewMap) createNewFromFlatten(fromField, toField string) (*ViewMap, error) {
	vm2 := ViewMap{}
	foundField := false
	for _, m := range *vm {
		if m.TargetField == fromField && !foundField {
			vm2.append(Mapping{
				TargetField: toField,
				SourceField: toField,
				Type:        m.Type,
			})
			foundField = true
		} else if m.TargetField == fromField {
			return nil, fmt.Errorf("Cannot create field with name %s, already exists", toField)
		} else {
			vm2.append(m)
		}
	}
	if !foundField {
		return nil, fmt.Errorf("Field %s not found in viewmap", fromField)
	}

	return &vm2, nil
}
