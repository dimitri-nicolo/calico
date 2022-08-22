// Copyright 2021-2022 Tigera Inc. All rights reserved.
package panorama

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	panw "github.com/PaloAltoNetworks/pango"
	"github.com/PaloAltoNetworks/pango/objs/addr"
	"github.com/PaloAltoNetworks/pango/objs/addrgrp"
	dvgrp "github.com/PaloAltoNetworks/pango/pnrm/dg"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"
	"github.com/projectcalico/calico/libcalico-go/lib/set"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	panutils "github.com/projectcalico/calico/firewall-integration/pkg/controllers/panorama/utils"
	"github.com/projectcalico/calico/firewall-integration/pkg/controllers/panorama/utils/mocks"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	expectedDataFolder = "./utils/data/expected/gns"
	inputDataFolder    = "./utils/data/input/"
)

// syncToDatastore
var _ = Describe("Tests controller syncToDatastore functionality", func() {
	var (
		dagc dynamicAddressGroupsController
		mccl *FakeDagCalicoClient
	)

	BeforeEach(func() {
		// Setup tags input for the dynamic address groups controller.
		tags := set.FromArray([]string{"tag1", "tag2", "tag3"})
		// Load the input data.
		var globalNetworkSetStartingState []v3.GlobalNetworkSet
		file := fmt.Sprintf("%s/%s", inputDataFolder, "globalNetworkSetStartingState.json")
		panutils.LoadData(file, &globalNetworkSetStartingState)
		// Define a mock calico client, with sample datastore input.
		mccl = NewFakeDagCalicoClient(globalNetworkSetStartingState)
		// Define a mock Panorama client.
		mpcl := &mocks.MockPanoramaClient{}
		// Create a cache to store GlobalNetworkSets in.
		mgns := make(map[string]interface{})
		listFunc := func() (map[string]interface{}, error) {
			return mgns, nil
		}
		cacheArgs := rcache.ResourceCacheArgs{
			ListFunc:    listFunc,
			ObjectType:  reflect.TypeOf(v3.GlobalNetworkSet{}),
			LogTypeDesc: "AddressGroupGlobalNetworkSets",
		}
		// Define a dynamic address groups controller.
		dagc = dynamicAddressGroupsController{
			tags:         tags,
			calicoClient: mccl,
			cache:        rcache.NewResourceCache(cacheArgs),
			pancli:       mpcl,
			isInSync:     true,
		}
		// Define the cache.
		for _, gns := range globalNetworkSetStartingState {
			dagc.cache.Set(gns.Name, gns)
		}
	})

	It("should handle an error other than KeyNotFound returned by datastore Get", func() {
		By("Defining the value returned by datastore Get as not a KeyNotFound error.")
		key := FakeGnsClientErrorKey
		err := dagc.syncToDatastore(key)

		By("Validating the returned error is \"some error\".")
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal(FakeGnsClientSomeErrorMessage))
	})

	It("should handle the datastor re-syncing a non-Panorama GlobalNetworkSet, without annotation: \"firewall.tigera.io/type\": \"Panorama\", returned by the datastore Get", func() {
		By("Defining a non-Panorama GlobalNetworkSet.")
		// Does not contain the "firewall.tigera.io/type":"Panorama" annotation.
		nonPanoramaGlobalNetworkSetInput := v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pan.addressgroup8-ri4np",
				Labels: map[string]string{
					"firewall.tigera.io/tag1": "",
					"firewall.tigera.io/tag3": "",
				},
				Annotations: map[string]string{
					"firewall.tigera.io/device-groups": "shared",
					"firewall.tigera.io/errors":        "unsupported-ip-ranges-present,unsupported-ip-wildcards-present",
					"firewall.tigera.io/name":          "address_group8",
					"firewall.tigera.io/object-type":   "AddressGroup",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/31", "10.10.10.11/31", "10.10.10.12/31", "10.10.10.13/31",
				},
				AllowedEgressDomains: []string{
					"www.tigera-test1.gr", "www.tigera-test4.gr", "www.tigera-test5.gr",
				},
			},
		}
		validPanoramaGlobalNetworkSetInput := v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pan.addressgroup8-ri4np",
				Labels: map[string]string{
					"firewall.tigera.io/tag1": "",
					"firewall.tigera.io/tag3": "",
				},
				Annotations: map[string]string{
					"firewall.tigera.io/device-groups": "shared",
					"firewall.tigera.io/errors":        "unsupported-ip-ranges-present,unsupported-ip-wildcards-present",
					"firewall.tigera.io/name":          "address_group8",
					"firewall.tigera.io/object-type":   "AddressGroup",
					"firewall.tigera.io/type":          "Panorama",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/31", "10.10.10.11/31", "10.10.10.12/31", "10.10.10.13/31",
				},
				AllowedEgressDomains: []string{
					"www.tigera-test1.gr", "www.tigera-test4.gr", "www.tigera-test5.gr",
				},
			},
		}

		By("Defining the GNS key.")
		key := nonPanoramaGlobalNetworkSetInput.Name

		By("Sync the key with a non-Panorama GlobalNetworkSet.")
		updateVal, err := mccl.gnsClient.Update(dagc.ctx, &nonPanoramaGlobalNetworkSetInput, metav1.UpdateOptions{})
		validatePanoramaGnsFields(*updateVal, nonPanoramaGlobalNetworkSetInput)
		Expect(err).To(BeNil())

		By("Testing a sync of a non-Panorama GlobalNetworkSet.")
		gnsBefore, _ := mccl.gnsClient.Get(dagc.ctx, key, metav1.GetOptions{})
		isPanorama := dagc.isPanoramaGlobalNetworkSet(gnsBefore)
		Expect(isPanorama).To(BeFalse())

		By("Sync the non-Panorama GlobalNetworkSet with the context of the cache back to a valid GNS.")
		dagc.cache.Set(key, validPanoramaGlobalNetworkSetInput)
		err = dagc.syncToDatastore(key)
		Expect(err).To(BeNil())

		By("Loading the expected data")
		var expectedGnsList []v3.GlobalNetworkSet
		file := fmt.Sprintf("%s/%s", expectedDataFolder, "expectedGnsList.json")
		panutils.LoadData(file, &expectedGnsList)

		By("Validating the datastore contains the expected values")
		gnsListAfterUpdate, _ := mccl.gnsClient.List(dagc.ctx, metav1.ListOptions{})
		Expect(len(gnsListAfterUpdate.Items)).To(Equal(len(expectedGnsList)))
		for _, expectedGns := range expectedGnsList {
			item, err := mccl.gnsClient.Get(dagc.ctx, expectedGns.Name, metav1.GetOptions{})
			Expect(err).To(BeNil())
			validatePanoramaGnsFields(*item, expectedGns)
		}
	})

	It("should handle a key that is stored in the datastore but not in the cache.", func() {
		By("Defining a Panorama GlobalNetworkSet.")
		validPanoramaglobalNetworkSetInput := v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pan.addressgroup22-ri4np",
				Labels: map[string]string{
					"firewall.tigera.io/tag1": "",
					"firewall.tigera.io/tag3": "",
				},
				Annotations: map[string]string{
					"firewall.tigera.io/device-groups": "shared",
					"firewall.tigera.io/errors":        "unsupported-ip-ranges-present,unsupported-ip-wildcards-present",
					"firewall.tigera.io/name":          "address_group22",
					"firewall.tigera.io/object-type":   "AddressGroup",
					"firewall.tigera.io/type":          "Panorama",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/31", "10.10.10.12/31", "10.10.10.13/31",
				},
				AllowedEgressDomains: []string{
					"www.tigera-test1.gr", "www.tigera-test4.gr", "www.tigera-test5.gr", "www.tigera-test9.gr",
				},
			},
		}

		By("Defining the GNS key.")
		key := validPanoramaglobalNetworkSetInput.Name

		By("Creating the GlobalNetworkSet in the datastore.")
		_, err := mccl.gnsClient.Create(dagc.ctx, &validPanoramaglobalNetworkSetInput, metav1.CreateOptions{})

		By("Validating the datastore Create did not return an error.")
		Expect(err).To(BeNil())

		By("Validating the cache does not contain the key.")
		_, found := dagc.cache.Get(key)
		Expect(found).NotTo(BeTrue())

		By("Validating the datastore does contain the key.")
		val, err := mccl.gnsClient.Get(dagc.ctx, key, metav1.GetOptions{})
		Expect(err).To(BeNil())
		Expect(val.Name).To(Equal(key))

		By("Syncing the datastore with the case. Delete key from datastore.")
		err = dagc.syncToDatastore(key)

		By("Validating sync to datastore does not return an error.")
		Expect(err).To(BeNil())

		By("Validating key has been deleted in the datastore.")
		_, err = mccl.gnsClient.Get(dagc.ctx, key, metav1.GetOptions{})
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal(" \"NotFound\" not found"))

		By("Loading the expected data")
		var expectedGnsList []v3.GlobalNetworkSet
		file := fmt.Sprintf("%s/%s", expectedDataFolder, "expectedGnsList.json")
		panutils.LoadData(file, &expectedGnsList)

		By("Verifying the datastore against the expected result list.")
		gnsListAfterSync, _ := mccl.gnsClient.List(dagc.ctx, metav1.ListOptions{})
		Expect(len(gnsListAfterSync.Items)).To(Equal(len(expectedGnsList)))
		for _, expectedGns := range expectedGnsList {
			item, err := mccl.gnsClient.Get(dagc.ctx, expectedGns.Name, metav1.GetOptions{})
			Expect(err).To(BeNil())
			validatePanoramaGnsFields(*item, expectedGns)
		}
	})

	It("should handle creating a new key in the datastore.", func() {
		By("Defining a Panorama GlobalNetworkSet.")
		validPanoramaGlobalNetworkSetInput := v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pan.addressgroup200-ri4np",
				Labels: map[string]string{
					"firewall.tigera.io/tag3": "",
					"firewall.tigera.io/tag1": "",
				},
				Annotations: map[string]string{
					"firewall.tigera.io/errors":        "unsupported-ip-ranges-present,unsupported-ip-wildcards-present",
					"firewall.tigera.io/name":          "address_group200",
					"firewall.tigera.io/object-type":   "AddressGroup",
					"firewall.tigera.io/type":          "Panorama",
					"firewall.tigera.io/device-groups": "shared",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/31", "10.10.10.12/31", "10.10.10.13/31",
				},
				AllowedEgressDomains: []string{
					"www.tigera-test1.gr", "www.tigera-test4.gr", "www.tigera-test5.gr", "www.tigera-test9.gr",
				},
			},
		}

		By("Defining the GNS key.")
		key := validPanoramaGlobalNetworkSetInput.Name

		By("Deleting the key, if it exists, from the datastore")
		err := dagc.calicoClient.GlobalNetworkSets().Delete(dagc.ctx, key, metav1.DeleteOptions{})
		Expect(err).To(BeNil())

		By("Validating the datastore does contain the key.")
		_, err = dagc.calicoClient.GlobalNetworkSets().Get(dagc.ctx, key, metav1.GetOptions{})
		Expect(err.Error()).To(Equal(" \"NotFound\" not found"))

		By("Validating the key does not exist in the cache.")
		_, found := dagc.cache.Get(key)
		Expect(found).NotTo(BeTrue())

		By("Inserting the key into the cache.")
		dagc.cache.Set(key, validPanoramaGlobalNetworkSetInput)

		By("Validating the cache does contain the key.")
		_, found = dagc.cache.Get(key)
		Expect(found).To(BeTrue())

		By("Syncing the datastore and creating new key in datastore.")
		err = dagc.syncToDatastore(key)
		Expect(err).To(BeNil())

		By("Validating key has been created in the datastore.")
		gns, err := mccl.gnsClient.Get(dagc.ctx, key, metav1.GetOptions{})
		Expect(err).To(BeNil())
		// Validate the Panorama fields are as expected.
		validatePanoramaGnsFields(*gns, validPanoramaGlobalNetworkSetInput)

		By("Loading the expected gns cached with added key data")
		var expectedGnsCacheWithAddedKey []v3.GlobalNetworkSet
		file := fmt.Sprintf("%s/%s", expectedDataFolder, "expectedGnsCacheWithAddedKey.json")
		panutils.LoadData(file, &expectedGnsCacheWithAddedKey)

		By("Validating the cache contains the expected values")
		keys := dagc.cache.ListKeys()
		Expect(len(keys)).To(Equal(len(expectedGnsCacheWithAddedKey)))
		for _, gnsVal := range expectedGnsCacheWithAddedKey {
			item, exists := dagc.cache.Get(gnsVal.Name)
			Expect(exists).To(BeTrue())
			validatePanoramaGnsFields(gnsVal, item.(v3.GlobalNetworkSet))
		}
	})

	It("should handle updating a key in the datastore.", func() {
		By("Defining a Panorama GlobalNetworkSet.")
		validPanoramaGlobalNetworkSetInput := v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pan.addressgroup8-ri4np",
				Labels: map[string]string{
					"firewall.tigera.io/tag3": "",
					"firewall.tigera.io/tag1": "",
				},
				Annotations: map[string]string{
					"firewall.tigera.io/errors":        "unsupported-ip-ranges-present,unsupported-ip-wildcards-present",
					"firewall.tigera.io/name":          "address_group8",
					"firewall.tigera.io/object-type":   "AddressGroup",
					"firewall.tigera.io/type":          "Panorama",
					"firewall.tigera.io/device-groups": "shared",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/31", "10.10.10.11/31", "10.10.10.12/31", "10.10.10.13/31",
				},
				AllowedEgressDomains: []string{
					"www.tigera-test1.gr", "www.tigera-test4.gr", "www.tigera-test5.gr",
				},
			},
		}

		By("Defining an new cache entry for Panorama GlobalNetworkSet.")
		cachePanoramaGlobalNetworkSetInput := v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pan.addressgroup8-ri4np",
				Labels: map[string]string{
					"firewall.tigera.io/tag3": "",
					"firewall.tigera.io/tag1": "",
				},
				Annotations: map[string]string{
					"firewall.tigera.io/errors":        "unsupported-ip-ranges-present,unsupported-ip-wildcards-present",
					"firewall.tigera.io/name":          "address_group8",
					"firewall.tigera.io/object-type":   "AddressGroup",
					"firewall.tigera.io/type":          "Panorama",
					"firewall.tigera.io/device-groups": "shared",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/31", "10.10.10.13/31",
				},
				AllowedEgressDomains: []string{
					"www.tigera-test1.gr", "www.tigera-test4.gr", "www.tigera-test5.gr", "www.tigera-test10.gr",
				},
			},
		}

		By("Defining the GNS key.")
		key := validPanoramaGlobalNetworkSetInput.Name

		By("Validating the datastore does contain the key.")
		oldGns, err := dagc.calicoClient.GlobalNetworkSets().Get(dagc.ctx, key, metav1.GetOptions{})
		Expect(err).To(BeNil())
		// Verify the relevant datastore GNS fields are equal to the inserted GNS ones.
		validatePanoramaGnsFields(*oldGns, validPanoramaGlobalNetworkSetInput)

		By("Updating the key in the cache with the value to be inserted into the datastore.")
		dagc.cache.Set(key, cachePanoramaGlobalNetworkSetInput)

		By("Validating the cache does contain the key.")
		item, found := dagc.cache.Get(key)
		cacheGns := item.(v3.GlobalNetworkSet)
		Expect(found).To(BeTrue())
		// The datastore and cache entries for the same key differ in Nets and AllowedEgressDomains.
		Expect(oldGns.Spec.Nets).NotTo(Equal(cacheGns.Spec.Nets))
		Expect(oldGns.Spec.AllowedEgressDomains).NotTo(Equal(cacheGns.Spec.AllowedEgressDomains))

		By("Syncing the datastore with the new GNS by updating the key.")
		sync := dagc.syncToDatastore(key)
		Expect(sync).To(BeNil())

		By("Validating key has been updated in the datastore.")
		gns, err := mccl.gnsClient.Get(dagc.ctx, key, metav1.GetOptions{})
		Expect(err).To(BeNil())
		// Validate the datastore fields have been updated and are now equal to the inserted GNS.
		validatePanoramaGnsFields(*gns, cachePanoramaGlobalNetworkSetInput)

		By("Loading the expected gns cached with added key data")
		var expectedGnsListWithUpdatedKey []v3.GlobalNetworkSet
		file := fmt.Sprintf("%s/%s", expectedDataFolder, "expectedGnsListWithUpdatedKey.json")
		panutils.LoadData(file, &expectedGnsListWithUpdatedKey)

		By("Validating the cache contains the expected values")
		keys := dagc.cache.ListKeys()
		Expect(len(keys)).To(Equal(len(expectedGnsListWithUpdatedKey)))
		for _, gnsVal := range expectedGnsListWithUpdatedKey {
			item, exists := dagc.cache.Get(gnsVal.Name)
			Expect(exists).To(BeTrue())
			validatePanoramaGnsFields(gnsVal, item.(v3.GlobalNetworkSet))
		}
	})
})

