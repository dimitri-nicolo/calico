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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	calico "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	_ "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/install"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"

	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

// TestGroupVersion is trivial.
func TestGroupVersion(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.NetworkPolicy{}
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
	gv := client.ProjectcalicoV3().RESTClient().APIVersion()
	if gv.Group != projectcalico.GroupName {
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
				return &projectcalico.NetworkPolicy{}
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
	cClient := client.ProjectcalicoV3()

	ns := "default"

	if p, e := cClient.NetworkPolicies(ns).Create(&v3.NetworkPolicy{}); nil == e {
		return fmt.Errorf("needs a name (%s)", p.Name)
	}

	return nil
}

// TestNetworkPolicyClient exercises the NetworkPolicy client.
func TestNetworkPolicyClient(t *testing.T) {
	const name = "test-networkpolicy"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.NetworkPolicy{}
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
	ns := "default"
	defaultTierPolicyName := "default" + "." + name
	policyClient := client.ProjectcalicoV3().NetworkPolicies(ns)
	policy := &v3.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: defaultTierPolicyName}}

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
	if defaultTierPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	updatedPolicy := policyServer
	updatedPolicy.Labels = map[string]string{"foo": "bar"}
	policyServer, err = policyClient.Update(updatedPolicy)
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", policy, err)
	}
	if defaultTierPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	// For testing out Tiered Policy
	tierClient := client.ProjectcalicoV3().Tiers()
	tier := &v3.Tier{
		ObjectMeta: metav1.ObjectMeta{Name: "net-sec"},
	}

	tierClient.Create(tier)
	defer func() {
		tierClient.Delete("net-sec", &metav1.DeleteOptions{})
	}()

	netSecPolicyName := "net-sec" + "." + name
	netSecPolicy := &v3.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netSecPolicyName}, Spec: calico.NetworkPolicySpec{Tier: "net-sec"}}
	policyServer, err = policyClient.Create(netSecPolicy)
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", netSecPolicy, err)
	}
	if netSecPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	// Should be listing the policy under default tier.
	policies, err = policyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if 1 != len(policies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(policies.Items))
	}

	// Should be listing the policy under "net-sec" tier
	policies, err = policyClient.List(metav1.ListOptions{FieldSelector: "spec.tier=net-sec"})
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
	opts := v1.ListOptions{Watch: true}
	wIface, err := policyClient.Watch(opts)
	if nil != err {
		return fmt.Errorf("Error on watch")
	}
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		for e := range wIface.ResultChan() {
			fmt.Println("Watch object: ", e)
			break
		}
	}()

	err = policyClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("policy should be deleted (%s)", err)
	}

	err = policyClient.Delete(netSecPolicyName, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("policy should be deleted (%s)", err)
	}

	wg.Wait()
	return nil
}

// TestStagedgNetworkPolicyClient exercises the StagedNetworkPolicy client.
func TestStagedNetworkPolicyClient(t *testing.T) {
	const name = "test-networkpolicy"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.NetworkPolicy{}
			})
			defer shutdownServer()
			if err := testStagedNetworkPolicyClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-stagednetworkpolicy test failed")
	}
}

func testStagedNetworkPolicyClient(client calicoclient.Interface, name string) error {
	ns := "default"
	defaultTierPolicyName := "default" + "." + name
	policyClient := client.ProjectcalicoV3().StagedNetworkPolicies(ns)
	policy := &v3.StagedNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: defaultTierPolicyName},
		Spec: calico.StagedNetworkPolicySpec{StagedAction: "Set", Selector: "foo == \"bar\""}}

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
	if defaultTierPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	updatedPolicy := policyServer
	updatedPolicy.Labels = map[string]string{"foo": "bar"}
	policyServer, err = policyClient.Update(updatedPolicy)
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", policy, err)
	}
	if defaultTierPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	// For testing out Tiered Policy
	tierClient := client.ProjectcalicoV3().Tiers()
	tier := &v3.Tier{
		ObjectMeta: metav1.ObjectMeta{Name: "net-sec"},
	}

	tierClient.Create(tier)
	defer func() {
		tierClient.Delete("net-sec", &metav1.DeleteOptions{})
	}()

	netSecPolicyName := "net-sec" + "." + name
	netSecPolicy := &v3.StagedNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netSecPolicyName}, Spec: calico.StagedNetworkPolicySpec{StagedAction: "Set", Selector: "foo == \"bar\"", Tier: "net-sec"}}
	policyServer, err = policyClient.Create(netSecPolicy)
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", netSecPolicy, err)
	}
	if netSecPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	// Should be listing the policy under default tier.
	policies, err = policyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if 1 != len(policies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(policies.Items))
	}

	// Should be listing the policy under "net-sec" tier
	policies, err = policyClient.List(metav1.ListOptions{FieldSelector: "spec.tier=net-sec"})
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

	// Watch Test:
	opts := v1.ListOptions{Watch: true}
	wIface, err := policyClient.Watch(opts)
	if nil != err {
		return fmt.Errorf("Error on watch")
	}
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		for e := range wIface.ResultChan() {
			fmt.Println("Watch object: ", e)
			break
		}
	}()

	err = policyClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("policy should be deleted (%s)", err)
	}

	err = policyClient.Delete(netSecPolicyName, &metav1.DeleteOptions{})
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
				return &projectcalico.Tier{}
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
	tierClient := client.ProjectcalicoV3().Tiers()
	tier := &v3.Tier{
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
				return &projectcalico.GlobalNetworkPolicy{}
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
	globalNetworkPolicyClient := client.ProjectcalicoV3().GlobalNetworkPolicies()
	defaultTierPolicyName := "default" + "." + name
	globalNetworkPolicy := &v3.GlobalNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: defaultTierPolicyName}}

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
	if defaultTierPolicyName != globalNetworkPolicyServer.Name {
		return fmt.Errorf("didn't get the same globalNetworkPolicy back from the server \n%+v\n%+v", globalNetworkPolicy, globalNetworkPolicyServer)
	}

	// For testing out Tiered Policy
	tierClient := client.ProjectcalicoV3().Tiers()
	tier := &v3.Tier{
		ObjectMeta: metav1.ObjectMeta{Name: "net-sec"},
	}

	tierClient.Create(tier)
	defer func() {
		tierClient.Delete("net-sec", &metav1.DeleteOptions{})
	}()

	netSecPolicyName := "net-sec" + "." + name
	netSecPolicy := &v3.GlobalNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netSecPolicyName}, Spec: calico.GlobalNetworkPolicySpec{Tier: "net-sec"}}
	globalNetworkPolicyServer, err = globalNetworkPolicyClient.Create(netSecPolicy)
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", netSecPolicy, err)
	}
	if netSecPolicyName != globalNetworkPolicyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", netSecPolicy, globalNetworkPolicyServer)
	}

	// Should be listing the policy under "default" tier
	globalNetworkPolicies, err = globalNetworkPolicyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkPolicies (%s)", err)
	}
	if 1 != len(globalNetworkPolicies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(globalNetworkPolicies.Items))
	}

	// Should be listing the policy under "net-sec" tier
	globalNetworkPolicies, err = globalNetworkPolicyClient.List(metav1.ListOptions{FieldSelector: "spec.tier=net-sec"})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkPolicies (%s)", err)
	}
	if 1 != len(globalNetworkPolicies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(globalNetworkPolicies.Items))
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

	err = globalNetworkPolicyClient.Delete(netSecPolicyName, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("policy should be deleted (%s)", err)
	}

	return nil
}

// TestStagedGlobalNetworkPolicyClient exercises the StagedGlobalNetworkPolicy client.
func TestStagedGlobalNetworkPolicyClient(t *testing.T) {
	const name = "test-stagedglobalnetworkpolicy"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.StagedGlobalNetworkPolicy{}
			})
			defer shutdownServer()
			if err := testStagedGlobalNetworkPolicyClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-Stagedglobalnetworkpolicy test failed")
	}
}

