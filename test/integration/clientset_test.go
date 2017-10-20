/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"

	// TODO: fix this upstream
	// we shouldn't have to install things to use our own generated client.

	// avoid error `servicecatalog/v1alpha1 is not enabled`

	_ "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/install"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v2"
	// avoid error `no kind is registered for the type metav1.ListOptions`
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/kubernetes/pkg/api/install"
	// our versioned types
	calicoclient "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"

	// our versioned client

	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestGroupVersion is trivial.
func TestGroupVersion(t *testing.T) {
	fmt.Println("GropVersion being tested.")
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &calico.NetworkPolicy{}
			})
			defer shutdownServer()
			if err := testGroupVersion(client); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run("group version", rootTestFunc()) {
		t.Error("test failed")
	}
}

func testGroupVersion(client calicoclient.Interface) error {
	gv := client.Projectcalico().RESTClient().APIVersion()
	if gv.Group != calico.GroupName {
		return fmt.Errorf("we should be testing the servicecatalog group, not %s", gv.Group)
	}
	return nil
}

func TestEtcdHealthCheckerSuccess(t *testing.T) {
	serverConfig := NewTestServerConfig()
	_, clientconfig, shutdownServer := withConfigGetFreshApiserverAndClient(t, serverConfig)
	t.Log(clientconfig.Host)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &http.Client{Transport: tr}
	resp, err := c.Get(clientconfig.Host + "/healthz")
	if nil != err || http.StatusOK != resp.StatusCode {
		t.Fatal("health check endpoint should not have failed")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("couldn't read response body", err)
	}
	if strings.Contains(string(body), "healthz check failed") {
		t.Fatal("health check endpoint should not have failed")
	}

	defer shutdownServer()
}

// TestNoName checks that all creates fail for objects that have no
// name given.
func TestNoName(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &calico.NetworkPolicy{}
			})
			defer shutdownServer()
			if err := testNoName(client); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run("no-name", rootTestFunc()) {
		t.Errorf("NoName test failed")
	}

}

func testNoName(client calicoclient.Interface) error {
	cClient := client.Projectcalico()

	ns := "namespace"

	if p, e := cClient.NetworkPolicies(ns).Create(&v2.NetworkPolicy{}); nil == e {
		return fmt.Errorf("needs a name (%s)", p.Name)
	}
	/*	if t, e := cClient.Tiers().Create(&v1.Tier{}); nil == e {
		return fmt.Errorf("needs a name (%s)", t.Name)
	}*/
	return nil
}

// TestNetworkPolicyClient exercises the NetworkPolicy client.
func TestNetworkPolicyClient(t *testing.T) {
	const name = "test-networkpolicy"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &calico.NetworkPolicy{}
			})
			defer shutdownServer()
			if err := testNetworkPolicyClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-networkpolicy test failed")
	}

}

func testNetworkPolicyClient(client calicoclient.Interface, name string) error {
	ns := "namespace"
	policyClient := client.Projectcalico().NetworkPolicies(ns)
	policy := &v2.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name}}

	// start from scratch
	policies, err := policyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if policies.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(policies.Items) > 0 {
		return fmt.Errorf("policies should not exist on start, had %v policies", len(policies.Items))
	}

	policyServer, err := policyClient.Create(policy)
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", policy, err)
	}
	if name != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	policies, err = policyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if 1 != len(policies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(policies.Items))
	}

	policyServer, err = policyClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting policy %s (%s)", name, err)
	}
	if name != policyServer.Name &&
		policy.ResourceVersion == policyServer.ResourceVersion {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	// check that the policy is the same from get and list
	/*policyListed := &policies.Items[0]
	if !reflect.DeepEqual(policyServer, policyListed) {
		fmt.Printf("Policy through Get: %v\n", policyServer)
		fmt.Printf("Policy through list: %v\n", policyListed)
		return fmt.Errorf(
			"Didn't get the same instance from list and get: diff: %v",
			diff.ObjectReflectDiff(policyServer, policyListed),
		)
	}*/
	// Watch Test:
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		opts := v1.ListOptions{}
		wIface, err := policyClient.Watch(opts)
		if nil != err {
			fmt.Errorf("Error on watch")
		}
		for e := range wIface.ResultChan() {
			fmt.Println("Watch object: ", e)
			break
		}
	}()

	err = policyClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("policy should be deleted (%s)", err)
	}
	wg.Wait()
	return nil
}

// TestTierClient exercises the Tier client.
func TestTierClient(t *testing.T) {
	const name = "test-tier"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &calico.Tier{}
			})
			defer shutdownServer()
			if err := testTierClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-tier test failed")
	}
}

func testTierClient(client calicoclient.Interface, name string) error {
	tierClient := client.Projectcalico().Tiers()
	tier := &v2.Tier{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	// start from scratch
	tiers, err := tierClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing tiers (%s)", err)
	}
	if tiers.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	tierServer, err := tierClient.Create(tier)
	if nil != err {
		return fmt.Errorf("error creating the tier '%v' (%v)", tier, err)
	}
	if name != tierServer.Name {
		return fmt.Errorf("didn't get the same tier back from the server \n%+v\n%+v", tier, tierServer)
	}

	tiers, err = tierClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing tiers (%s)", err)
	}

	tierServer, err = tierClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting tier %s (%s)", name, err)
	}
	if name != tierServer.Name &&
		tier.ResourceVersion == tierServer.ResourceVersion {
		return fmt.Errorf("didn't get the same tier back from the server \n%+v\n%+v", tier, tierServer)
	}

	err = tierClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("tier should be deleted (%s)", err)
	}

	return nil
}

// TestGlobalNetworkPolicyClient exercises the GlobalNetworkPolicy client.
func TestGlobalNetworkPolicyClient(t *testing.T) {
	const name = "test-globalnetworkpolicy"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &calico.GlobalNetworkPolicy{}
			})
			defer shutdownServer()
			if err := testGlobalNetworkPolicyClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-globalnetworkpolicy test failed")
	}

}

func testGlobalNetworkPolicyClient(client calicoclient.Interface, name string) error {
	globalNetworkPolicyClient := client.Projectcalico().GlobalNetworkPolicies()
	globalNetworkPolicy := &v2.GlobalNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name}}

	// start from scratch
	globalNetworkPolicies, err := globalNetworkPolicyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkPolicies (%s)", err)
	}
	if globalNetworkPolicies.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalNetworkPolicyServer, err := globalNetworkPolicyClient.Create(globalNetworkPolicy)
	if nil != err {
		return fmt.Errorf("error creating the globalNetworkPolicy '%v' (%v)", globalNetworkPolicy, err)
	}
	if name != globalNetworkPolicyServer.Name {
		return fmt.Errorf("didn't get the same globalNetworkPolicy back from the server \n%+v\n%+v", globalNetworkPolicy, globalNetworkPolicyServer)
	}

	globalNetworkPolicies, err = globalNetworkPolicyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkPolicies (%s)", err)
	}

	globalNetworkPolicyServer, err = globalNetworkPolicyClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalNetworkPolicy %s (%s)", name, err)
	}
	if name != globalNetworkPolicyServer.Name &&
		globalNetworkPolicy.ResourceVersion == globalNetworkPolicyServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalNetworkPolicy back from the server \n%+v\n%+v", globalNetworkPolicy, globalNetworkPolicyServer)
	}

	err = globalNetworkPolicyClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalNetworkPolicy should be deleted (%s)", err)
	}

	return nil
}