// updateCache
var _ = Describe("Tests controller updateCache functionality", func() {
	var (
		dagc                          dynamicAddressGroupsController
		mccl                          *FakeDagCalicoClient
		tags                          set.Set[string]
		globalNetworkSetStartingState []v3.GlobalNetworkSet

		deviceGroupsTest1 = dvgrp.Entry{
			Name: "device_group1",
		}
	)

	BeforeEach(func() {
		// Setup tags input for the dynamic address groups controller.
		tags = set.FromArray([]string{"tag1", "tag2", "tag3"})
		// Load the input data.
		file := fmt.Sprintf("%s/%s", inputDataFolder, "globalNetworkSetStartingState.json")
		panutils.LoadData(file, &globalNetworkSetStartingState)
		// Define a mock calico client, with sample datastore input.
		mccl = NewFakeDagCalicoClient(globalNetworkSetStartingState)
	})

	It("should handle an error generated by the call to GetAddressGroups", func() {
		By("Defining the dynamic address groups controller with the mock correct Panorama client")
		// Define a mock Panorama client.
		mpcl := &mocks.MockPanoramaClient{}
		mpcl.On("GetAddressEntries", "").Return([]addr.Entry{}, nil)
		mpcl.On("GetAddressGroupEntries", "").Return([]addrgrp.Entry{}, fmt.Errorf("error getting address group entries"))
		mpcl.On("GetClient").Return(&panw.Panorama{})
		mpcl.On("GetDeviceGroupEntry", "").Return(dvgrp.Entry{}, nil)
		mpcl.On("GetDeviceGroups").Return([]string{""}, nil)

		// Create a cache to store GlobalNetworkSets in.
		mgns := make(map[string]interface{})
		listFunc := func() (map[string]interface{}, error) {
			return mgns, nil
		}
		cacheArgs := rcache.ResourceCacheArgs{
			ListFunc:    listFunc,
			ObjectType:  reflect.TypeOf(v3.GlobalNetworkSet{}),
			LogTypeDesc: "AddressGroupGlobalNetworkSets",
		}
		// Define a dynamic address groups controller.
		dagc = dynamicAddressGroupsController{
			tags:         tags,
			calicoClient: mccl,
			cache:        rcache.NewResourceCache(cacheArgs),
			pancli:       mpcl,
			isInSync:     true,
		}
		// Define the cache.
		for _, gns := range globalNetworkSetStartingState {
			dagc.cache.Set(gns.Name, gns)
		}

		By("Verifying the call to GetAddressGroupEntries() returns an error")
		_, err := mpcl.GetAddressGroupEntries("")
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("error getting address group entries"))

		By("Verifying the updateCache returns successfully, when GetAddressGroupEntries() returns an error.")
		dagc.isInSync = true
		dagc.updateCache()
	})

	It("should updating the cache with fresh values from the call to AddressGroups().", func() {
		By("Load the input data")
		var addressGroupsDeviceGroup1Test []addrgrp.Entry
		file := fmt.Sprintf("%s/%s", inputDataFolder, "addressGroupsDeviceGroup1Test.json")
		panutils.LoadData(file, &addressGroupsDeviceGroup1Test)

		By("Defining the dynamic address groups controller with the mock Panorama client")
		// Define a mock Panorama client.
		mpcl := &mocks.MockPanoramaClient{}
		mpcl.On("GetAddressEntries", "").Return([]addr.Entry{}, nil)
		mpcl.On("GetAddressGroupEntries", "").Return(addressGroupsDeviceGroup1Test, nil)
		mpcl.On("GetClient").Return(&panw.Panorama{})
		mpcl.On("GetDeviceGroupEntry", "").Return(deviceGroupsTest1, nil)
		mpcl.On("GetDeviceGroups").Return([]string{}, nil)

		// Create a cache to store GlobalNetworkSets in.
		mgns := make(map[string]interface{})
		listFunc := func() (map[string]interface{}, error) {
			return mgns, nil
		}
		cacheArgs := rcache.ResourceCacheArgs{
			ListFunc:    listFunc,
			ObjectType:  reflect.TypeOf(v3.GlobalNetworkSet{}),
			LogTypeDesc: "AddressGroupGlobalNetworkSets",
		}
		// Define a dynamic address groups controller.
		dagc = dynamicAddressGroupsController{
			tags:         tags,
			calicoClient: mccl,
			cache:        rcache.NewResourceCache(cacheArgs),
			pancli:       mpcl,
			isInSync:     true,
		}

		By("Verifying the cache is empty.")
		Expect(len(dagc.cache.ListKeys())).To(Equal(0))

		By("Adding a key in the cache that does not exist in the list returned by GetAddressGroupEntries.")
		gns := v3.GlobalNetworkSet{}
		gns.Name = panutils.GetRFC1123Name("pan.address_group30")
		key := gns.Name
		dagc.cache.Set(key, gns)

		By("Verifying the updateCache does not update isInSync, as it has returned earlier")
		dagc.updateCache()

		By("Verifying the additional key no longer exists in the cache.")
		_, found := dagc.cache.Get(key)
		Expect(found).To(BeFalse())

		By("Verifying the list of keys in the cache equals the address groups translated into Gns when updatedCache is called.")
		keys := dagc.cache.ListKeys()
		names := func() []string {
			names := make([]string, 0, len(addressGroupsDeviceGroup1Test))
			for _, gns := range addressGroupsDeviceGroup1Test {
				names = append(names, panutils.GetRFC1123Name("pan."+gns.Name))
			}
			return names
		}()
		sort.Strings(keys)
		sort.Strings(names)
		Expect(reflect.DeepEqual(keys, names)).To(BeTrue())
	})
})