func testStagedGlobalNetworkPolicyClient(client calicoclient.Interface, name string) error {
	stagedGlobalNetworkPolicyClient := client.ProjectcalicoV3().StagedGlobalNetworkPolicies()
	defaultTierPolicyName := "default" + "." + name
	stagedGlobalNetworkPolicy := &v3.StagedGlobalNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: defaultTierPolicyName},
		Spec: calico.StagedGlobalNetworkPolicySpec{StagedAction: "Set", Selector: "foo == \"bar\""}}

	// start from scratch
	stagedGlobalNetworkPolicies, err := stagedGlobalNetworkPolicyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing stagedglobalNetworkPolicies (%s)", err)
	}
	if stagedGlobalNetworkPolicies.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	stagedGlobalNetworkPolicyServer, err := stagedGlobalNetworkPolicyClient.Create(stagedGlobalNetworkPolicy)
	if nil != err {
		return fmt.Errorf("error creating the stagedGlobalNetworkPolicy '%v' (%v)", stagedGlobalNetworkPolicy, err)
	}
	if defaultTierPolicyName != stagedGlobalNetworkPolicyServer.Name {
		return fmt.Errorf("didn't get the same stagedGlobalNetworkPolicy back from the server \n%+v\n%+v", stagedGlobalNetworkPolicy, stagedGlobalNetworkPolicyServer)
	}

	// For testing out Tiered Policy
	tierClient := client.ProjectcalicoV3().Tiers()
	tier := &v3.Tier{
		ObjectMeta: metav1.ObjectMeta{Name: "net-sec"},
	}

	tierClient.Create(tier)
	defer func() {
		tierClient.Delete("net-sec", &metav1.DeleteOptions{})
	}()

	netSecPolicyName := "net-sec" + "." + name
	netSecPolicy := &v3.StagedGlobalNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netSecPolicyName}, Spec: calico.StagedGlobalNetworkPolicySpec{StagedAction: "Set", Selector: "foo == \"bar\"", Tier: "net-sec"}}
	stagedGlobalNetworkPolicyServer, err = stagedGlobalNetworkPolicyClient.Create(netSecPolicy)
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", netSecPolicy, err)
	}
	if netSecPolicyName != stagedGlobalNetworkPolicyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", netSecPolicy, stagedGlobalNetworkPolicyServer)
	}

	// Should be listing the policy under "default" tier
	stagedGlobalNetworkPolicies, err = stagedGlobalNetworkPolicyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing stagedGlobalNetworkPolicies (%s)", err)
	}
	if 1 != len(stagedGlobalNetworkPolicies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(stagedGlobalNetworkPolicies.Items))
	}

	// Should be listing the policy under "net-sec" tier
	stagedGlobalNetworkPolicies, err = stagedGlobalNetworkPolicyClient.List(metav1.ListOptions{FieldSelector: "spec.tier=net-sec"})
	if err != nil {
		return fmt.Errorf("error listing stagedGlobalNetworkPolicies (%s)", err)
	}
	if 1 != len(stagedGlobalNetworkPolicies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(stagedGlobalNetworkPolicies.Items))
	}

	stagedGlobalNetworkPolicyServer, err = stagedGlobalNetworkPolicyClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting stagedGlobalNetworkPolicy %s (%s)", name, err)
	}
	if name != stagedGlobalNetworkPolicyServer.Name &&
		stagedGlobalNetworkPolicy.ResourceVersion == stagedGlobalNetworkPolicyServer.ResourceVersion {
		return fmt.Errorf("didn't get the same stagedGlobalNetworkPolicy back from the server \n%+v\n%+v", stagedGlobalNetworkPolicy, stagedGlobalNetworkPolicyServer)
	}

	err = stagedGlobalNetworkPolicyClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("stagedGlobalNetworkPolicy should be deleted (%s)", err)
	}

	err = stagedGlobalNetworkPolicyClient.Delete(netSecPolicyName, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("policy should be deleted (%s)", err)
	}

	return nil
}

// TestGlobalNetworkSetClient exercises the GlobalNetworkSet client.
func TestGlobalNetworkSetClient(t *testing.T) {
	const name = "test-globalnetworkset"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.GlobalNetworkSet{}
			})
			defer shutdownServer()
			if err := testGlobalNetworkSetClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-globalnetworkset test failed")
	}
}

func testGlobalNetworkSetClient(client calicoclient.Interface, name string) error {
	globalNetworkSetClient := client.ProjectcalicoV3().GlobalNetworkSets()
	globalNetworkSet := &v3.GlobalNetworkSet{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	// start from scratch
	globalNetworkSets, err := globalNetworkSetClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkSets (%s)", err)
	}
	if globalNetworkSets.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalNetworkSetServer, err := globalNetworkSetClient.Create(globalNetworkSet)
	if nil != err {
		return fmt.Errorf("error creating the globalNetworkSet '%v' (%v)", globalNetworkSet, err)
	}
	if name != globalNetworkSetServer.Name {
		return fmt.Errorf("didn't get the same globalNetworkSet back from the server \n%+v\n%+v", globalNetworkSet, globalNetworkSetServer)
	}

	globalNetworkSets, err = globalNetworkSetClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkSets (%s)", err)
	}

	globalNetworkSetServer, err = globalNetworkSetClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalNetworkSet %s (%s)", name, err)
	}
	if name != globalNetworkSetServer.Name &&
		globalNetworkSet.ResourceVersion == globalNetworkSetServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalNetworkSet back from the server \n%+v\n%+v", globalNetworkSet, globalNetworkSetServer)
	}

	err = globalNetworkSetClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalNetworkSet should be deleted (%s)", err)
	}

	return nil
}

// TestNetworkSetClient exercises the NetworkSet client.
func TestNetworkSetClient(t *testing.T) {
	const name = "test-networkset"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.NetworkSet{}
			})
			defer shutdownServer()
			if err := testNetworkSetClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-networkset test failed")
	}
}

func testNetworkSetClient(client calicoclient.Interface, name string) error {
	ns := "default"
	networkSetClient := client.ProjectcalicoV3().NetworkSets(ns)
	networkSet := &v3.NetworkSet{ObjectMeta: metav1.ObjectMeta{Name: name}}

	// start from scratch
	networkSets, err := networkSetClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing networkSets (%s)", err)
	}
	if networkSets.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(networkSets.Items) > 0 {
		return fmt.Errorf("networkSets should not exist on start, had %v networkSets", len(networkSets.Items))
	}

	networkSetServer, err := networkSetClient.Create(networkSet)
	if nil != err {
		return fmt.Errorf("error creating the networkSet '%v' (%v)", networkSet, err)
	}

	updatedNetworkSet := networkSetServer
	updatedNetworkSet.Labels = map[string]string{"foo": "bar"}
	networkSetServer, err = networkSetClient.Update(updatedNetworkSet)
	if nil != err {
		return fmt.Errorf("error updating the networkSet '%v' (%v)", networkSet, err)
	}

	// Should be listing the networkSet.
	networkSets, err = networkSetClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing networkSets (%s)", err)
	}
	if 1 != len(networkSets.Items) {
		return fmt.Errorf("should have exactly one networkSet, had %v networkSets", len(networkSets.Items))
	}

	networkSetServer, err = networkSetClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting networkSet %s (%s)", name, err)
	}
	if name != networkSetServer.Name &&
		networkSet.ResourceVersion == networkSetServer.ResourceVersion {
		return fmt.Errorf("didn't get the same networkSet back from the server \n%+v\n%+v", networkSet, networkSetServer)
	}

	// Watch Test:
	opts := v1.ListOptions{Watch: true}
	wIface, err := networkSetClient.Watch(opts)
	if nil != err {
		return fmt.Errorf("Error on watch")
	}
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		for e := range wIface.ResultChan() {
			fmt.Println("Watch object: ", e)
			break
		}
	}()

	err = networkSetClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("networkSet should be deleted (%s)", err)
	}

	wg.Wait()
	return nil
}

// TestLicenseKeyClient exercises the LicenseKey client.
func TestLicenseKeyClient(t *testing.T) {
	const name = "default"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.LicenseKey{}
			})
			defer shutdownServer()
			if err := testLicenseKeyClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-licensekey test failed")
	}
}

