package v1

import (
	"encoding/json"
	"fmt"
	"sort"
)

type ServicePort struct {
	NamespacedName `json:",inline"`
	Protocol       string `json:"protocol,omitempty"`
	PortName       string `json:"port_name,omitempty"`
	Port           int    `json:"port,omitempty"`
}

func (s ServicePort) String() string {
	return fmt.Sprintf("ServicePort(%s/%s:%d %s %s)", s.Namespace, s.Name, s.Port, s.PortName, s.Protocol)
}

// ServicePorts is the set of service ports. This is JSON marshaled as a slice, but stored as a map to avoid
// duplication.
type ServicePorts map[ServicePort]struct{}

func (ns ServicePorts) MarshalJSON() ([]byte, error) {
	names := ns.AsSortedSlice()
	return json.Marshal(names)
}

func (ns *ServicePorts) UnmarshalJSON(b []byte) error {
	var names []ServicePort
	err := json.Unmarshal(b, &names)
	if err != nil {
		return err
	}

	*ns = make(map[ServicePort]struct{})
	for _, name := range names {
		(*ns)[name] = struct{}{}
	}
	return nil
}

func (ns ServicePorts) AsSortedSlice() []ServicePort {
	if len(ns) == 0 {
		return nil
	}
	var names SortableServicePorts
	for name := range ns {
		names = append(names, name)
	}
	sort.Sort(names)
	return names
}

// SortableServicePorts is used to sort a set of services.
type SortableServicePorts []ServicePort

func (s SortableServicePorts) Len() int {
	return len(s)
}
func (s SortableServicePorts) Less(i, j int) bool {
	if s[i].Namespace < s[j].Namespace {
		return true
	} else if s[i].Namespace > s[j].Namespace {
		return false
	}

	if s[i].Name < s[j].Name {
		return true
	} else if s[i].Name > s[j].Name {
		return false
	}

	if s[i].PortName < s[j].PortName {
		return true
	} else if s[i].PortName > s[j].PortName {
		return false
	}

	if s[i].Protocol < s[j].Protocol {
		return true
	} else if s[i].Protocol > s[j].Protocol {
		return false
	}

	if s[i].Port < s[j].Port {
		return true
	} else if s[i].Port > s[j].Port {
		return false
	}

	return false
}

func (s SortableServicePorts) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
