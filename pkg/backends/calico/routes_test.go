package calico

import (
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func addEndpointSubset(ep *v1.Endpoints, nodename string) {
	ep.Subsets = append(ep.Subsets, v1.EndpointSubset{
		Addresses: []v1.EndpointAddress{
			v1.EndpointAddress{
				NodeName: &nodename}}})
}

func buildSimpleExternalService() (svc *v1.Service, ep *v1.Endpoints) {
	meta := metav1.ObjectMeta{Namespace: "foo", Name: "bar"}
	svc = &v1.Service{
		ObjectMeta: meta,
		Spec: v1.ServiceSpec{
			Type:                  v1.ServiceTypeNodePort,
			ClusterIP:             "127.0.0.1",
			ExternalTrafficPolicy: v1.ServiceExternalTrafficPolicyTypeLocal,
		}}
	ep = &v1.Endpoints{
		ObjectMeta: meta,
	}
	return
}

func buildSimpleInternalService(isAnnotated bool) (svc *v1.Service, ep *v1.Endpoints) {
	meta := metav1.ObjectMeta{
		Namespace:   "foo-int",
		Name:        "bar-int",
		Annotations: make(map[string]string),
	}
	svc = &v1.Service{
		ObjectMeta: meta,
		Spec: v1.ServiceSpec{
			Type:      v1.ServiceTypeClusterIP,
			ClusterIP: "127.0.0.2",
		}}

	if isAnnotated {
		svc.Annotations[advertiseClusterIPAnnotation] = "true"
	}
	ep = &v1.Endpoints{
		ObjectMeta: meta,
	}
	return
}

var _ = Describe("RouteGenerator", func() {
	var rg *routeGenerator
	BeforeEach(func() {
		rg = &routeGenerator{
			nodeName:    "foobar",
			svcIndexer:  cache.NewIndexer(cache.MetaNamespaceKeyFunc, nil),
			epIndexer:   cache.NewIndexer(cache.MetaNamespaceKeyFunc, nil),
			svcRouteMap: make(map[string]string),
			client: &client{
				cache:  make(map[string]string),
				synced: true,
			},
		}
		rg.client.watcherCond = sync.NewCond(&rg.client.cacheLock)
	})
	Describe("getServiceForEndpoints", func() {
		It("should get corresponding service for endpoints", func() {
			svc, ep := buildSimpleExternalService()
			err := rg.svcIndexer.Add(svc)
			Expect(err).ToNot(HaveOccurred())
			fetchedSvc, key := rg.getServiceForEndpoints(ep)
			Expect(fetchedSvc.ObjectMeta).To(Equal(svc.ObjectMeta))
			Expect(key).To(Equal("foo/bar"))
		})
	})
	Describe("getEndpointsForService", func() {
		It("should get corresponding endpoints for service", func() {
			svc, ep := buildSimpleExternalService()
			err := rg.epIndexer.Add(ep)
			Expect(err).ToNot(HaveOccurred())
			fetchedEp, key := rg.getEndpointsForService(svc)
			Expect(fetchedEp.ObjectMeta).To(Equal(ep.ObjectMeta))
			Expect(key).To(Equal("foo/bar"))
		})
	})

	Describe("(un)setRouteForSvc", func() {
		Context("svc = svc, ep = nil", func() {
			It("should set and unset routes for an external service with local traffic policy", func() {
				svc, ep := buildSimpleExternalService()
				addEndpointSubset(ep, rg.nodeName)

				err := rg.epIndexer.Add(ep)
				Expect(err).ToNot(HaveOccurred())
				rg.setRouteForSvc(svc, nil)
				Expect(rg.svcRouteMap["foo/bar"]).To(Equal("127.0.0.1/32"))
				rg.unsetRouteForSvc(ep)
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
			})
			It("should not set routes for an external service with cluster traffic policy", func() {
				svc, ep := buildSimpleExternalService()
				svc.Spec.ExternalTrafficPolicy = v1.ServiceExternalTrafficPolicyTypeCluster
				addEndpointSubset(ep, rg.nodeName)

				err := rg.epIndexer.Add(ep)
				Expect(err).ToNot(HaveOccurred())
				rg.setRouteForSvc(svc, nil)
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
				rg.unsetRouteForSvc(ep)
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
			})
		})
		Context("svc = nil, ep = ep", func() {
			It("should set and unset routes for an external service with local traffic policy", func() {
				svc, ep := buildSimpleExternalService()
				addEndpointSubset(ep, rg.nodeName)

				err := rg.svcIndexer.Add(svc)
				Expect(err).ToNot(HaveOccurred())

				rg.setRouteForSvc(nil, ep)
				Expect(rg.svcRouteMap["foo/bar"]).To(Equal("127.0.0.1/32"))
				rg.unsetRouteForSvc(ep)
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
			})
			It("should not set routes for an external service with cluster traffic policy", func() {
				svc, ep := buildSimpleExternalService()
				svc.Spec.ExternalTrafficPolicy = v1.ServiceExternalTrafficPolicyTypeCluster
				addEndpointSubset(ep, rg.nodeName)

				err := rg.svcIndexer.Add(svc)
				Expect(err).ToNot(HaveOccurred())

				rg.setRouteForSvc(nil, ep)
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
				rg.unsetRouteForSvc(ep)
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
			})
			It("should not set routes for an internal service without the special annotation", func() {
				svc, ep := buildSimpleInternalService(false)
				addEndpointSubset(ep, rg.nodeName)

				err := rg.svcIndexer.Add(svc)
				Expect(err).ToNot(HaveOccurred())

				rg.setRouteForSvc(nil, ep)
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
				rg.unsetRouteForSvc(ep)
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
			})
			It("should set routes for an internal service with the special annotation", func() {
				svc, ep := buildSimpleInternalService(true)
				addEndpointSubset(ep, rg.nodeName)

				err := rg.svcIndexer.Add(svc)
				Expect(err).ToNot(HaveOccurred())

				rg.setRouteForSvc(nil, ep)
				Expect(rg.svcRouteMap["foo-int/bar-int"]).To(Equal("127.0.0.2/32"))
				rg.unsetRouteForSvc(ep)
				Expect(rg.svcRouteMap["foo-int/bar-int"]).To(BeEmpty())
			})
		})
	})

	Describe("resourceInformerHandlers", func() {
		var (
			svc    *v1.Service
			ep     *v1.Endpoints
			svcInt *v1.Service
			epInt  *v1.Endpoints
		)

		BeforeEach(func() {
			svc, ep = buildSimpleExternalService()
			svcInt, epInt = buildSimpleInternalService(true)

			addEndpointSubset(ep, rg.nodeName)
			addEndpointSubset(epInt, rg.nodeName)
			err := rg.epIndexer.Add(ep)
			Expect(err).ToNot(HaveOccurred())
			err = rg.epIndexer.Add(epInt)
			Expect(err).ToNot(HaveOccurred())

			err = rg.svcIndexer.Add(svc)
			Expect(err).ToNot(HaveOccurred())
			err = rg.svcIndexer.Add(svcInt)
		})

		It("should remove clusterIPs when endpoints are deleted", func() {
			// Trigger a service add - it should update the cache with its route.
			initRevision := rg.client.cacheRevision
			rg.onSvcAdd(svc)
			Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
			Expect(rg.svcRouteMap["foo/bar"]).To(Equal("127.0.0.1/32"))
			Expect(rg.client.cache["/calico/staticroutes/127.0.0.1-32"]).To(Equal("127.0.0.1/32"))

			// Simulate the remove of the local endpoint. It should withdraw the route.
			ep.Subsets = []v1.EndpointSubset{}
			err := rg.epIndexer.Add(ep)
			Expect(err).ToNot(HaveOccurred())

			rg.onEPAdd(ep)
			Expect(rg.client.cacheRevision).To(Equal(initRevision + 2))
			Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
			Expect(rg.client.cache["/calico/staticroutes/127.0.0.1-32"]).To(BeEmpty())
			Expect(rg.client.cache).To(Equal(map[string]string{}))
		})

		Context("onSvc[Add|Delete]", func() {
			It("should add the external service's clusterIP into the svcRouteMap", func() {
				// add
				initRevision := rg.client.cacheRevision
				rg.onSvcAdd(svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap["foo/bar"]).To(Equal("127.0.0.1/32"))
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.1-32"]).To(Equal("127.0.0.1/32"))

				// delete
				rg.onSvcDelete(svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 2))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo/bar"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.1-32"))
			})
			It("should not add the external service's clusterIP into the svcRouteMap (traffic policy = Cluster)", func() {
				// add
				initRevision := rg.client.cacheRevision
				svc.Spec.ExternalTrafficPolicy = v1.ServiceExternalTrafficPolicyTypeCluster
				rg.onSvcAdd(svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision))
				Expect(rg.svcRouteMap["foo/bar"]).To(BeEmpty())
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.1-32"]).To(BeEmpty())

				// delete
				rg.onSvcDelete(svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo/bar"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.1-32"))
			})
			It("should add the internal service's clusterIP into the svcRouteMap", func() {
				// add
				initRevision := rg.client.cacheRevision

				rg.onSvcAdd(svcInt)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap["foo-int/bar-int"]).To(Equal("127.0.0.2/32"))
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.2-32"]).To(Equal("127.0.0.2/32"))

				// delete
				rg.onSvcDelete(svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo/bar"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.1-32"))
			})
		})

		Context("onSvcUpdate", func() {
			It("should add the service's clusterIP into the svcRouteMap and then remove it for unsupported service type", func() {
				initRevision := rg.client.cacheRevision
				rg.onSvcUpdate(nil, svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap["foo/bar"]).To(Equal("127.0.0.1/32"))
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.1-32"]).To(Equal("127.0.0.1/32"))

				// set to unsupported service type
				svc.Spec.Type = v1.ServiceTypeExternalName
				rg.onSvcUpdate(nil, svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 2))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo/bar"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.1-32"))
			})
			It("should add the internal service's clusterIP into the svcRouteMap and then remove it for missing annotation", func() {
				svc, ep = buildSimpleInternalService(true)

				addEndpointSubset(ep, rg.nodeName)
				err := rg.epIndexer.Add(ep)
				Expect(err).ToNot(HaveOccurred())

				err = rg.svcIndexer.Add(svc)
				Expect(err).ToNot(HaveOccurred())

				initRevision := rg.client.cacheRevision
				rg.onSvcUpdate(nil, svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap["foo-int/bar-int"]).To(Equal("127.0.0.2/32"))
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.2-32"]).To(Equal("127.0.0.2/32"))

				// delete the special annotation
				delete(svc.Annotations, advertiseClusterIPAnnotation)
				rg.onSvcUpdate(nil, svc)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 2))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo-int/bar-int"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.2-32"))
			})
		})

		Context("onEp[Add|Delete]", func() {
			It("should add the service's clusterIP into the svcRouteMap", func() {
				// add
				initRevision := rg.client.cacheRevision
				rg.onEPAdd(ep)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap["foo/bar"]).To(Equal("127.0.0.1/32"))
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.1-32"]).To(Equal("127.0.0.1/32"))

				// delete
				rg.onEPDelete(ep)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 2))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo/bar"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.1-32"))
			})
			It("should add the internal service's clusterIP into the svcRouteMap", func() {
				// add
				initRevision := rg.client.cacheRevision
				rg.onEPAdd(epInt)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap["foo-int/bar-int"]).To(Equal("127.0.0.2/32"))
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.2-32"]).To(Equal("127.0.0.2/32"))

				// delete
				rg.onEPDelete(epInt)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 2))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo-int/bar-int"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.2-32"))
			})
		})

		Context("onEpDelete", func() {
			It("should add the service's clusterIP into the svcRouteMap and then remove it for unsupported service type", func() {
				initRevision := rg.client.cacheRevision
				rg.onEPUpdate(nil, ep)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap["foo/bar"]).To(Equal("127.0.0.1/32"))
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.1-32"]).To(Equal("127.0.0.1/32"))

				// set to unsupported service type
				svc.Spec.Type = v1.ServiceTypeExternalName
				rg.onEPUpdate(nil, ep)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 2))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo/bar"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.1-32"))
			})
			It("should add the internal service's clusterIP into the svcRouteMap and then remove it for missing annotation", func() {
				initRevision := rg.client.cacheRevision
				rg.onEPUpdate(nil, epInt)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 1))
				Expect(rg.svcRouteMap["foo-int/bar-int"]).To(Equal("127.0.0.2/32"))
				Expect(rg.client.cache["/calico/staticroutes/127.0.0.2-32"]).To(Equal("127.0.0.2/32"))

				// delete the special annotation
				delete(svcInt.Annotations, advertiseClusterIPAnnotation)
				rg.onEPUpdate(nil, epInt)
				Expect(rg.client.cacheRevision).To(Equal(initRevision + 2))
				Expect(rg.svcRouteMap).ToNot(HaveKey("foo-int/bar-int"))
				Expect(rg.client.cache).ToNot(HaveKey("/calico/staticroutes/127.0.0.2-32"))
			})
		})
	})
})