func testLicenseKeyClient(client calicoclient.Interface, name string) error {
	licenseKeyClient := client.ProjectcalicoV3().LicenseKeys()

	licenseKeys, err := licenseKeyClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing licenseKeys (%s)", err)
	}
	if licenseKeys.Items == nil {
		return fmt.Errorf("items field should not be set to nil")
	}

	// Validate that a license not encrypted with production key is rejected
	corruptLicenseKey := &v3.LicenseKey{ObjectMeta: metav1.ObjectMeta{Name: name}}

	_, err = licenseKeyClient.Create(corruptLicenseKey)
	if err == nil {
		return fmt.Errorf("expected creating the emptyLicenseKey")
	}

	// Confirm that valid, but expired licenses, are rejected
	expiredLicenseKey := getExpiredLicenseKey(name)
	_, err = licenseKeyClient.Create(expiredLicenseKey)
	if err == nil {
		return fmt.Errorf("expected creating the expiredLicenseKey")
	} else if err.Error() != "LicenseKey.projectcalico.org \"default\" is invalid: LicenseKeySpec.token: Internal error: the license you're trying to create expired on 2019-02-08 07:59:59 +0000 UTC" {
		fmt.Printf("Incorrect error: %+v\n", err)
	}

	return nil
}

const expiredLicenseCertificate = `-----BEGIN CERTIFICATE-----
MIIFxjCCA66gAwIBAgIQVq3rz5D4nQF1fIgMEh71DzANBgkqhkiG9w0BAQsFADCB
tTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1
cml0eSA8c2lydEB0aWdlcmEuaW8+MT8wPQYDVQQDEzZUaWdlcmEgRW50aXRsZW1l
bnRzIEludGVybWVkaWF0ZSBDZXJ0aWZpY2F0ZSBBdXRob3JpdHkwHhcNMTgwNDA1
MjEzMDI5WhcNMjAxMDA2MjEzMDI5WjCBnjELMAkGA1UEBhMCVVMxEzARBgNVBAgT
CkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xFDASBgNVBAoTC1Rp
Z2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1cml0eSA8c2lydEB0aWdlcmEuaW8+MSgw
JgYDVQQDEx9UaWdlcmEgRW50aXRsZW1lbnRzIENlcnRpZmljYXRlMIIBojANBgkq
hkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAwg3LkeHTwMi651af/HEXi1tpM4K0LVqb
5oUxX5b5jjgi+LHMPzMI6oU+NoGPHNqirhAQqK/k7W7r0oaMe1APWzaCAZpHiMxE
MlsAXmLVUrKg/g+hgrqeije3JDQutnN9h5oZnsg1IneBArnE/AKIHH8XE79yMG49
LaKpPGhpF8NoG2yoWFp2ekihSohvqKxa3m6pxoBVdwNxN0AfWxb60p2SF0lOi6B3
hgK6+ILy08ZqXefiUs+GC1Af4qI1jRhPkjv3qv+H1aQVrq6BqKFXwWIlXCXF57CR
hvUaTOG3fGtlVyiPE4+wi7QDo0cU/+Gx4mNzvmc6lRjz1c5yKxdYvgwXajSBx2pw
kTP0iJxI64zv7u3BZEEII6ak9mgUU1CeGZ1KR2Xu80JiWHAYNOiUKCBYHNKDCUYl
RBErYcAWz2mBpkKyP6hbH16GjXHTTdq5xENmRDHabpHw5o+21LkWBY25EaxjwcZa
Y3qMIOllTZ2iRrXu7fSP6iDjtFCcE2bFAgMBAAGjZzBlMA4GA1UdDwEB/wQEAwIF
oDATBgNVHSUEDDAKBggrBgEFBQcDAjAdBgNVHQ4EFgQUIY7LzqNTzgyTBE5efHb5
kZ71BUEwHwYDVR0jBBgwFoAUxZA5kifzo4NniQfGKb+4wruTIFowDQYJKoZIhvcN
AQELBQADggIBAAK207LaqMrnphF6CFQnkMLbskSpDZsKfqqNB52poRvUrNVUOB1w
3dSEaBUjhFgUU6yzF+xnuH84XVbjD7qlM3YbdiKvJS9jrm71saCKMNc+b9HSeQAU
DGY7GPb7Y/LG0GKYawYJcPpvRCNnDLsSVn5N4J1foWAWnxuQ6k57ymWwcddibYHD
OPakOvO4beAnvax3+K5dqF0bh2Np79YolKdIgUVzf4KSBRN4ZE3AOKlBfiKUvWy6
nRGvu8O/8VaI0vGaOdXvWA5b61H0o5cm50A88tTm2LHxTXynE3AYriHxsWBbRpoM
oFnmDaQtGY67S6xGfQbwxrwCFd1l7rGsyBQ17cuusOvMNZEEWraLY/738yWKw3qX
U7KBxdPWPIPd6iDzVjcZrS8AehUEfNQ5yd26gDgW+rZYJoAFYv0vydMEyoI53xXs
cpY84qV37ZC8wYicugidg9cFtD+1E0nVgOLXPkHnmc7lIDHFiWQKfOieH+KoVCbb
zdFu3rhW31ygphRmgszkHwApllCTBBMOqMaBpS8eHCnetOITvyB4Kiu1/nKvVxhY
exit11KQv8F3kTIUQRm0qw00TSBjuQHKoG83yfimlQ8OazciT+aLpVaY8SOrrNnL
IJ8dHgTpF9WWHxx04DDzqrT7Xq99F9RzDzM7dSizGxIxonoWcBjiF6n5
-----END CERTIFICATE-----`

const expiredLicenseToken = `eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiI5WGxaNTlIb3FfTXRkU25oIiwidGFnIjoiSng5SnJFWEpidThYTktBRTFkNF9odyIsInR5cCI6IkpXVCJ9.3aOzJ8CseHdknq4-5iyyVQ.Ajhfz-axov0_Fb64.0YE2hNz_KvgatHKB8hJCgemy09n8zJDc6haiFLkYGh9L96MXEhCUXg9V9iLioi311BtLT6RWXLuQspTNHLDvdIJLyPoNR3OvIYcHTz7kHhaX61lGutAEUBDdByPczoLVkkZccaKgIP8xho4XWmkjDMWXvhMXcTilN3cgeAEdQILXWL1pDPf-h0u-a7Esw5d0O8Ok1CBjFLrthgGnCVtMH5t7l3kBiWbzmAVo7Nz9Eegki0bmOqVSzBxmpDspNitbZFxzYWV23Km6Lmx_FWsEsTtx4nLyBARuxBQsf_l2UjqwowXUlK26Lw7Vqt6e8Upbw4sUrMjIZQzBbKwbAfPFm14QwgXmOfkcMwpeqz8v4oVml3WDIK4Ree6K-Z-ae-cMRGGCTHdp6XDidwykYAQXYC4pbdm-Hm86qO6AYODP_v8lvorXJQgfC0L4Mf5_7uM3daYxIa_80ZlNF9Ffa4YPsB4CuJFbHEEhSStDUlxCNTh5W1SnhgYgelVttnwaYCCVHlyqP4vCCGYgQIkoy01RKCq5dqXl2JPqpUt1bJZ_ywDlhi1xTKrO4uA6qfvKR_tNC1eYPrNmAR7sXMTj8gbUpklvh00edn-sHaR0yTj7ShMbAkK0o9WKUmElsMa_cpjTQ7dVEw6E1hoxjIdEI9kL87ex8uPRQ5383Df-NxO8I093Ef1RXVROeQp3Sass38ewkBuAM32AHUNfY8eP3aaw1ntGzeh93sa015Ob158t5W4ExsVp25RvM0RaV7UBhX0rkbCIVclJR87PkoSAfxtH5E1pkyBJB7rwGHKhWo0kO7U0QFLhAE2_l77pLME_QaHXLogUdLgGbloH2igxFLzboNfTs2yTc2JHeJDiPZDJBs-hbOEdJDD_JX_BcSWw_ZKFxeqA36RZl8LHvXOIS0C4LXmG9qAJvIabIlSIkVRNoSPWL8iXfCwkGHLl3uFc0_0USnunVIELwtEiaf2RUUv2-W1oHBBkrmkW2vxtwMB7GMItUs4l2oR024Qgqm4w9aIVBHvpz9f9QBKcUiWOyMvrqwCLUPDApQdU9bETwEngZOYtSZdX5G5qU-WbpVH91Y7ta81mJEm7Dtj55S5Vyx0NXyeO49M.BWNb4Ddh5iAoVq9eA8Vo_w`

func getExpiredLicenseKey(name string) *v3.LicenseKey {
	expiredLicenseKey := &v3.LicenseKey{ObjectMeta: metav1.ObjectMeta{Name: name}}

	expiredLicenseKey.Spec.Certificate = expiredLicenseCertificate
	expiredLicenseKey.Spec.Token = expiredLicenseToken

	return expiredLicenseKey
}

