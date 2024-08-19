package client

import (
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/auth"
)

type QueryLabelsReq struct {
	api.ResourceType
	auth.Permission
}

type QueryLabelsResp struct {
	ResourceTypeLabelMap map[api.ResourceType][]LabelKeyValuePair `json:"resourceTypeLabelMap"`
}

type LabelKeyValuePair struct {
	LabelKey    string   `json:"labelKey"`
	LabelValues []string `json:"labelValues"`
}
