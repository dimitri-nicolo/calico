package replayer

import (
	"encoding/json"

	elastic "github.com/olivere/elastic/v7"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/projectcalico/calico/lma/pkg/list"
)

func unmarshalSearch(str string) (*elastic.SearchResult, error) {
	doc := new(elastic.SearchResult)
	if err := json.Unmarshal([]byte(str), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// extractAuditEvent converts the search result hits into an audit event array.
func extractAuditEvents(str string) ([]*auditv1.Event, error) {
	doc, err := unmarshalSearch(str)
	if err != nil {
		return nil, err
	}
	events := make([]*auditv1.Event, len(doc.Hits.Hits))
	for i, hit := range doc.Hits.Hits {
		event := new(auditv1.Event)
		if err = json.Unmarshal(hit.Source, event); err != nil {
			return nil, err
		}
		events[i] = event
	}
	return events, nil
}

// extractLists converts the search result hits into a list.
func extractLists(str string) ([]*list.TimestampedResourceList, error) {
	doc, err := unmarshalSearch(str)
	if err != nil {
		return nil, err
	}
	lists := make([]*list.TimestampedResourceList, len(doc.Hits.Hits))
	for i, hit := range doc.Hits.Hits {
		list := new(list.TimestampedResourceList)
		if err = json.Unmarshal(hit.Source, list); err != nil {
			return nil, err
		}
		lists[i] = list
	}
	return lists, nil
}

func GetEEAuditEventsDoc() (*elastic.SearchResult, error) {
	return unmarshalSearch(eeAuditEventsJSONString)
}

func GetKubeAuditEventsDoc() (*elastic.SearchResult, error) {
	return unmarshalSearch(kubeAuditEventsJSONString)
}

func GetListsDoc() (*elastic.SearchResult, error) {
	return unmarshalSearch(listsJSONString)
}

func GetEEAuditEvents() ([]*auditv1.Event, error) {
	return extractAuditEvents(eeAuditEventsJSONString)
}

func GetKubeAuditEvents() ([]*auditv1.Event, error) {
	return extractAuditEvents(kubeAuditEventsJSONString)
}

func GetLists() ([]*list.TimestampedResourceList, error) {
	return extractLists(listsJSONString)
}