// TestGlobalAlertClient exercises the GlobalAlert client.
func TestGlobalAlertClient(t *testing.T) {
	const name = "test-globalalert"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.GlobalAlert{}
			})
			defer shutdownServer()
			if err := testGlobalAlertClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-globalalert test failed")
	}
}

func testGlobalAlertClient(client calicoclient.Interface, name string) error {
	globalAlertClient := client.ProjectcalicoV3().GlobalAlerts()
	globalAlert := &v3.GlobalAlert{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.GlobalAlertSpec{
			DataSet:     "dns",
			Description: "test",
			Severity:    100,
		},
		Status: calico.GlobalAlertStatus{
			LastUpdate:     &v1.Time{Time: time.Now()},
			Active:         false,
			Healthy:        false,
			ExecutionState: "test",
			LastExecuted:   &v1.Time{Time: time.Now()},
			LastEvent:      &v1.Time{Time: time.Now()},
			ErrorConditions: []calico.ErrorCondition{
				{Type: "foo", Message: "bar"},
			},
		},
	}

	// start from scratch
	globalAlerts, err := globalAlertClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalAlerts (%s)", err)
	}
	if globalAlerts.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalAlertServer, err := globalAlertClient.Create(globalAlert)
	if nil != err {
		return fmt.Errorf("error creating the globalAlert '%v' (%v)", globalAlert, err)
	}
	if name != globalAlertServer.Name {
		return fmt.Errorf("didn't get the same globalAlert back from the server \n%+v\n%+v", globalAlert, globalAlertServer)
	}
	if !reflect.DeepEqual(globalAlertServer.Status, calico.GlobalAlertStatus{}) {
		return fmt.Errorf("status was set on create to %#v", globalAlertServer.Status)
	}

	globalAlerts, err = globalAlertClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalAlerts (%s)", err)
	}
	if len(globalAlerts.Items) != 1 {
		return fmt.Errorf("expected 1 globalAlert got %d", len(globalAlerts.Items))
	}

	globalAlertServer, err = globalAlertClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalAlert %s (%s)", name, err)
	}
	if name != globalAlertServer.Name &&
		globalAlert.ResourceVersion == globalAlertServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalAlert back from the server \n%+v\n%+v", globalAlert, globalAlertServer)
	}

	globalAlertUpdate := globalAlertServer.DeepCopy()
	globalAlertUpdate.Spec.Metric = "count"
	globalAlertUpdate.Status.LastUpdate = &v1.Time{Time: time.Now()}
	globalAlertServer, err = globalAlertClient.Update(globalAlertUpdate)
	if err != nil {
		return fmt.Errorf("error updating globalAlert %s (%s)", name, err)
	}
	if globalAlertServer.Spec.Metric != globalAlertUpdate.Spec.Metric {
		return errors.New("didn't update spec.content")
	}
	if globalAlertServer.Status.LastUpdate != nil {
		return errors.New("status was updated by Update()")
	}

	globalAlertUpdate = globalAlertServer.DeepCopy()
	globalAlertUpdate.Status.LastUpdate = &v1.Time{Time: time.Now()}
	globalAlertUpdate.Labels = map[string]string{"foo": "bar"}
	globalAlertUpdate.Spec.Metric = ""
	globalAlertServer, err = globalAlertClient.UpdateStatus(globalAlertUpdate)
	if err != nil {
		return fmt.Errorf("error updating globalAlert %s (%s)", name, err)
	}
	if globalAlertServer.Status.LastUpdate == nil {
		return fmt.Errorf("didn't update status. %v != %v", globalAlertUpdate.Status, globalAlertServer.Status)
	}
	if _, ok := globalAlertServer.Labels["foo"]; ok {
		return fmt.Errorf("updatestatus updated labels")
	}
	if globalAlertServer.Spec.Metric == "" {
		return fmt.Errorf("updatestatus updated spec")
	}

	err = globalAlertClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalAlert should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().GlobalAlerts().Watch(v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching GlobalAlerts (%s)", err)
	}
	var events []watch.Event
	done := sync.WaitGroup{}
	done.Add(1)
	timeout := time.After(500 * time.Millisecond)
	var timeoutErr error
	// watch for 2 events
	go func() {
		defer done.Done()
		for i := 0; i < 2; i++ {
			select {
			case e := <-w.ResultChan():
				events = append(events, e)
			case <-timeout:
				timeoutErr = fmt.Errorf("timed out wating for events")
				return
			}
		}
		return
	}()

	// Create two GlobalAlerts
	for i := 0; i < 2; i++ {
		ga := &v3.GlobalAlert{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ga%d", i)},
			Spec: calico.GlobalAlertSpec{
				Description: "test",
				Severity:    100,
				DataSet:     "dns",
			},
		}
		_, err = globalAlertClient.Create(ga)
		if err != nil {
			return fmt.Errorf("error creating the globalAlert '%v' (%v)", ga, err)
		}
	}
	done.Wait()
	if timeoutErr != nil {
		return timeoutErr
	}
	if len(events) != 2 {
		return fmt.Errorf("expected 2 watch events got %d", len(events))
	}

	return nil
}

// TestGlobalAlertTemplateClient exercises the GlobalAlertTemplate client.
func TestGlobalAlertTemplateClient(t *testing.T) {
	const name = "test-globalalert"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.GlobalAlertTemplate{}
			})
			defer shutdownServer()
			if err := testGlobalAlertTemplateClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-globalalert test failed")
	}
}

func testGlobalAlertTemplateClient(client calicoclient.Interface, name string) error {
	globalAlertClient := client.ProjectcalicoV3().GlobalAlertTemplates()
	globalAlert := &v3.GlobalAlertTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.GlobalAlertSpec{
			DataSet:     "dns",
			Description: "test",
			Severity:    100,
		},
	}

	// start from scratch
	globalAlerts, err := globalAlertClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalAlertTemplates (%s)", err)
	}
	if globalAlerts.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalAlertServer, err := globalAlertClient.Create(globalAlert)
	if nil != err {
		return fmt.Errorf("error creating the globalAlertTemplate '%v' (%v)", globalAlert, err)
	}
	if name != globalAlertServer.Name {
		return fmt.Errorf("didn't get the same globalAlertTemplate back from the server \n%+v\n%+v", globalAlert, globalAlertServer)
	}

	globalAlerts, err = globalAlertClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalAlertTemplates (%s)", err)
	}
	if len(globalAlerts.Items) != 1 {
		return fmt.Errorf("expected 1 globalAlertTemplate got %d", len(globalAlerts.Items))
	}

	globalAlertServer, err = globalAlertClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalAlertTemplate %s (%s)", name, err)
	}
	if name != globalAlertServer.Name &&
		globalAlert.ResourceVersion == globalAlertServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalAlertTemplate back from the server \n%+v\n%+v", globalAlert, globalAlertServer)
	}

	globalAlertUpdate := globalAlertServer.DeepCopy()
	globalAlertUpdate.Spec.Metric = "count"
	globalAlertServer, err = globalAlertClient.Update(globalAlertUpdate)
	if err != nil {
		return fmt.Errorf("error updating globalAlertTemplate %s (%s)", name, err)
	}
	if globalAlertServer.Spec.Metric != globalAlertUpdate.Spec.Metric {
		return errors.New("didn't update spec.content")
	}

	err = globalAlertClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalAlertTemplate should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().GlobalAlertTemplates().Watch(v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching GlobalAlertTemplates (%s)", err)
	}
	var events []watch.Event
	done := sync.WaitGroup{}
	done.Add(1)
	timeout := time.After(500 * time.Millisecond)
	var timeoutErr error
	// watch for 2 events
	go func() {
		defer done.Done()
		for i := 0; i < 2; i++ {
			select {
			case e := <-w.ResultChan():
				events = append(events, e)
			case <-timeout:
				timeoutErr = fmt.Errorf("timed out wating for events")
				return
			}
		}
		return
	}()

	// Create two GlobalAlertTemplates
	for i := 0; i < 2; i++ {
		ga := &v3.GlobalAlertTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ga%d", i)},
			Spec: calico.GlobalAlertSpec{
				Description: "test",
				Severity:    100,
				DataSet:     "dns",
			},
		}
		_, err = globalAlertClient.Create(ga)
		if err != nil {
			return fmt.Errorf("error creating the globalAlertTemplate '%v' (%v)", ga, err)
		}
	}
	done.Wait()
	if timeoutErr != nil {
		return timeoutErr
	}
	if len(events) != 2 {
		return fmt.Errorf("expected 2 watch events got %d", len(events))
	}

	return nil
}