// convertAddressGroupToGlobalNetworkSet
var _ = DescribeTable(
	"convertAddressGroupToGlobalNetworkSet",
	func(ag panutils.AddressGroup, expectedGns v3.GlobalNetworkSet) {
		tags := set.FromArray([]string{"tag1", "tag2", "tag3"})
		dagc := dynamicAddressGroupsController{
			tags: tags,
		}
		gns := dagc.convertAddressGroupToGlobalNetworkSet(ag)

		Expect(gns).To(Equal(expectedGns))
	},
	Entry(
		"Source AddressGroup is empty.",
		panutils.AddressGroup{},
		v3.GlobalNetworkSet{
			TypeMeta: metav1.TypeMeta{Kind: "", APIVersion: ""},
			ObjectMeta: metav1.ObjectMeta{
				Name:   "pan-74128",
				Labels: map[string]string{"firewall.tigera.io/address-group": ""},
				Annotations: map[string]string{
					"firewall.tigera.io/name":          "",
					"firewall.tigera.io/device-groups": "shared",
					"firewall.tigera.io/errors":        "",
					"firewall.tigera.io/type":          "Panorama",
					"firewall.tigera.io/object-type":   "AddressGroup",
				},
				OwnerReferences: nil,
				Finalizers:      nil,
				ManagedFields:   nil,
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets:                 nil,
				AllowedEgressDomains: nil,
			},
		},
	),
	Entry(
		"Typical address group.",
		panutils.AddressGroup{
			Entry: addrgrp.Entry{
				Name:            "address.grp1",
				Description:     "",
				StaticAddresses: []string{}, // unordered
				DynamicMatch:    "tag1 OR (tag3 AND tag6)",
				Tags: []string{
					"tag2",
				}, // ordered
			},
			Addresses: panutils.Addresses{
				IpNetmasks: []string{
					"10.10.10.10/32",
					"10.10.10.11/20",
					"192.168.204.204/1",
				},
				Fqdns: []string{
					"www.tigera.io",
					"projectcalico.org",
				},
				IpRanges: []string{
					"1.1.1.1-2.2.2.2",
				},
				IpWildcards: []string{
					"10.132.1.2/0.0.2.255",
					"192.132.3.4/0.0.2.50",
				},
			},
			Err: errors.New("invalid group"),
		},
		v3.GlobalNetworkSet{
			TypeMeta: metav1.TypeMeta{Kind: "", APIVersion: ""},
			ObjectMeta: metav1.ObjectMeta{
				Name: "pan.address.grp1",
				Labels: map[string]string{
					"firewall.tigera.io/address-group": "address.grp1",
					"firewall.tigera.io/tag2":          "",
				},
				Annotations: map[string]string{
					"firewall.tigera.io/name":          "address.grp1",
					"firewall.tigera.io/device-groups": "shared",
					"firewall.tigera.io/errors":        "unsupported-ip-ranges-present,unsupported-ip-wildcards-present,invalid group",
					"firewall.tigera.io/type":          "Panorama",
					"firewall.tigera.io/object-type":   "AddressGroup",
				},
				OwnerReferences: nil,
				Finalizers:      nil,
				ManagedFields:   nil,
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets:                 []string{"10.10.10.10/32", "10.10.10.11/20", "192.168.204.204/1"},
				AllowedEgressDomains: []string{"www.tigera.io", "projectcalico.org"},
			},
		},
	),
)

