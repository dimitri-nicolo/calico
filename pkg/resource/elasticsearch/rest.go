// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

// esk8srest provides an implementation of the rest.RESTClient that can handle k8s requests to elasticsearch.k8s.elastic.co
package elasticsearch

import (
	esalpha1 "github.com/elastic/cloud-on-k8s/pkg/apis/elasticsearch/v1alpha1"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// RESTClient is a wrapper for the rest.RESTClient that can handle requests to the elasticsearch.k8s.elastic.co API group
type RESTClient interface {
	rest.Interface
	CalculateTigeraElasticsearchHash() (string, error)
}

type restClient struct {
	*rest.RESTClient
}

// NewRESTClient creates a new instance of the RESTClient from the given rest.Config
func NewRESTClient(config *rest.Config) (RESTClient, error) {
	if err := esalpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}

	config.APIPath = "/apis"
	config.GroupVersion = &schema.GroupVersion{Group: "elasticsearch.k8s.elastic.co", Version: "v1alpha1"}

	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	restCli, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &restClient{RESTClient: restCli}, nil
}

// CalculateTigeraElasticsearchHash calculates and returns a hash that can be used to determine if the tigera elasticsearch
// cluster has changed
func (r *restClient) CalculateTigeraElasticsearchHash() (string, error) {
	es := &esalpha1.Elasticsearch{}
	err := r.Get().Resource("elasticsearches").Namespace(resource.TigeraElasticsearchNamespace).Name(resource.DefaultTSEEInstanceName).Do().Into(es)
	if err != nil {
		return "", err
	}

	return resource.CreateHashFromObject(es.CreationTimestamp)
}