// TestGlobalThreatFeedClient exercises the GlobalThreatFeed client.
func TestGlobalThreatFeedClient(t *testing.T) {
	const name = "test-globalthreatfeed"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.GlobalThreatFeed{}
			})
			defer shutdownServer()
			if err := testGlobalThreatFeedClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-globalthreatfeed test failed")
	}
}

func testGlobalThreatFeedClient(client calicoclient.Interface, name string) error {
	globalThreatFeedClient := client.ProjectcalicoV3().GlobalThreatFeeds()
	globalThreatFeed := &v3.GlobalThreatFeed{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: calico.GlobalThreatFeedStatus{
			LastSuccessfulSync:   metav1.Time{time.Now()},
			LastSuccessfulSearch: metav1.Time{time.Now()},
			ErrorConditions: []calico.ErrorCondition{
				{
					Type:    "foo",
					Message: "bar",
				},
			},
		},
	}

	// start from scratch
	globalThreatFeeds, err := globalThreatFeedClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalThreatFeeds (%s)", err)
	}
	if globalThreatFeeds.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalThreatFeedServer, err := globalThreatFeedClient.Create(globalThreatFeed)
	if nil != err {
		return fmt.Errorf("error creating the globalThreatFeed '%v' (%v)", globalThreatFeed, err)
	}
	if name != globalThreatFeedServer.Name {
		return fmt.Errorf("didn't get the same globalThreatFeed back from the server \n%+v\n%+v", globalThreatFeed, globalThreatFeedServer)
	}
	if !reflect.DeepEqual(globalThreatFeedServer.Status, calico.GlobalThreatFeedStatus{}) {
		return fmt.Errorf("status was set on create to %#v", globalThreatFeedServer.Status)
	}

	globalThreatFeeds, err = globalThreatFeedClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalThreatFeeds (%s)", err)
	}
	if len(globalThreatFeeds.Items) != 1 {
		return fmt.Errorf("expected 1 globalThreatFeed got %d", len(globalThreatFeeds.Items))
	}

	globalThreatFeedServer, err = globalThreatFeedClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalThreatFeed %s (%s)", name, err)
	}
	if name != globalThreatFeedServer.Name &&
		globalThreatFeed.ResourceVersion == globalThreatFeedServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalThreatFeed back from the server \n%+v\n%+v", globalThreatFeed, globalThreatFeedServer)
	}

	globalThreatFeedUpdate := globalThreatFeedServer.DeepCopy()
	globalThreatFeedUpdate.Spec.Content = "IPSet"
	globalThreatFeedUpdate.Status.LastSuccessfulSync = v1.Time{Time: time.Now()}
	globalThreatFeedServer, err = globalThreatFeedClient.Update(globalThreatFeedUpdate)
	if err != nil {
		return fmt.Errorf("error updating globalThreatFeed %s (%s)", name, err)
	}
	if globalThreatFeedServer.Spec.Content != globalThreatFeedUpdate.Spec.Content {
		return errors.New("didn't update spec.content")
	}
	if !globalThreatFeedServer.Status.LastSuccessfulSync.Time.Equal(time.Time{}) {
		return errors.New("status was updated by Update()")
	}

	globalThreatFeedUpdate = globalThreatFeedServer.DeepCopy()
	globalThreatFeedUpdate.Status.LastSuccessfulSync = v1.Time{Time: time.Now()}
	globalThreatFeedUpdate.Labels = map[string]string{"foo": "bar"}
	globalThreatFeedUpdate.Spec.Content = ""
	globalThreatFeedServer, err = globalThreatFeedClient.UpdateStatus(globalThreatFeedUpdate)
	if err != nil {
		return fmt.Errorf("error updating globalThreatFeed %s (%s)", name, err)
	}
	if globalThreatFeedServer.Status.LastSuccessfulSync.Time.Equal(time.Time{}) {
		return fmt.Errorf("didn't update status. %v != %v", globalThreatFeedUpdate.Status, globalThreatFeedServer.Status)
	}
	if _, ok := globalThreatFeedServer.Labels["foo"]; ok {
		return fmt.Errorf("updatestatus updated labels")
	}
	if globalThreatFeedServer.Spec.Content == "" {
		return fmt.Errorf("updatestatus updated spec")
	}

	err = globalThreatFeedClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalThreatFeed should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().GlobalThreatFeeds().Watch(v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching GlobalThreatFeeds (%s)", err)
	}
	var events []watch.Event
	done := sync.WaitGroup{}
	done.Add(1)
	timeout := time.After(500 * time.Millisecond)
	var timeoutErr error
	// watch for 2 events
	go func() {
		defer done.Done()
		for i := 0; i < 2; i++ {
			select {
			case e := <-w.ResultChan():
				events = append(events, e)
			case <-timeout:
				timeoutErr = fmt.Errorf("timed out wating for events")
				return
			}
		}
		return
	}()

	// Create two GlobalThreatFeeds
	for i := 0; i < 2; i++ {
		gtf := &v3.GlobalThreatFeed{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("gtf%d", i)},
		}
		_, err = globalThreatFeedClient.Create(gtf)
		if err != nil {
			return fmt.Errorf("error creating the globalThreatFeed '%v' (%v)", gtf, err)
		}
	}
	done.Wait()
	if timeoutErr != nil {
		return timeoutErr
	}
	if len(events) != 2 {
		return fmt.Errorf("expected 2 watch events got %d", len(events))
	}

	return nil
}

// TestHostEndpointClient exercises the HostEndpoint client.
func TestHostEndpointClient(t *testing.T) {
	const name = "test-hostendpoint"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.HostEndpoint{}
			})
			defer shutdownServer()
			if err := testHostEndpointClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-hostendpoint test failed")
	}
}

func createTestHostEndpoint(name string, ip string, node string) *v3.HostEndpoint {
	hostEndpoint := &v3.HostEndpoint{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	hostEndpoint.Spec.ExpectedIPs = []string{ip}
	hostEndpoint.Spec.Node = node

	return hostEndpoint
}

func testHostEndpointClient(client calicoclient.Interface, name string) error {
	hostEndpointClient := client.ProjectcalicoV3().HostEndpoints()

	hostEndpoint := createTestHostEndpoint(name, "192.168.0.1", "test-node")

	// start from scratch
	hostEndpoints, err := hostEndpointClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing hostEndpoints (%s)", err)
	}
	if hostEndpoints.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	hostEndpointServer, err := hostEndpointClient.Create(hostEndpoint)
	if nil != err {
		return fmt.Errorf("error creating the hostEndpoint '%v' (%v)", hostEndpoint, err)
	}
	if name != hostEndpointServer.Name {
		return fmt.Errorf("didn't get the same hostEndpoint back from the server \n%+v\n%+v", hostEndpoint, hostEndpointServer)
	}

	hostEndpoints, err = hostEndpointClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing hostEndpoints (%s)", err)
	}
	if len(hostEndpoints.Items) != 1 {
		return fmt.Errorf("expected 1 hostEndpoint entry, got %d", len(hostEndpoints.Items))
	}

	hostEndpointServer, err = hostEndpointClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting hostEndpoint %s (%s)", name, err)
	}
	if name != hostEndpointServer.Name &&
		hostEndpoint.ResourceVersion == hostEndpointServer.ResourceVersion {
		return fmt.Errorf("didn't get the same hostEndpoint back from the server \n%+v\n%+v", hostEndpoint, hostEndpointServer)
	}

	err = hostEndpointClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("hostEndpoint should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().HostEndpoints().Watch(v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching HostEndpoints (%s)", err)
	}
	var events []watch.Event
	done := sync.WaitGroup{}
	done.Add(1)
	timeout := time.After(500 * time.Millisecond)
	var timeoutErr error
	// watch for 2 events
	go func() {
		defer done.Done()
		for i := 0; i < 2; i++ {
			select {
			case e := <-w.ResultChan():
				events = append(events, e)
			case <-timeout:
				timeoutErr = fmt.Errorf("timed out wating for events")
				return
			}
		}
		return
	}()

	// Create two HostEndpoints
	for i := 0; i < 2; i++ {
		hep := createTestHostEndpoint(fmt.Sprintf("hep%d", i), "192.168.0.1", "test-node")
		_, err = hostEndpointClient.Create(hep)
		if err != nil {
			return fmt.Errorf("error creating hostEndpoint '%v' (%v)", hep, err)
		}
	}

	done.Wait()
	if timeoutErr != nil {
		return timeoutErr
	}
	if len(events) != 2 {
		return fmt.Errorf("expected 2 watch events got %d", len(events))
	}

	return nil
}