// copyGlobalNetworkSet
var _ = DescribeTable(
	"copyGlobalNetworkSet",
	func(src v3.GlobalNetworkSet, expectedDestGns v3.GlobalNetworkSet) {
		dagc := dynamicAddressGroupsController{}
		dest := v3.GlobalNetworkSet{}
		dagc.copyGlobalNetworkSet(&dest, src)

		Expect(dest).To(Equal(expectedDestGns))
	},
	Entry(
		"Source GlobalNetworkSet is nil.",
		nil,
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		},
	),
	Entry(
		"Source GlobalNetworkSet with an empty name, and Labels and Annotations are nil.",
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "",
				Labels:      nil,
				Annotations: nil,
			},
		},
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "",
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		},
	),
	Entry(
		"Source GlobalNetworkSet with name defined, and Labels and Annotations are nil.",
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "gnsName",
				Labels:      nil,
				Annotations: nil,
			},
		},
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "gnsName",
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		},
	),
	Entry(
		"Source GlobalNetworkSet with name defined, Labels are set and Annotations are nil.",
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{},
			},
		},
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{},
			},
		},
	),
	Entry(
		"Source GlobalNetworkSet with name defined, Labels are set and Annotations are set.",
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{
					FirewallPrefix + "device-groups": "shared",
					FirewallPrefix + "errors":        "unsupported-ip-ranges-present",
					FirewallPrefix + "name":          "address.grp1.shared",
					FirewallPrefix + "object-type":   "AddressGroup",
					FirewallPrefix + "type":          "Panorama",
				},
			},
		},
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{
					FirewallPrefix + "device-groups": "shared",
					FirewallPrefix + "errors":        "unsupported-ip-ranges-present",
					FirewallPrefix + "name":          "address.grp1.shared",
					FirewallPrefix + "object-type":   "AddressGroup",
					FirewallPrefix + "type":          "Panorama",
				},
			},
		},
	),
	Entry(
		"Source GlobalNetworkSet with name defined, Labels are set and Annotations are set, and .",
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{
					FirewallPrefix + "device-groups": "shared",
					FirewallPrefix + "errors":        "unsupported-ip-ranges-present",
					FirewallPrefix + "name":          "address.grp1.shared",
					FirewallPrefix + "object-type":   "AddressGroup",
					FirewallPrefix + "type":          "Panorama",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets:                 nil,
				AllowedEgressDomains: nil,
			},
		},
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{
					FirewallPrefix + "device-groups": "shared",
					FirewallPrefix + "errors":        "unsupported-ip-ranges-present",
					FirewallPrefix + "name":          "address.grp1.shared",
					FirewallPrefix + "object-type":   "AddressGroup",
					FirewallPrefix + "type":          "Panorama",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets:                 nil,
				AllowedEgressDomains: nil,
			},
		},
	),
	Entry(
		"Source GlobalNetworkSet with name defined, Labels are set and Annotations are set, and .",
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{
					FirewallPrefix + "device-groups": "shared",
					FirewallPrefix + "errors":        "unsupported-ip-ranges-present",
					FirewallPrefix + "name":          "address.grp1.shared",
					FirewallPrefix + "object-type":   "AddressGroup",
					FirewallPrefix + "type":          "Panorama",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/32",
					"192.10.10.1/31",
					"192.10.10.1/32",
				},
				AllowedEgressDomains: []string{
					"www.tigera.io",
					"kubernetes.io",
					"www.projectcalico.org",
				},
			},
		},
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{
					FirewallPrefix + "device-groups": "shared",
					FirewallPrefix + "errors":        "unsupported-ip-ranges-present",
					FirewallPrefix + "name":          "address.grp1.shared",
					FirewallPrefix + "object-type":   "AddressGroup",
					FirewallPrefix + "type":          "Panorama",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/32",
					"192.10.10.1/31",
					"192.10.10.1/32",
				},
				AllowedEgressDomains: []string{
					"www.tigera.io",
					"kubernetes.io",
					"www.projectcalico.org",
				},
			},
		},
	),
	Entry(
		"Source GlobalNetworkSet where TypeMeta is not copied from src to dest.",
		v3.GlobalNetworkSet{
			TypeMeta: metav1.TypeMeta{
				Kind: "List",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{
					FirewallPrefix + "device-groups": "shared",
					FirewallPrefix + "errors":        "unsupported-ip-ranges-present",
					FirewallPrefix + "name":          "address.grp1.shared",
					FirewallPrefix + "object-type":   "AddressGroup",
					FirewallPrefix + "type":          "Panorama",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/32",
					"192.10.10.1/31",
					"192.10.10.1/32",
				},
				AllowedEgressDomains: []string{
					"www.tigera.io",
					"kubernetes.io",
					"www.projectcalico.org",
				},
			},
		},
		v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gnsName",
				Labels: map[string]string{
					FirewallPrefix + "tag1": "",
					FirewallPrefix + "tag4": "",
				},
				Annotations: map[string]string{
					FirewallPrefix + "device-groups": "shared",
					FirewallPrefix + "errors":        "unsupported-ip-ranges-present",
					FirewallPrefix + "name":          "address.grp1.shared",
					FirewallPrefix + "object-type":   "AddressGroup",
					FirewallPrefix + "type":          "Panorama",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: []string{
					"10.10.10.10/32",
					"192.10.10.1/31",
					"192.10.10.1/32",
				},
				AllowedEgressDomains: []string{
					"www.tigera.io",
					"kubernetes.io",
					"www.projectcalico.org",
				},
			},
		},
	),
)

