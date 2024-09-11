// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package endpoint

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/projectcalico/calico/fluent-bit/plugins/out_linseed/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/resource"
)

type EndpointController struct {
	dynamicClient  dynamic.Interface
	dynamicFactory dynamicinformer.DynamicSharedInformerFactory

	mu       sync.RWMutex
	endpoint string
}

func NewController(cfg *config.Config) (*EndpointController, error) {
	// initialize kubernetes clients
	dynamicClient, err := dynamic.NewForConfig(cfg.RestConfig)
	if err != nil {
		return nil, err
	}

	return &EndpointController{
		dynamicClient:  dynamicClient,
		dynamicFactory: dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0),

		endpoint: cfg.Endpoint,
	}, nil
}

func (c *EndpointController) Run(stopCh <-chan struct{}) error {
	if c.endpoint == "" {
		logrus.Debug("empty endpoint from environment variable or plugin config. read cluster resource instead")

		endpoint, err := c.getAndWatchEndpoint()
		if err != nil {
			return err
		}
		c.endpoint = endpoint
		logrus.Infof("log ingestion endpoint is set to %q", c.endpoint)

		// Start initializes all requested informers. They are handled in goroutines
		// which run until the stop channel gets closed.
		c.dynamicFactory.Start(stopCh)
		c.dynamicFactory.WaitForCacheSync(stopCh)
		logrus.Debug("dynamic shared informer factory is started")
	}

	logrus.Info("linseed plugin endpoint controller is started")
	return nil
}

func (c *EndpointController) Endpoint() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.endpoint
}

func (c *EndpointController) getAndWatchEndpoint() (string, error) {
	gvr := schema.GroupVersionResource{
		Group:    "operator.tigera.io",
		Version:  "v1",
		Resource: "nonclusterhosts",
	}

	// get existing NonClusterHost resource
	nonclusterhost, err := c.dynamicClient.Resource(gvr).Namespace(corev1.NamespaceAll).Get(context.Background(), resource.DefaultTSEEInstanceName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	endpoint, err := extractEndpointFromSpec(nonclusterhost.Object["spec"])
	if err != nil {
		return "", err
	}

	// watch for NonClusterHost resource changes
	nonClusterHostInformer := c.dynamicFactory.ForResource(gvr).Informer()
	if _, err = nonClusterHostInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updateFunc,
	}); err != nil {
		return "", err
	}

	return endpoint, nil
}

func (c *EndpointController) updateFunc(oldObj, newObj interface{}) {
	logrus.Debug("receive nonclusterhost update event")

	unstructuredObj, ok := newObj.(*unstructured.Unstructured)
	if !ok {
		logrus.Warn("failed to cast new nonclusterhost object. skip update")
		return
	}

	endpoint, err := extractEndpointFromSpec(unstructuredObj.Object["spec"])
	if err != nil {
		logrus.WithError(err).Warn("failed to extract endpoint from the nonclusterhost object. skip update")
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.endpoint = endpoint
	logrus.Infof("log ingestion endpoint is changed to %q", c.endpoint)
}

func extractEndpointFromSpec(obj interface{}) (string, error) {
	spec, ok := obj.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("failed to cast spec object")
	}
	ep, ok := spec["endpoint"]
	if !ok {
		return "", fmt.Errorf("failed to get endpoint from spec")
	}
	endpoint, ok := ep.(string)
	if !ok {
		return "", fmt.Errorf("failed to cast endpoint to string")
	}
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return "", err
	}
	return endpoint, nil
}