// TestGlobalReportClient exercises the GlobalReport client.
func TestGlobalReportClient(t *testing.T) {
	const name = "test-global-report"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.GlobalReport{}
			})
			defer shutdownServer()
			if err := testGlobalReportClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("GlobalReport test failed")
	}
}

func testGlobalReportClient(client calicoclient.Interface, name string) error {
	globalReportTypeName := "inventory"
	globalReportClient := client.ProjectcalicoV3().GlobalReports()
	globalReport := &v3.GlobalReport{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.ReportSpec{
			ReportType: globalReportTypeName,
		},
		Status: calico.ReportStatus{
			LastSuccessfulReportJobs: []calico.CompletedReportJob{
				{
					ReportJob: calico.ReportJob{
						Start: metav1.Time{time.Now()},
						End:   metav1.Time{time.Now()},
						Job: &corev1.ObjectReference{
							Kind:      "NetworkPolicy",
							Name:      "fbar-srj",
							Namespace: "fbar-ns-srj",
						},
					},
					JobCompletionTime: &metav1.Time{time.Now()},
				},
			},
			LastFailedReportJobs: []calico.CompletedReportJob{
				{
					ReportJob: calico.ReportJob{
						Start: metav1.Time{time.Now()},
						End:   metav1.Time{time.Now()},
						Job: &corev1.ObjectReference{
							Kind:      "NetworkPolicy",
							Name:      "fbar-frj",
							Namespace: "fbar-ns-frj",
						},
					},
					JobCompletionTime: &metav1.Time{time.Now()},
				},
			},
			ActiveReportJobs: []calico.ReportJob{
				{
					Start: metav1.Time{time.Now()},
					End:   metav1.Time{time.Now()},
					Job: &corev1.ObjectReference{
						Kind:      "NetworkPolicy",
						Name:      "fbar-arj",
						Namespace: "fbar-ns-arj",
					},
				},
			},
			LastScheduledReportJob: &calico.ReportJob{
				Start: metav1.Time{time.Now()},
				End:   metav1.Time{time.Now()},
				Job: &corev1.ObjectReference{
					Kind:      "NetworkPolicy",
					Name:      "fbar-lsj",
					Namespace: "fbar-ns-lsj",
				},
			},
		},
	}

	// Make sure there is no GlobalReport configured.
	globalReports, err := globalReportClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalReports (%s)", err)
	}
	if globalReports.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	// Create/List/Get/Delete tests.

	// We now need a GlobalReportType resource before GlobalReport can be created.
	globalReportTypeClient := client.ProjectcalicoV3().GlobalReportTypes()
	globalReportType := &v3.GlobalReportType{
		ObjectMeta: metav1.ObjectMeta{Name: globalReportTypeName},
		Spec: calico.ReportTypeSpec{
			UISummaryTemplate: calico.ReportTemplate{
				Name:     "uist",
				Template: "Report Name: {{ .ReportName }}",
			},
		},
	}
	_, err = globalReportTypeClient.Create(globalReportType)
	if nil != err {
		return fmt.Errorf("error creating the pre-requisite globalReportType '%v' (%v)", globalReportType, err)
	}

	globalReportServer, err := globalReportClient.Create(globalReport)
	if nil != err {
		return fmt.Errorf("error creating the globalReport '%v' (%v)", globalReport, err)
	}
	if name != globalReportServer.Name {
		return fmt.Errorf("didn't get the same globalReport back from the server \n%+v\n%+v", globalReport, globalReportServer)
	}
	if !reflect.DeepEqual(globalReportServer.Status, calico.ReportStatus{}) {
		return fmt.Errorf("status was set on create to %#v", globalReportServer.Status)
	}

	globalReports, err = globalReportClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalReports (%s)", err)
	}
	if len(globalReports.Items) != 1 {
		return fmt.Errorf("expected 1 globalReport entry, got %d", len(globalReports.Items))
	}

	globalReportServer, err = globalReportClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalReport %s (%s)", name, err)
	}
	if name != globalReportServer.Name &&
		globalReport.ResourceVersion == globalReportServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalReport back from the server \n%+v\n%+v", globalReport, globalReportServer)
	}

	// Pupulate both GlobalReport and ReportStatus.
	// Verify that Update() modifies GlobalReport only.
	globalReportUpdate := globalReportServer.DeepCopy()
	globalReportUpdate.Spec.Schedule = "1 * * * *"
	globalReportUpdate.Status.LastSuccessfulReportJobs = []calico.CompletedReportJob{
		{JobCompletionTime: &v1.Time{Time: time.Now()}},
	}
	globalReportServer, err = globalReportClient.Update(globalReportUpdate)
	if err != nil {
		return fmt.Errorf("error updating globalReport %s (%s)", name, err)
	}
	if globalReportServer.Spec.Schedule != globalReportUpdate.Spec.Schedule {
		return errors.New("GlobalReport Update() didn't update Spec.Schedule")
	}
	if len(globalReportServer.Status.LastSuccessfulReportJobs) != 0 {
		return errors.New("GlobalReport status was updated by Update()")
	}

	// Pupulate both GlobalReport and ReportStatus.
	// Verify that UpdateStatus() modifies ReportStatus only.
	globalReportUpdate = globalReportServer.DeepCopy()
	globalReportUpdate.Status.LastSuccessfulReportJobs = []calico.CompletedReportJob{
		{JobCompletionTime: &v1.Time{Time: time.Now()}},
	}
	globalReportUpdate.Labels = map[string]string{"foo": "bar"}
	globalReportServer, err = globalReportClient.UpdateStatus(globalReportUpdate)
	if err != nil {
		return fmt.Errorf("error updating globalReport %s (%s)", name, err)
	}
	if len(globalReportServer.Status.LastSuccessfulReportJobs) == 0 ||
		globalReportServer.Status.LastSuccessfulReportJobs[0].JobCompletionTime == nil ||
		globalReportServer.Status.LastSuccessfulReportJobs[0].JobCompletionTime.Time.Equal(time.Time{}) {
		return fmt.Errorf("didn't update GlobalReport status. %v != %v", globalReportUpdate.Status, globalReportServer.Status)
	}
	if _, ok := globalReportServer.Labels["foo"]; ok {
		return fmt.Errorf("updatestatus updated labels")
	}

	err = globalReportClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalReport should be deleted (%s)", err)
	}

	// Check list-ing GlobalReport resource works with watch option.
	w, err := client.ProjectcalicoV3().GlobalReports().Watch(v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching GlobalReports (%s)", err)
	}
	var events []watch.Event
	done := sync.WaitGroup{}
	done.Add(1)
	timeout := time.After(500 * time.Millisecond)
	var timeoutErr error
	// watch for 2 events
	go func() {
		defer done.Done()
		for i := 0; i < 2; i++ {
			select {
			case e := <-w.ResultChan():
				events = append(events, e)
			case <-timeout:
				timeoutErr = fmt.Errorf("timed out wating for events")
				return
			}
		}
		return
	}()

	// Create two GlobalReports
	for i := 0; i < 2; i++ {
		gr := &v3.GlobalReport{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("gr%d", i)},
			Spec:       calico.ReportSpec{ReportType: "inventory"},
		}
		_, err = globalReportClient.Create(gr)
		if err != nil {
			return fmt.Errorf("error creating globalReport '%v' (%v)", gr, err)
		}
	}

	done.Wait()
	if timeoutErr != nil {
		return timeoutErr
	}
	if len(events) != 2 {
		return fmt.Errorf("expected 2 watch events got %d", len(events))
	}

	// Undo pre-requisite creating GlobalReportType.
	err = globalReportTypeClient.Delete(globalReportTypeName, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("error deleting the pre-requisite globalReportType '%v' (%v)", globalReportType, err)
	}

	return nil
}

// TestGlobalReportTypeClient exercises the GlobalReportType client.
func TestGlobalReportTypeClient(t *testing.T) {
	const name = "test-global-report-type"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.GlobalReportType{}
			})
			defer shutdownServer()
			if err := testGlobalReportTypeClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("GlobalReportType test failed")
	}
}