// isPanoramaGlobalNetworkSet
var _ = DescribeTable(
	"isPanoramaGlobalNetworkSet",
	func(gns *v3.GlobalNetworkSet, expectedValue bool) {
		dagc := dynamicAddressGroupsController{}
		isPanGNS := dagc.isPanoramaGlobalNetworkSet(gns)

		Expect(isPanGNS).To(Equal(expectedValue))
	},
	Entry(
		"GlobalNetworkSet is nil.",
		nil,
		false,
	),
	Entry(
		"GlobalNetworkSet is empty.",
		&v3.GlobalNetworkSet{},
		false,
	),
	Entry(
		"GlobalNetworkSet is not Panorama. Does not contain an annotation",
		&v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gns1",
			},
		},
		false,
	),
	Entry(
		"GlobalNetworkSet is not Panorama. Does not contain \"firewall.tigera.io/type\" annotation",
		&v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gns1",
				Annotations: map[string]string{
					FirewallPrefix + "notType": "thisAnnotation",
				},
			},
		},
		false,
	),
	Entry(
		"GlobalNetworkSet is not Panorama. \"firewall.tigera.io/type\" annotation value is not Panorama",
		&v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gns1",
				Annotations: map[string]string{
					FirewallPrefix + "type": "NotPanorama",
				},
			},
		},
		false,
	),
	Entry(
		"GlobalNetworkSet is Panorama",
		&v3.GlobalNetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gns1",
				Annotations: map[string]string{
					FirewallPrefix + "type": "Panorama",
				},
			},
		},
		true,
	),
)

func validatePanoramaGnsFields(gns v3.GlobalNetworkSet, expectedGns v3.GlobalNetworkSet) {
	Expect(gns.Name).To(Equal(expectedGns.Name))
	Expect(gns.Annotations).To(Equal(expectedGns.Annotations))
	Expect(gns.ObjectMeta.Labels).To(Equal(expectedGns.Labels))
	Expect(gns.Spec.Nets).To(Equal(expectedGns.Spec.Nets))
	Expect(gns.Spec.AllowedEgressDomains).To(Equal(expectedGns.Spec.AllowedEgressDomains))
}
