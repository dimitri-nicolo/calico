package v1

import (
	"encoding/json"
	"sort"
)

type NamespacedName struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

func (n NamespacedName) String() string {
	return n.Namespace + "/" + n.Name
}

// NamespacedNames is the set of namespaced names. This is JSON marshaled as a slice., but stored as a map to avoid
// duplication.
type NamespacedNames map[NamespacedName]struct{}

func (ns NamespacedNames) MarshalJSON() ([]byte, error) {
	names := ns.AsSortedSlice()
	return json.Marshal(names)
}

func (ns *NamespacedNames) UnmarshalJSON(b []byte) error {
	var names []NamespacedName
	err := json.Unmarshal(b, &names)
	if err != nil {
		return err
	}

	*ns = make(map[NamespacedName]struct{})
	for _, name := range names {
		(*ns)[name] = struct{}{}
	}
	return nil
}

func (ns NamespacedNames) AsSortedSlice() []NamespacedName {
	if len(ns) == 0 {
		return nil
	}
	var names SortableNamespacedNames
	for name := range ns {
		names = append(names, name)
	}
	sort.Sort(names)
	return names
}

// SortableServices is used to sort a set of services.
type SortableNamespacedNames []NamespacedName

func (s SortableNamespacedNames) Len() int {
	return len(s)
}
func (s SortableNamespacedNames) Less(i, j int) bool {
	if s[i].Namespace < s[j].Namespace {
		return true
	} else if s[i].Namespace == s[j].Namespace && s[i].Name < s[j].Name {
		return true
	}
	return false
}
func (s SortableNamespacedNames) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