func testGlobalReportTypeClient(client calicoclient.Interface, name string) error {
	globalReportTypeClient := client.ProjectcalicoV3().GlobalReportTypes()
	globalReportType := &v3.GlobalReportType{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.ReportTypeSpec{
			UISummaryTemplate: calico.ReportTemplate{
				Name:     "uist",
				Template: "Report Name: {{ .ReportName }}",
			},
		},
	}

	// Make sure there is no GlobalReportType configured.
	globalReportTypes, err := globalReportTypeClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalReportTypes (%s)", err)
	}
	if globalReportTypes.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	// Create/List/Get/Delete tests.
	globalReportTypeServer, err := globalReportTypeClient.Create(globalReportType)
	if nil != err {
		return fmt.Errorf("error creating the globalReportType '%v' (%v)", globalReportType, err)
	}
	if name != globalReportTypeServer.Name {
		return fmt.Errorf("didn't get the same globalReportType back from the server \n%+v\n%+v", globalReportType, globalReportTypeServer)
	}

	globalReportTypes, err = globalReportTypeClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalReportTypes (%s)", err)
	}
	if len(globalReportTypes.Items) != 1 {
		return fmt.Errorf("expected 1 globalReportType entry, got %d", len(globalReportTypes.Items))
	}

	globalReportTypeServer, err = globalReportTypeClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalReportType %s (%s)", name, err)
	}
	if name != globalReportTypeServer.Name &&
		globalReportType.ResourceVersion == globalReportTypeServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalReportType back from the server \n%+v\n%+v", globalReportType, globalReportTypeServer)
	}

	err = globalReportTypeClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalReportType should be deleted (%s)", err)
	}

	// Check list-ing GlobalReportType resource works with watch option.
	w, err := client.ProjectcalicoV3().GlobalReportTypes().Watch(v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching GlobalReportTypes (%s)", err)
	}
	var events []watch.Event
	done := sync.WaitGroup{}
	done.Add(1)
	timeout := time.After(500 * time.Millisecond)
	var timeoutErr error
	// watch for 2 events
	go func() {
		defer done.Done()
		for i := 0; i < 2; i++ {
			select {
			case e := <-w.ResultChan():
				events = append(events, e)
			case <-timeout:
				timeoutErr = fmt.Errorf("timed out wating for events")
				return
			}
		}
		return
	}()

	// Create two GlobalReports
	for i := 0; i < 2; i++ {
		grt := &v3.GlobalReportType{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("grt%d", i)},
			Spec: calico.ReportTypeSpec{
				UISummaryTemplate: calico.ReportTemplate{
					Name:     fmt.Sprintf("uist%d", i),
					Template: "Report Name: {{ .ReportName }}",
				},
			},
		}
		_, err = globalReportTypeClient.Create(grt)
		if err != nil {
			return fmt.Errorf("error creating globalReportType '%v' (%v)", grt, err)
		}
	}

	done.Wait()
	if timeoutErr != nil {
		return timeoutErr
	}
	if len(events) != 2 {
		return fmt.Errorf("expected 2 watch events got %d", len(events))
	}

	return nil
}

// TestIPPoolClient exercises the IPPool client.
func TestIPPoolClient(t *testing.T) {
	const name = "test-ippool"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.IPPool{}
			})
			defer shutdownServer()
			if err := testIPPoolClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-ippool test failed")
	}
}

func testIPPoolClient(client calicoclient.Interface, name string) error {
	ippoolClient := client.ProjectcalicoV3().IPPools()
	ippool := &v3.IPPool{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.IPPoolSpec{
			CIDR: "192.168.0.0/16",
		},
	}

	// start from scratch
	ippools, err := ippoolClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ippools (%s)", err)
	}
	if ippools.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	ippoolServer, err := ippoolClient.Create(ippool)
	if nil != err {
		return fmt.Errorf("error creating the ippool '%v' (%v)", ippool, err)
	}
	if name != ippoolServer.Name {
		return fmt.Errorf("didn't get the same ippool back from the server \n%+v\n%+v", ippool, ippoolServer)
	}

	ippools, err = ippoolClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ippools (%s)", err)
	}

	ippoolServer, err = ippoolClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting ippool %s (%s)", name, err)
	}
	if name != ippoolServer.Name &&
		ippool.ResourceVersion == ippoolServer.ResourceVersion {
		return fmt.Errorf("didn't get the same ippool back from the server \n%+v\n%+v", ippool, ippoolServer)
	}

	err = ippoolClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("ippool should be deleted (%s)", err)
	}

	return nil
}

// TestBGPConfigurationClient exercises the BGPConfiguration client.
func TestBGPConfigurationClient(t *testing.T) {
	const name = "test-bgpconfig"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.BGPConfiguration{}
			})
			defer shutdownServer()
			if err := testBGPConfigurationClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-bgpconfig test failed")
	}
}

func testBGPConfigurationClient(client calicoclient.Interface, name string) error {
	bgpConfigClient := client.ProjectcalicoV3().BGPConfigurations()
	resName := "bgpconfig-test"
	bgpConfig := &v3.BGPConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: resName},
		Spec: calico.BGPConfigurationSpec{
			LogSeverityScreen: "Info",
		},
	}

	// start from scratch
	bgpConfigList, err := bgpConfigClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing bgpConfiguration (%s)", err)
	}
	if bgpConfigList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	bgpRes, err := bgpConfigClient.Create(bgpConfig)
	if nil != err {
		return fmt.Errorf("error creating the bgpConfiguration '%v' (%v)", bgpConfig, err)
	}
	if resName != bgpRes.Name {
		return fmt.Errorf("didn't get the same bgpConfig back from server\n%+v\n%+v", bgpConfig, bgpRes)
	}

	bgpRes, err = bgpConfigClient.Get(resName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting bgpConfiguration %s (%s)", resName, err)
	}

	err = bgpConfigClient.Delete(resName, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("BGPConfiguration should be deleted (%s)", err)
	}

	return nil
}

// TestBGPPeerClient exercises the BGPPeer client.
func TestBGPPeerClient(t *testing.T) {
	const name = "test-bgppeer"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.BGPPeer{}
			})
			defer shutdownServer()
			if err := testBGPPeerClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-bgppeer test failed")
	}
}

func testBGPPeerClient(client calicoclient.Interface, name string) error {
	bgpPeerClient := client.ProjectcalicoV3().BGPPeers()
	resName := "bgppeer-test"
	bgpPeer := &v3.BGPPeer{
		ObjectMeta: metav1.ObjectMeta{Name: resName},
		Spec: calico.BGPPeerSpec{
			Node:     "node1",
			PeerIP:   "10.0.0.1",
			ASNumber: numorstring.ASNumber(6512),
		},
	}

	// start from scratch
	bgpPeerList, err := bgpPeerClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing bgpPeer (%s)", err)
	}
	if bgpPeerList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	bgpRes, err := bgpPeerClient.Create(bgpPeer)
	if nil != err {
		return fmt.Errorf("error creating the bgpPeer '%v' (%v)", bgpPeer, err)
	}
	if resName != bgpRes.Name {
		return fmt.Errorf("didn't get the same bgpPeer back from server\n%+v\n%+v", bgpPeer, bgpRes)
	}

	bgpRes, err = bgpPeerClient.Get(resName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting bgpPeer %s (%s)", resName, err)
	}

	err = bgpPeerClient.Delete(resName, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("BGPPeer should be deleted (%s)", err)
	}

	return nil
}

// TestProfileClient exercises the Profile client.
func TestProfileClient(t *testing.T) {
	const name = "test-profile"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.Profile{}
			})
			defer shutdownServer()
			if err := testProfileClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-profile test failed")
	}
}

func testProfileClient(client calicoclient.Interface, name string) error {
	profileClient := client.ProjectcalicoV3().Profiles()
	resName := "profile-test"
	profile := &v3.Profile{
		ObjectMeta: metav1.ObjectMeta{Name: resName},
		Spec: calico.ProfileSpec{
			LabelsToApply: map[string]string{
				"aa": "bb",
			},
		},
	}

	// start from scratch
	profileList, err := profileClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing profile (%s)", err)
	}
	if profileList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	profileRes, err := profileClient.Create(profile)
	if nil != err {
		return fmt.Errorf("error creating the profile '%v' (%v)", profile, err)
	}
	if resName != profileRes.Name {
		return fmt.Errorf("didn't get the same profile back from server\n%+v\n%+v", profile, profileRes)
	}

	profileRes, err = profileClient.Get(resName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting profile %s (%s)", resName, err)
	}

	err = profileClient.Delete(resName, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("Profile should be deleted (%s)", err)
	}

	return nil
}

// TestRemoteClusterConfigurationClient exercises the RemoteClusterConfiguration client.
func TestRemoteClusterConfigurationClient(t *testing.T) {
	const name = "test-remoteclusterconfig"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.RemoteClusterConfiguration{}
			})
			defer shutdownServer()
			if err := testRemoteClusterConfigurationClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-remoteclusterconfig test failed")

	}
}

func testRemoteClusterConfigurationClient(client calicoclient.Interface, name string) error {
	rccClient := client.ProjectcalicoV3().RemoteClusterConfigurations()
	resName := "rcc-test"
	rcc := &v3.RemoteClusterConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: resName},
		Spec: calico.RemoteClusterConfigurationSpec{
			DatastoreType: "etcdv3",
			EtcdConfig: calico.EtcdConfig{
				EtcdEndpoints: "https://127.0.0.1:999",
				EtcdUsername:  "user",
				EtcdPassword:  "abc123",
			},
		},
	}

	// start from scratch
	rccList, err := rccClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing remoteClusterConfiguration (%s)", err)
	}
	if rccList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	rccRes, err := rccClient.Create(rcc)
	if nil != err {
		return fmt.Errorf("error creating the remoteClusterConfiguration '%v' (%v)", rcc, err)
	}
	if resName != rccRes.Name {
		return fmt.Errorf("didn't get the same remoteClusterConfiguration back from server\n%+v\n%+v", rcc, rccRes)
	}

	rccRes, err = rccClient.Get(resName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting remoteClusterConfiguration %s (%s)", resName, err)
	}

	err = rccClient.Delete(resName, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("RemoteClusterConfiguration should be deleted (%s)", err)
	}

	return nil
}

// TestFelixConfigurationClient exercises the FelixConfiguration client.
func TestFelixConfigurationClient(t *testing.T) {
	const name = "test-felixconfig"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.FelixConfiguration{}
			})
			defer shutdownServer()
			if err := testFelixConfigurationClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-felixConfig test failed")
	}
}

func testFelixConfigurationClient(client calicoclient.Interface, name string) error {
	felixConfigClient := client.ProjectcalicoV3().FelixConfigurations()
	ptrTrue := true
	ptrInt := 1432
	felixConfig := &v3.FelixConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.FelixConfigurationSpec{
			UseInternalDataplaneDriver: &ptrTrue,
			DataplaneDriver:            "test-dataplane-driver",
			MetadataPort:               &ptrInt,
		},
	}

	// start from scratch
	felixConfigs, err := felixConfigClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing felixConfigs (%s)", err)
	}
	if felixConfigs.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	felixConfigServer, err := felixConfigClient.Create(felixConfig)
	if nil != err {
		return fmt.Errorf("error creating the felixConfig '%v' (%v)", felixConfig, err)
	}
	if name != felixConfigServer.Name {
		return fmt.Errorf("didn't get the same felixConfig back from the server \n%+v\n%+v", felixConfig, felixConfigServer)
	}

	felixConfigs, err = felixConfigClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing felixConfigs (%s)", err)
	}

	felixConfigServer, err = felixConfigClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting felixConfig %s (%s)", name, err)
	}
	if name != felixConfigServer.Name &&
		felixConfig.ResourceVersion == felixConfigServer.ResourceVersion {
		return fmt.Errorf("didn't get the same felixConfig back from the server \n%+v\n%+v", felixConfig, felixConfigServer)
	}

	err = felixConfigClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("felixConfig should be deleted (%s)", err)
	}

	return nil
}

// TestManagedClusterClient exercises the ManagedCluster client.
func TestManagedClusterClient(t *testing.T) {
	const name = "test-managedcluster"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.ManagedCluster{}
			})
			defer shutdownServer()
			if err := testManagedClusterClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-managedcluster test failed")
	}
}

func testManagedClusterClient(client calicoclient.Interface, name string) error {
	managedClusterClient := client.ProjectcalicoV3().ManagedClusters()
	managedCluster := &v3.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.ManagedClusterSpec{
			InstallationManifest: "manifest v1",
		},
	}

	// start from scratch
	managedClusters, err := managedClusterClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing managedClusters (%s)", err)
	}
	if managedClusters.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	// ------------------------------------------------------------------------------------------
	managedClusterServer, err := managedClusterClient.Create(managedCluster)
	if nil != err {
		return fmt.Errorf("error creating the managedCluster '%v' (%v)", managedCluster, err)
	}
	if name != managedClusterServer.Name {
		return fmt.Errorf("didn't get the same managedCluster back from the server \n%+v\n%+v", managedCluster, managedClusterServer)
	}

	// ------------------------------------------------------------------------------------------
	managedClusters, err = managedClusterClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing managedClusters (%s)", err)
	}
	if len(managedClusters.Items) != 1 {
		return fmt.Errorf("expected 1 managedCluster got %d", len(managedClusters.Items))
	}

	// ------------------------------------------------------------------------------------------
	managedClusterServer, err = managedClusterClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting managedCluster %s (%s)", name, err)
	}
	if name != managedClusterServer.Name &&
		managedCluster.ResourceVersion == managedClusterServer.ResourceVersion {
		return fmt.Errorf("didn't get the same managedCluster back from the server \n%+v\n%+v", managedCluster, managedClusterServer)
	}

	// ------------------------------------------------------------------------------------------
	managedClusterUpdate := managedClusterServer.DeepCopy()
	managedClusterUpdate.Spec.InstallationManifest = "manifest v2"
	managedClusterServer, err = managedClusterClient.Update(managedClusterUpdate)
	if err != nil {
		return fmt.Errorf("error updating managedCluster %s (%s)", name, err)
	}
	if managedClusterServer.Spec.InstallationManifest != managedClusterUpdate.Spec.InstallationManifest {
		return errors.New("didn't update spec.installationManifest")
	}

	// ------------------------------------------------------------------------------------------
	err = managedClusterClient.Delete(name, &metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("managedCluster should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().ManagedClusters().Watch(v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching ManagedClusters (%s)", err)
	}
	var events []watch.Event
	done := sync.WaitGroup{}
	done.Add(1)
	timeout := time.After(500 * time.Millisecond)
	var timeoutErr error
	// watch for 2 events
	go func() {
		defer done.Done()
		for i := 0; i < 2; i++ {
			select {
			case e := <-w.ResultChan():
				events = append(events, e)
			case <-timeout:
				timeoutErr = fmt.Errorf("timed out wating for events")
				return
			}
		}
		return
	}()

	// Create two ManagedClusters
	for i := 0; i < 2; i++ {
		mc := &v3.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("mc%d", i)},
		}
		_, err = managedClusterClient.Create(mc)
		if err != nil {
			return fmt.Errorf("error creating the managedCluster '%v' (%v)", mc, err)
		}
	}
	done.Wait()
	if timeoutErr != nil {
		return timeoutErr
	}
	if len(events) != 2 {
		return fmt.Errorf("expected 2 watch events got %d", len(events))
	}

	return nil
}

// TestClusterInformationClient exercises the ClusterInformation client.
func TestClusterInformationClient(t *testing.T) {
	const name = "default"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &projectcalico.ClusterInformation{}
			})
			defer shutdownServer()
			if err := testClusterInformationClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-clusterinformation test failed")
	}
}

func testClusterInformationClient(client calicoclient.Interface, name string) error {
	clusterInformationClient := client.ProjectcalicoV3().ClusterInformations()

	ci, err := clusterInformationClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ClusterInformation (%s)", err)
	}
	if ci.Items == nil {
		return fmt.Errorf("items field should not be set to nil")
	}

	// Confirm it's not possible to create a clusterInformation obj with name other than "default"
	validClusterInfo := &v3.ClusterInformation{ObjectMeta: metav1.ObjectMeta{Name: "test-clusterinformation"}}

	_, err = clusterInformationClient.Create(validClusterInfo)
	if err == nil {
		return fmt.Errorf("expected error creating validClusterInfo with name other than \"default\"")
	}

	return nil
}
