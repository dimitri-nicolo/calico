// Copyright (c) 2017-2022 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package util

package integration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	calico "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"github.com/tigera/api/pkg/lib/numorstring"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/calico/apiserver/pkg/apiserver"
	"github.com/projectcalico/calico/apiserver/pkg/registry/projectcalico/authenticationreview"
	"github.com/projectcalico/calico/apiserver/pkg/registry/projectcalico/authorizationreview"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	libapiv3 "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	libclient "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	licclient "github.com/projectcalico/calico/licensing/client"
	licFeatures "github.com/projectcalico/calico/licensing/client/features"
	"github.com/projectcalico/calico/licensing/utils"
)

// TestGroupVersion is trivial.
func TestGroupVersion(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.NetworkPolicy{}
			}, true)
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
	if gv.Group != v3.GroupName {
		return fmt.Errorf("we should be testing the servicecatalog group, not %s", gv.Group)
	}
	return nil
}

func TestEtcdHealthCheckerSuccess(t *testing.T) {
	serverConfig := NewTestServerConfig()
	_, _, clientconfig, shutdownServer := withConfigGetFreshApiserverServerAndClient(t, serverConfig)
	t.Log(clientconfig.Host)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &http.Client{Transport: tr}
	var success bool
	var resp *http.Response
	var err error
	retryInterval := 500 * time.Millisecond
	for i := 0; i < 5; i++ {
		resp, err = c.Get(clientconfig.Host + "/healthz")
		if nil != err || http.StatusOK != resp.StatusCode {
			success = false
			time.Sleep(retryInterval)
		} else {
			success = true
			break
		}
	}

	if !success {
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
				return &v3.NetworkPolicy{}
			}, true)
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

	if p, e := cClient.NetworkPolicies(ns).Create(context.Background(), &v3.NetworkPolicy{}, metav1.CreateOptions{}); nil == e {
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
				return &v3.NetworkPolicy{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	policies, err := policyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if policies.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(policies.Items) > 0 {
		return fmt.Errorf("policies should not exist on start, had %v policies", len(policies.Items))
	}

	policyServer, err := policyClient.Create(ctx, policy, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", policy, err)
	}
	if defaultTierPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	updatedPolicy := policyServer
	updatedPolicy.Labels = map[string]string{"foo": "bar"}
	policyServer, err = policyClient.Update(ctx, updatedPolicy, metav1.UpdateOptions{})
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

	tierClient.Create(ctx, tier, metav1.CreateOptions{})
	defer func() {
		tierClient.Delete(ctx, "net-sec", metav1.DeleteOptions{})
	}()

	netSecPolicyName := "net-sec" + "." + name
	netSecPolicy := &v3.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netSecPolicyName}, Spec: calico.NetworkPolicySpec{Tier: "net-sec"}}
	policyServer, err = policyClient.Create(ctx, netSecPolicy, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", netSecPolicy, err)
	}
	if netSecPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	// Should be listing the policy under default tier.
	policies, err = policyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if 1 != len(policies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(policies.Items))
	}

	// Should be listing the policy under "net-sec" tier
	policies, err = policyClient.List(ctx, metav1.ListOptions{FieldSelector: "spec.tier=net-sec"})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if 1 != len(policies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(policies.Items))
	}

	policyServer, err = policyClient.Get(ctx, name, metav1.GetOptions{})
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
	wIface, err := policyClient.Watch(ctx, opts)
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

	err = policyClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("policy should be deleted (%s)", err)
	}

	err = policyClient.Delete(ctx, netSecPolicyName, metav1.DeleteOptions{})
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
				return &v3.NetworkPolicy{}
			}, true)
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
	policy := &v3.StagedNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: defaultTierPolicyName},
		Spec:       calico.StagedNetworkPolicySpec{StagedAction: "Set", Selector: "foo == \"bar\""},
	}
	ctx := context.Background()

	// start from scratch
	policies, err := policyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if policies.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(policies.Items) > 0 {
		return fmt.Errorf("policies should not exist on start, had %v policies", len(policies.Items))
	}

	policyServer, err := policyClient.Create(ctx, policy, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", policy, err)
	}
	if defaultTierPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	updatedPolicy := policyServer
	updatedPolicy.Labels = map[string]string{"foo": "bar"}
	policyServer, err = policyClient.Update(ctx, updatedPolicy, metav1.UpdateOptions{})
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

	tierClient.Create(ctx, tier, metav1.CreateOptions{})
	defer func() {
		tierClient.Delete(ctx, "net-sec", metav1.DeleteOptions{})
	}()

	netSecPolicyName := "net-sec" + "." + name
	netSecPolicy := &v3.StagedNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netSecPolicyName}, Spec: calico.StagedNetworkPolicySpec{StagedAction: "Set", Selector: "foo == \"bar\"", Tier: "net-sec"}}
	policyServer, err = policyClient.Create(ctx, netSecPolicy, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", netSecPolicy, err)
	}
	if netSecPolicyName != policyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	// Should be listing the policy under default tier.
	policies, err = policyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if 1 != len(policies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(policies.Items))
	}

	// Should be listing the policy under "net-sec" tier
	policies, err = policyClient.List(ctx, metav1.ListOptions{FieldSelector: "spec.tier=net-sec"})
	if err != nil {
		return fmt.Errorf("error listing policies (%s)", err)
	}
	if 1 != len(policies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(policies.Items))
	}

	policyServer, err = policyClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting policy %s (%s)", name, err)
	}
	if name != policyServer.Name &&
		policy.ResourceVersion == policyServer.ResourceVersion {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", policy, policyServer)
	}

	// Watch Test:
	opts := v1.ListOptions{Watch: true}
	wIface, err := policyClient.Watch(ctx, opts)
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

	err = policyClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("policy should be deleted (%s)", err)
	}

	err = policyClient.Delete(ctx, netSecPolicyName, metav1.DeleteOptions{})
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
				return &v3.Tier{}
			}, true)
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
	ctx := context.Background()

	err := createEnterprise(client, ctx)
	if err == nil {
		return fmt.Errorf("Could not create a license")
	}

	// start from scratch
	tiers, err := tierClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing tiers (%s)", err)
	}
	if tiers.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	tierServer, err := tierClient.Create(ctx, tier, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the tier '%v' (%v)", tier, err)
	}
	if name != tierServer.Name {
		return fmt.Errorf("didn't get the same tier back from the server \n%+v\n%+v", tier, tierServer)
	}

	tiers, err = tierClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing tiers (%s)", err)
	}

	tierServer, err = tierClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting tier %s (%s)", name, err)
	}
	if name != tierServer.Name &&
		tier.ResourceVersion == tierServer.ResourceVersion {
		return fmt.Errorf("didn't get the same tier back from the server \n%+v\n%+v", tier, tierServer)
	}

	err = tierClient.Delete(ctx, name, metav1.DeleteOptions{})
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
				return &v3.GlobalNetworkPolicy{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	globalNetworkPolicies, err := globalNetworkPolicyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkPolicies (%s)", err)
	}
	if globalNetworkPolicies.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalNetworkPolicyServer, err := globalNetworkPolicyClient.Create(ctx, globalNetworkPolicy, metav1.CreateOptions{})
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

	tierClient.Create(ctx, tier, metav1.CreateOptions{})
	defer func() {
		tierClient.Delete(ctx, "net-sec", metav1.DeleteOptions{})
	}()

	netSecPolicyName := "net-sec" + "." + name
	netSecPolicy := &v3.GlobalNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netSecPolicyName}, Spec: calico.GlobalNetworkPolicySpec{Tier: "net-sec"}}
	globalNetworkPolicyServer, err = globalNetworkPolicyClient.Create(ctx, netSecPolicy, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", netSecPolicy, err)
	}
	if netSecPolicyName != globalNetworkPolicyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", netSecPolicy, globalNetworkPolicyServer)
	}

	// Should be listing the policy under "default" tier
	globalNetworkPolicies, err = globalNetworkPolicyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkPolicies (%s)", err)
	}
	if 1 != len(globalNetworkPolicies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(globalNetworkPolicies.Items))
	}

	// Should be listing the policy under "net-sec" tier
	globalNetworkPolicies, err = globalNetworkPolicyClient.List(ctx, metav1.ListOptions{FieldSelector: "spec.tier=net-sec"})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkPolicies (%s)", err)
	}
	if 1 != len(globalNetworkPolicies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(globalNetworkPolicies.Items))
	}

	globalNetworkPolicyServer, err = globalNetworkPolicyClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalNetworkPolicy %s (%s)", name, err)
	}
	if name != globalNetworkPolicyServer.Name &&
		globalNetworkPolicy.ResourceVersion == globalNetworkPolicyServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalNetworkPolicy back from the server \n%+v\n%+v", globalNetworkPolicy, globalNetworkPolicyServer)
	}

	err = globalNetworkPolicyClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalNetworkPolicy should be deleted (%s)", err)
	}

	err = globalNetworkPolicyClient.Delete(ctx, netSecPolicyName, metav1.DeleteOptions{})
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
				return &v3.StagedGlobalNetworkPolicy{}
			}, true)
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
	stagedGlobalNetworkPolicy := &v3.StagedGlobalNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: defaultTierPolicyName},
		Spec:       calico.StagedGlobalNetworkPolicySpec{StagedAction: "Set", Selector: "foo == \"bar\""},
	}
	ctx := context.Background()

	// start from scratch
	stagedGlobalNetworkPolicies, err := stagedGlobalNetworkPolicyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing stagedglobalNetworkPolicies (%s)", err)
	}
	if stagedGlobalNetworkPolicies.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	stagedGlobalNetworkPolicyServer, err := stagedGlobalNetworkPolicyClient.Create(ctx, stagedGlobalNetworkPolicy, metav1.CreateOptions{})
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

	tierClient.Create(ctx, tier, metav1.CreateOptions{})
	defer func() {
		tierClient.Delete(ctx, "net-sec", metav1.DeleteOptions{})
	}()

	netSecPolicyName := "net-sec" + "." + name
	netSecPolicy := &v3.StagedGlobalNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netSecPolicyName}, Spec: calico.StagedGlobalNetworkPolicySpec{StagedAction: "Set", Selector: "foo == \"bar\"", Tier: "net-sec"}}
	stagedGlobalNetworkPolicyServer, err = stagedGlobalNetworkPolicyClient.Create(ctx, netSecPolicy, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the policy '%v' (%v)", netSecPolicy, err)
	}
	if netSecPolicyName != stagedGlobalNetworkPolicyServer.Name {
		return fmt.Errorf("didn't get the same policy back from the server \n%+v\n%+v", netSecPolicy, stagedGlobalNetworkPolicyServer)
	}

	// Should be listing the policy under "default" tier
	stagedGlobalNetworkPolicies, err = stagedGlobalNetworkPolicyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing stagedGlobalNetworkPolicies (%s)", err)
	}
	if 1 != len(stagedGlobalNetworkPolicies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(stagedGlobalNetworkPolicies.Items))
	}

	// Should be listing the policy under "net-sec" tier
	stagedGlobalNetworkPolicies, err = stagedGlobalNetworkPolicyClient.List(ctx, metav1.ListOptions{FieldSelector: "spec.tier=net-sec"})
	if err != nil {
		return fmt.Errorf("error listing stagedGlobalNetworkPolicies (%s)", err)
	}
	if 1 != len(stagedGlobalNetworkPolicies.Items) {
		return fmt.Errorf("should have exactly one policies, had %v policies", len(stagedGlobalNetworkPolicies.Items))
	}

	stagedGlobalNetworkPolicyServer, err = stagedGlobalNetworkPolicyClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting stagedGlobalNetworkPolicy %s (%s)", name, err)
	}
	if name != stagedGlobalNetworkPolicyServer.Name &&
		stagedGlobalNetworkPolicy.ResourceVersion == stagedGlobalNetworkPolicyServer.ResourceVersion {
		return fmt.Errorf("didn't get the same stagedGlobalNetworkPolicy back from the server \n%+v\n%+v", stagedGlobalNetworkPolicy, stagedGlobalNetworkPolicyServer)
	}

	err = stagedGlobalNetworkPolicyClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("stagedGlobalNetworkPolicy should be deleted (%s)", err)
	}

	err = stagedGlobalNetworkPolicyClient.Delete(ctx, netSecPolicyName, metav1.DeleteOptions{})
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
				return &v3.GlobalNetworkSet{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	globalNetworkSets, err := globalNetworkSetClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkSets (%s)", err)
	}
	if globalNetworkSets.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalNetworkSetServer, err := globalNetworkSetClient.Create(ctx, globalNetworkSet, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the globalNetworkSet '%v' (%v)", globalNetworkSet, err)
	}
	if name != globalNetworkSetServer.Name {
		return fmt.Errorf("didn't get the same globalNetworkSet back from the server \n%+v\n%+v", globalNetworkSet, globalNetworkSetServer)
	}

	globalNetworkSets, err = globalNetworkSetClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalNetworkSets (%s)", err)
	}

	globalNetworkSetServer, err = globalNetworkSetClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalNetworkSet %s (%s)", name, err)
	}
	if name != globalNetworkSetServer.Name &&
		globalNetworkSet.ResourceVersion == globalNetworkSetServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalNetworkSet back from the server \n%+v\n%+v", globalNetworkSet, globalNetworkSetServer)
	}

	err = globalNetworkSetClient.Delete(ctx, name, metav1.DeleteOptions{})
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
				return &v3.NetworkSet{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	networkSets, err := networkSetClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing networkSets (%s)", err)
	}
	if networkSets.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(networkSets.Items) > 0 {
		return fmt.Errorf("networkSets should not exist on start, had %v networkSets", len(networkSets.Items))
	}

	networkSetServer, err := networkSetClient.Create(ctx, networkSet, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the networkSet '%v' (%v)", networkSet, err)
	}

	updatedNetworkSet := networkSetServer
	updatedNetworkSet.Labels = map[string]string{"foo": "bar"}
	networkSetServer, err = networkSetClient.Update(ctx, updatedNetworkSet, metav1.UpdateOptions{})
	if nil != err {
		return fmt.Errorf("error updating the networkSet '%v' (%v)", networkSet, err)
	}

	// Should be listing the networkSet.
	networkSets, err = networkSetClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing networkSets (%s)", err)
	}
	if 1 != len(networkSets.Items) {
		return fmt.Errorf("should have exactly one networkSet, had %v networkSets", len(networkSets.Items))
	}

	networkSetServer, err = networkSetClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting networkSet %s (%s)", name, err)
	}
	if name != networkSetServer.Name &&
		networkSet.ResourceVersion == networkSetServer.ResourceVersion {
		return fmt.Errorf("didn't get the same networkSet back from the server \n%+v\n%+v", networkSet, networkSetServer)
	}

	// Watch Test:
	opts := v1.ListOptions{Watch: true}
	wIface, err := networkSetClient.Watch(ctx, opts)
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

	err = networkSetClient.Delete(ctx, name, metav1.DeleteOptions{})
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
				return &v3.LicenseKey{}
			}, false)
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
	ctx := context.Background()

	licenseKeys, err := licenseKeyClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing licenseKeys (%s)", err)
	}
	if licenseKeys.Items == nil {
		return fmt.Errorf("items field should not be set to nil")
	}

	// Validate that a license not encrypted with production key is rejected
	corruptLicenseKey := &v3.LicenseKey{ObjectMeta: metav1.ObjectMeta{Name: name}}

	_, err = licenseKeyClient.Create(ctx, corruptLicenseKey, metav1.CreateOptions{})
	if err == nil {
		return fmt.Errorf("expected creating the emptyLicenseKey")
	}

	// Confirm that valid, but expired licenses, are rejected
	expiredLicenseKey := utils.ExpiredTestLicense()
	_, err = licenseKeyClient.Create(ctx, expiredLicenseKey, metav1.CreateOptions{})
	if err == nil {
		return fmt.Errorf("expected creating the expiredLicenseKey")
	} else if err.Error() != "LicenseKey.projectcalico.org \"default\" is invalid: LicenseKeySpec.token: Internal error: the license you're trying to create expired on 2019-02-08 07:59:59 +0000 UTC" {
		fmt.Printf("Incorrect error: %+v\n", err)
	}
	// Valid Enterprise License with Maximum supported Nodes 100
	enterpriseValidLicenseKey := utils.ValidEnterpriseTestLicense()
	claims, err := licclient.Decode(*enterpriseValidLicenseKey)
	if err != nil {
		fmt.Printf("Failed to decode 'valid' license  %v\n", err)
		return err
	}

	lic, err := licenseKeyClient.Create(ctx, enterpriseValidLicenseKey, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Check for License Expiry date %v\n", err)
		return err
	}

	// Check for maximum nodes.
	if lic.Status.MaxNodes != *claims.Nodes {
		fmt.Printf("Valid License's Maximum Node doesn't match :%d\n", lic.Status.MaxNodes)
		return fmt.Errorf("Incorrect Maximum Nodes in LicenseKey")
	}

	// Check for Certificate Expiry date exists.  Since hte cert is provided to us as configuration, we can't check
	// the exact date.
	if !lic.Status.Expiry.Time.After(time.Now()) {
		fmt.Printf("Valid License's Expiry date missing/in past:%v\n", lic.Status.Expiry)
		return fmt.Errorf("License Expiry date don't match")
	}

	if lic.Status.Package != "Enterprise" {
		fmt.Printf("License's package type does not match :%v\n", lic.Status.Package)
		return fmt.Errorf("License Package Type does not match")
	}

	if !reflect.DeepEqual(lic.Status.Features, sortedKeys(map[string]bool{"cnx": true, "all": true})) {
		fmt.Printf("License's features do not match :%v with %v\n", lic.Status.Features, sortedKeys(map[string]bool{"cnx": true, "all": true}))
		return fmt.Errorf("License features do not match")
	}

	// Valid CloudPro License with Maximum supported Nodes 100
	cloudProLicenseKey := utils.ValidCloudProTestLicense()
	licenseKeyClient.Delete(ctx, "default", metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("Could not delete license %v\n", err)
		return err
	}
	lic, err = licenseKeyClient.Create(ctx, cloudProLicenseKey, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Check for License Expiry date %v\n", err)
		return err
	}

	claims, err = licclient.Decode(*cloudProLicenseKey)
	if err != nil {
		fmt.Printf("Failed to decode 'valid' license  %v\n", err)
		return err
	}

	// Check for Maximum nodes
	if lic.Status.MaxNodes != *claims.Nodes {
		fmt.Printf("Valid License's Maximum Node doesn't match :%d\n", lic.Status.MaxNodes)
		return fmt.Errorf("Incorrect Maximum Nodes in LicenseKey")
	}

	// Check for Certificate Expiry date
	if !lic.Status.Expiry.Time.After(time.Now()) {
		fmt.Printf("Valid License's Expiry date missing/in past:%v\n", lic.Status.Expiry)
		return fmt.Errorf("License Expiry date don't match")
	}

	if lic.Status.Package != "CloudPro" {
		fmt.Printf("License's package type does not match :%v\n", lic.Status.Package)
		return fmt.Errorf("License Package Type does not match")
	}

	// Extract out "cloud" and "pro" which isn't really the expected
	// feature.
	features := []string{}
	for _, feat := range lic.Status.Features {
		if feat == "cloud" || feat == "pro" {
			continue
		}
		features = append(features, feat)
	}
	if !reflect.DeepEqual(features, sortedKeys(licFeatures.CloudProFeatures)) {
		fmt.Printf("License's features do not match :%v with %v\n", lic.Status.Features, sortedKeys(licFeatures.CloudProFeatures))
		return fmt.Errorf("License features do not match")
	}

	return nil
}

// TestAlertExceptionClient exercises the AlertException client.
func TestAlertExceptionClient(t *testing.T) {
	const name = "test-alertexception"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.AlertException{}
			}, true)
			defer shutdownServer()
			if err := testAlertExceptionClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-alertexception test failed")
	}
}

func testAlertExceptionClient(client calicoclient.Interface, name string) error {
	alertExceptionClient := client.ProjectcalicoV3().AlertExceptions()
	alertException := &v3.AlertException{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.AlertExceptionSpec{
			Description: "alert exception description",
			Selector:    "origin=someorigin",
			StartTime:   metav1.Time{Time: time.Now()},
		},
		Status: calico.AlertExceptionStatus{},
	}
	ctx := context.Background()

	// start from scratch
	alertExceptions, err := alertExceptionClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing alertException (%s)", err)
	}
	if alertExceptions.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	alertExceptionServer, err := alertExceptionClient.Create(ctx, alertException, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the alertException '%v' (%v)", alertException, err)
	}
	if name != alertExceptionServer.Name {
		return fmt.Errorf("didn't get the same alertException back from the server \n%+v\n%+v", alertException, alertExceptionServer)
	}
	if !reflect.DeepEqual(alertExceptionServer.Status, calico.AlertExceptionStatus{}) {
		return fmt.Errorf("status was set on create to %#v", alertException.Status)
	}

	alertExceptions, err = alertExceptionClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing alertExceptions (%s)", err)
	}
	if len(alertExceptions.Items) != 1 {
		return fmt.Errorf("expected 1 alertException got %d", len(alertExceptions.Items))
	}

	alertExceptionServer, err = alertExceptionClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting alertException %s (%s)", name, err)
	}
	if name != alertExceptionServer.Name &&
		alertException.ResourceVersion == alertExceptionServer.ResourceVersion {
		return fmt.Errorf("didn't get the same alertException back from the server \n%+v\n%+v", alertException, alertExceptionServer)
	}

	alertExceptionUpdate := alertExceptionServer.DeepCopy()
	alertExceptionUpdate.Spec.Description += "-updated"
	alertExceptionServer, err = alertExceptionClient.Update(ctx, alertExceptionUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating alertException %s (%s)", name, err)
	}
	if alertExceptionServer.Spec.Description != alertExceptionUpdate.Spec.Description {
		return errors.New("didn't update spec.description")
	}

	alertExceptionUpdate = alertExceptionServer.DeepCopy()
	alertExceptionUpdate.Labels = map[string]string{"foo": "bar"}
	statusDescription := "status"
	alertExceptionUpdate.Spec.Description = statusDescription
	alertExceptionServer, err = alertExceptionClient.UpdateStatus(ctx, alertExceptionUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating alertException %s (%s)", name, err)
	}
	if _, ok := alertExceptionServer.Labels["foo"]; ok {
		return fmt.Errorf("updatestatus updated labels")
	}
	if alertExceptionServer.Spec.Description == statusDescription {
		return fmt.Errorf("updatestatus updated spec")
	}

	err = alertExceptionClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("alertException should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().AlertExceptions().Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching AlertExceptions (%s)", err)
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
				timeoutErr = fmt.Errorf("timed out waiting for events")
				return
			}
		}
		return
	}()

	// Create two AlertExceptions
	for i := 0; i < 2; i++ {
		ae := &v3.AlertException{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ae%d", i)},
			Spec: calico.AlertExceptionSpec{
				Description: "test",
				Selector:    "origin=someorigin",
				StartTime:   metav1.Time{Time: time.Now()},
			},
		}
		_, err = alertExceptionClient.Create(ctx, ae, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating the alertException '%v' (%v)", ae, err)
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

// TestGlobalAlertClient exercises the GlobalAlert client.
func TestGlobalAlertClient(t *testing.T) {
	const name = "test-globalalert"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.GlobalAlert{}
			}, true)
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
			LastUpdate:   &v1.Time{Time: time.Now()},
			Active:       false,
			Healthy:      false,
			LastExecuted: &v1.Time{Time: time.Now()},
			LastEvent:    &v1.Time{Time: time.Now()},
			ErrorConditions: []calico.ErrorCondition{
				{Type: "foo", Message: "bar"},
			},
		},
	}
	ctx := context.Background()

	// start from scratch
	globalAlerts, err := globalAlertClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalAlerts (%s)", err)
	}
	if globalAlerts.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalAlertServer, err := globalAlertClient.Create(ctx, globalAlert, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the globalAlert '%v' (%v)", globalAlert, err)
	}
	if name != globalAlertServer.Name {
		return fmt.Errorf("didn't get the same globalAlert back from the server \n%+v\n%+v", globalAlert, globalAlertServer)
	}
	if !reflect.DeepEqual(globalAlertServer.Status, calico.GlobalAlertStatus{}) {
		return fmt.Errorf("status was set on create to %#v", globalAlertServer.Status)
	}

	globalAlerts, err = globalAlertClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalAlerts (%s)", err)
	}
	if len(globalAlerts.Items) != 1 {
		return fmt.Errorf("expected 1 globalAlert got %d", len(globalAlerts.Items))
	}

	globalAlertServer, err = globalAlertClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalAlert %s (%s)", name, err)
	}
	if name != globalAlertServer.Name &&
		globalAlert.ResourceVersion == globalAlertServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalAlert back from the server \n%+v\n%+v", globalAlert, globalAlertServer)
	}

	globalAlertUpdate := globalAlertServer.DeepCopy()
	globalAlertUpdate.Spec.Description += "-updated"
	globalAlertUpdate.Status.LastUpdate = &v1.Time{Time: time.Now()}
	globalAlertServer, err = globalAlertClient.Update(ctx, globalAlertUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating globalAlert %s (%s)", name, err)
	}
	if globalAlertServer.Spec.Description != globalAlertUpdate.Spec.Description {
		return errors.New("didn't update spec.content")
	}
	if globalAlertServer.Status.LastUpdate != nil {
		return errors.New("status was updated by Update()")
	}

	globalAlertUpdate = globalAlertServer.DeepCopy()
	globalAlertUpdate.Status.LastUpdate = &v1.Time{Time: time.Now()}
	globalAlertUpdate.Labels = map[string]string{"foo": "bar"}
	statusDescription := "status"
	globalAlertUpdate.Spec.Description = statusDescription
	globalAlertServer, err = globalAlertClient.UpdateStatus(ctx, globalAlertUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating globalAlert %s (%s)", name, err)
	}
	if globalAlertServer.Status.LastUpdate == nil {
		return fmt.Errorf("didn't update status. %v != %v", globalAlertUpdate.Status, globalAlertServer.Status)
	}
	if _, ok := globalAlertServer.Labels["foo"]; ok {
		return fmt.Errorf("updatestatus updated labels")
	}
	if globalAlertServer.Spec.Description == statusDescription {
		return fmt.Errorf("updatestatus updated spec")
	}

	err = globalAlertClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalAlert should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().GlobalAlerts().Watch(ctx, v1.ListOptions{})
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
		_, err = globalAlertClient.Create(ctx, ga, metav1.CreateOptions{})
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
				return &v3.GlobalAlertTemplate{}
			}, true)
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
			Summary:     "foo",
			DataSet:     "dns",
			Description: "test",
			Severity:    100,
		},
	}
	ctx := context.Background()

	// start from scratch
	globalAlerts, err := globalAlertClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalAlertTemplates (%s)", err)
	}
	if globalAlerts.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalAlertServer, err := globalAlertClient.Create(ctx, globalAlert, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the globalAlertTemplate '%v' (%v)", globalAlert, err)
	}
	if name != globalAlertServer.Name {
		return fmt.Errorf("didn't get the same globalAlertTemplate back from the server \n%+v\n%+v", globalAlert, globalAlertServer)
	}

	globalAlerts, err = globalAlertClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalAlertTemplates (%s)", err)
	}
	if len(globalAlerts.Items) != 1 {
		return fmt.Errorf("expected 1 globalAlertTemplate got %d", len(globalAlerts.Items))
	}

	globalAlertServer, err = globalAlertClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalAlertTemplate %s (%s)", name, err)
	}
	if name != globalAlertServer.Name &&
		globalAlert.ResourceVersion == globalAlertServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalAlertTemplate back from the server \n%+v\n%+v", globalAlert, globalAlertServer)
	}

	globalAlertUpdate := globalAlertServer.DeepCopy()
	globalAlertUpdate.Spec.Description += "-update"
	globalAlertServer, err = globalAlertClient.Update(ctx, globalAlertUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating globalAlertTemplate %s (%s)", name, err)
	}
	if globalAlertServer.Spec.Description != globalAlertUpdate.Spec.Description {
		return errors.New("didn't update spec.content")
	}

	err = globalAlertClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalAlertTemplate should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().GlobalAlertTemplates().Watch(ctx, v1.ListOptions{})
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
				Summary:     "bar",
				Description: "test",
				Severity:    100,
				DataSet:     "dns",
			},
		}
		_, err = globalAlertClient.Create(ctx, ga, metav1.CreateOptions{})
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
	var mode *v3.ThreatFeedMode
	mode = new(v3.ThreatFeedMode)
	*mode = v3.ThreatFeedModeEnabled

	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.GlobalThreatFeed{
					Spec: v3.GlobalThreatFeedSpec{
						Mode:        mode,
						Description: "test",
					},
				}
			}, true)
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

// TestIPReservationClient exercises the IPReservation client.
func TestIPReservationClient(t *testing.T) {
	const name = "test-ipreservation"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.IPReservation{}
			}, true)
			defer shutdownServer()
			if err := testIPReservationClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-ipreservation test failed")
	}
}

func testIPReservationClient(client calicoclient.Interface, name string) error {
	ipreservationClient := client.ProjectcalicoV3().IPReservations()
	ipreservation := &v3.IPReservation{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v3.IPReservationSpec{
			ReservedCIDRs: []string{"192.168.0.0/16"},
		},
	}
	ctx := context.Background()

	// start from scratch
	ipreservations, err := ipreservationClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ipreservations (%s)", err)
	}
	if ipreservations.Items == nil {
		return fmt.Errorf("items field should not be set to nil")
	}

	ipreservationServer, err := ipreservationClient.Create(ctx, ipreservation, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the ipreservation '%v' (%v)", ipreservation, err)
	}
	if name != ipreservationServer.Name {
		return fmt.Errorf("didn't get the same ipreservation back from the server \n%+v\n%+v", ipreservation, ipreservationServer)
	}

	ipreservations, err = ipreservationClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ipreservations (%s)", err)
	}

	ipreservationServer, err = ipreservationClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting ipreservation %s (%s)", name, err)
	}
	if name != ipreservationServer.Name &&
		ipreservation.ResourceVersion == ipreservationServer.ResourceVersion {
		return fmt.Errorf("didn't get the same ipreservation back from the server \n%+v\n%+v", ipreservation, ipreservationServer)
	}

	err = ipreservationClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("ipreservation should be deleted (%s)", err)
	}

	return nil
}

func testGlobalThreatFeedClient(client calicoclient.Interface, name string) error {
	var mode *v3.ThreatFeedMode
	mode = new(v3.ThreatFeedMode)
	*mode = v3.ThreatFeedModeEnabled

	globalThreatFeedClient := client.ProjectcalicoV3().GlobalThreatFeeds()
	globalThreatFeed := &v3.GlobalThreatFeed{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.GlobalThreatFeedSpec{
			Mode:        mode,
			Description: "test",
		},
		Status: calico.GlobalThreatFeedStatus{
			LastSuccessfulSync:   &metav1.Time{Time: time.Now()},
			LastSuccessfulSearch: &metav1.Time{Time: time.Now()},
			ErrorConditions: []calico.ErrorCondition{
				{
					Type:    "foo",
					Message: "bar",
				},
			},
		},
	}
	ctx := context.Background()

	// start from scratch
	globalThreatFeeds, err := globalThreatFeedClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalThreatFeeds (%s)", err)
	}
	if globalThreatFeeds.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	globalThreatFeedServer, err := globalThreatFeedClient.Create(ctx, globalThreatFeed, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the globalThreatFeed '%v' (%v)", globalThreatFeed, err)
	}
	if name != globalThreatFeedServer.Name {
		return fmt.Errorf("didn't get the same globalThreatFeed back from the server \n%+v\n%+v", globalThreatFeed, globalThreatFeedServer)
	}
	if !reflect.DeepEqual(globalThreatFeedServer.Status, calico.GlobalThreatFeedStatus{}) {
		return fmt.Errorf("status was set on create to %#v", globalThreatFeedServer.Status)
	}

	globalThreatFeeds, err = globalThreatFeedClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalThreatFeeds (%s)", err)
	}
	if len(globalThreatFeeds.Items) != 1 {
		return fmt.Errorf("expected 1 globalThreatFeed got %d", len(globalThreatFeeds.Items))
	}

	globalThreatFeedServer, err = globalThreatFeedClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalThreatFeed %s (%s)", name, err)
	}
	if name != globalThreatFeedServer.Name &&
		globalThreatFeed.ResourceVersion == globalThreatFeedServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalThreatFeed back from the server \n%+v\n%+v", globalThreatFeed, globalThreatFeedServer)
	}

	globalThreatFeedUpdate := globalThreatFeedServer.DeepCopy()
	globalThreatFeedUpdate.Spec.Content = "IPSet"
	globalThreatFeedUpdate.Spec.Mode = mode
	globalThreatFeedUpdate.Spec.Description = "test"
	globalThreatFeedUpdate.Status.LastSuccessfulSync = &v1.Time{Time: time.Now()}
	globalThreatFeedServer, err = globalThreatFeedClient.Update(ctx, globalThreatFeedUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating globalThreatFeed %s (%s)", name, err)
	}
	if globalThreatFeedServer.Spec.Content != globalThreatFeedUpdate.Spec.Content {
		return errors.New("didn't update spec.content")
	}
	if globalThreatFeedServer.Status.LastSuccessfulSync != nil {
		return errors.New("status was updated by Update()")
	}

	// NOTE: The update status test currently doesn't work because the GlobalThreatFeed's crd.projectcalico.org status
	// is set as a subresource and the apiserver doesn't handle subresource yet. Uncomment this when this is dealt with.

	globalThreatFeedUpdate = globalThreatFeedServer.DeepCopy()
	globalThreatFeedUpdate.Status.LastSuccessfulSync = &v1.Time{Time: time.Now()}
	globalThreatFeedUpdate.Labels = map[string]string{"foo": "bar"}
	globalThreatFeedUpdate.Spec.Content = ""
	globalThreatFeedServer, err = globalThreatFeedClient.UpdateStatus(ctx, globalThreatFeedUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating globalThreatFeed %s (%s)", name, err)
	}
	if globalThreatFeedServer.Status.LastSuccessfulSync == nil {
		return fmt.Errorf("didn't update status. %v != %v", globalThreatFeedUpdate.Status, globalThreatFeedServer.Status)
	}
	if _, ok := globalThreatFeedServer.Labels["foo"]; ok {
		return fmt.Errorf("updatestatus updated labels")
	}
	if globalThreatFeedServer.Spec.Content == "" {
		return fmt.Errorf("updatestatus updated spec")
	}

	err = globalThreatFeedClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalThreatFeed should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().GlobalThreatFeeds().Watch(ctx, v1.ListOptions{})
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
			Spec: v3.GlobalThreatFeedSpec{
				Mode:        mode,
				Description: "test",
			},
		}
		_, err = globalThreatFeedClient.Create(ctx, gtf, metav1.CreateOptions{})
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

	// Delete two GlobalThreatFeeds
	for i := 0; i < 2; i++ {
		gtf := fmt.Sprintf("gtf%d", i)
		err = globalThreatFeedClient.Delete(ctx, gtf, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("error creating the globalThreatFeed '%v' (%v)", gtf, err)
		}
	}

	return nil
}

// TestHostEndpointClient exercises the HostEndpoint client.
func TestHostEndpointClient(t *testing.T) {
	const name = "test-hostendpoint"
	client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
		return &v3.HostEndpoint{}
	}, true)
	defer shutdownServer()
	defer deleteHostEndpointClient(client, name)
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.HostEndpoint{}
			}, true)
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

func deleteHostEndpointClient(client calicoclient.Interface, name string) error {
	hostEndpointClient := client.ProjectcalicoV3().HostEndpoints()
	ctx := context.Background()

	return hostEndpointClient.Delete(ctx, name, v1.DeleteOptions{})
}

func testHostEndpointClient(client calicoclient.Interface, name string) error {
	hostEndpointClient := client.ProjectcalicoV3().HostEndpoints()

	hostEndpoint := createTestHostEndpoint(name, "192.168.0.1", "test-node")
	ctx := context.Background()

	// start from scratch
	hostEndpoints, err := hostEndpointClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing hostEndpoints (%s)", err)
	}
	if hostEndpoints.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	hostEndpointServer, err := hostEndpointClient.Create(ctx, hostEndpoint, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the hostEndpoint '%v' (%v)", hostEndpoint, err)
	}
	if name != hostEndpointServer.Name {
		return fmt.Errorf("didn't get the same hostEndpoint back from the server \n%+v\n%+v", hostEndpoint, hostEndpointServer)
	}

	hostEndpoints, err = hostEndpointClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing hostEndpoints (%s)", err)
	}
	if len(hostEndpoints.Items) != 1 {
		return fmt.Errorf("expected 1 hostEndpoint entry, got %d", len(hostEndpoints.Items))
	}

	hostEndpointServer, err = hostEndpointClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting hostEndpoint %s (%s)", name, err)
	}
	if name != hostEndpointServer.Name &&
		hostEndpoint.ResourceVersion == hostEndpointServer.ResourceVersion {
		return fmt.Errorf("didn't get the same hostEndpoint back from the server \n%+v\n%+v", hostEndpoint, hostEndpointServer)
	}

	err = hostEndpointClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("hostEndpoint should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().HostEndpoints().Watch(ctx, v1.ListOptions{})
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
		_, err = hostEndpointClient.Create(ctx, hep, metav1.CreateOptions{})
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
				return &v3.GlobalReport{}
			}, true)
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
						Start: metav1.Time{Time: time.Now()},
						End:   metav1.Time{Time: time.Now()},
						Job: &corev1.ObjectReference{
							Kind:      "NetworkPolicy",
							Name:      "fbar-srj",
							Namespace: "fbar-ns-srj",
						},
					},
					JobCompletionTime: &metav1.Time{Time: time.Now()},
				},
			},
			LastFailedReportJobs: []calico.CompletedReportJob{
				{
					ReportJob: calico.ReportJob{
						Start: metav1.Time{Time: time.Now()},
						End:   metav1.Time{Time: time.Now()},
						Job: &corev1.ObjectReference{
							Kind:      "NetworkPolicy",
							Name:      "fbar-frj",
							Namespace: "fbar-ns-frj",
						},
					},
					JobCompletionTime: &metav1.Time{Time: time.Now()},
				},
			},
			ActiveReportJobs: []calico.ReportJob{
				{
					Start: metav1.Time{Time: time.Now()},
					End:   metav1.Time{Time: time.Now()},
					Job: &corev1.ObjectReference{
						Kind:      "NetworkPolicy",
						Name:      "fbar-arj",
						Namespace: "fbar-ns-arj",
					},
				},
			},
			LastScheduledReportJob: &calico.ReportJob{
				Start: metav1.Time{Time: time.Now()},
				End:   metav1.Time{Time: time.Now()},
				Job: &corev1.ObjectReference{
					Kind:      "NetworkPolicy",
					Name:      "fbar-lsj",
					Namespace: "fbar-ns-lsj",
				},
			},
		},
	}
	ctx := context.Background()

	// Make sure there is no GlobalReport configured.
	globalReports, err := globalReportClient.List(ctx, metav1.ListOptions{})
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
	_, err = globalReportTypeClient.Create(ctx, globalReportType, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the pre-requisite globalReportType '%v' (%v)", globalReportType, err)
	}

	globalReportServer, err := globalReportClient.Create(ctx, globalReport, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the globalReport '%v' (%v)", globalReport, err)
	}
	if name != globalReportServer.Name {
		return fmt.Errorf("didn't get the same globalReport back from the server \n%+v\n%+v", globalReport, globalReportServer)
	}
	if !reflect.DeepEqual(globalReportServer.Status, calico.ReportStatus{}) {
		return fmt.Errorf("status was set on create to %#v", globalReportServer.Status)
	}

	globalReports, err = globalReportClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalReports (%s)", err)
	}
	if len(globalReports.Items) != 1 {
		return fmt.Errorf("expected 1 globalReport entry, got %d", len(globalReports.Items))
	}

	globalReportServer, err = globalReportClient.Get(ctx, name, metav1.GetOptions{})
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

	globalReportServer, err = globalReportClient.Update(ctx, globalReportUpdate, metav1.UpdateOptions{})
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
		{ReportJob: calico.ReportJob{
			Start: v1.Time{Time: time.Now()},
			End:   v1.Time{Time: time.Now()},
			Job:   &corev1.ObjectReference{},
		}, JobCompletionTime: &v1.Time{Time: time.Now()}},
	}
	globalReportUpdate.Labels = map[string]string{"foo": "bar"}
	globalReportServer, err = globalReportClient.UpdateStatus(ctx, globalReportUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating globalReport status %s (%s)", name, err)
	}
	if len(globalReportServer.Status.LastSuccessfulReportJobs) == 0 ||
		globalReportServer.Status.LastSuccessfulReportJobs[0].JobCompletionTime == nil ||
		globalReportServer.Status.LastSuccessfulReportJobs[0].JobCompletionTime.Time.Equal(time.Time{}) {
		return fmt.Errorf("didn't update GlobalReport status. %v != %v", globalReportUpdate.Status, globalReportServer.Status)
	}
	if _, ok := globalReportServer.Labels["foo"]; ok {
		return fmt.Errorf("updatestatus updated labels")
	}

	err = globalReportClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalReport should be deleted (%s)", err)
	}

	// Check list-ing GlobalReport resource works with watch option.
	w, err := client.ProjectcalicoV3().GlobalReports().Watch(ctx, v1.ListOptions{})
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
		_, err = globalReportClient.Create(ctx, gr, metav1.CreateOptions{})
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
	err = globalReportTypeClient.Delete(ctx, globalReportTypeName, metav1.DeleteOptions{})
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
				return &v3.GlobalReportType{}
			}, true)
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
	ctx := context.Background()

	// Make sure there is no GlobalReportType configured.
	globalReportTypes, err := globalReportTypeClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalReportTypes (%s)", err)
	}
	if globalReportTypes.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	// Create/List/Get/Delete tests.
	globalReportTypeServer, err := globalReportTypeClient.Create(ctx, globalReportType, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the globalReportType '%v' (%v)", globalReportType, err)
	}
	if name != globalReportTypeServer.Name {
		return fmt.Errorf("didn't get the same globalReportType back from the server \n%+v\n%+v", globalReportType, globalReportTypeServer)
	}

	globalReportTypes, err = globalReportTypeClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing globalReportTypes (%s)", err)
	}
	if len(globalReportTypes.Items) != 1 {
		return fmt.Errorf("expected 1 globalReportType entry, got %d", len(globalReportTypes.Items))
	}

	globalReportTypeServer, err = globalReportTypeClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting globalReportType %s (%s)", name, err)
	}
	if name != globalReportTypeServer.Name &&
		globalReportType.ResourceVersion == globalReportTypeServer.ResourceVersion {
		return fmt.Errorf("didn't get the same globalReportType back from the server \n%+v\n%+v", globalReportType, globalReportTypeServer)
	}

	err = globalReportTypeClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("globalReportType should be deleted (%s)", err)
	}

	// Check list-ing GlobalReportType resource works with watch option.
	w, err := client.ProjectcalicoV3().GlobalReportTypes().Watch(ctx, v1.ListOptions{})
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
		_, err = globalReportTypeClient.Create(ctx, grt, metav1.CreateOptions{})
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
				return &v3.IPPool{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	ippools, err := ippoolClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ippools (%s)", err)
	}
	if ippools.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	ippoolServer, err := ippoolClient.Create(ctx, ippool, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the ippool '%v' (%v)", ippool, err)
	}
	if name != ippoolServer.Name {
		return fmt.Errorf("didn't get the same ippool back from the server \n%+v\n%+v", ippool, ippoolServer)
	}

	ippools, err = ippoolClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ippools (%s)", err)
	}

	ippoolServer, err = ippoolClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting ippool %s (%s)", name, err)
	}
	if name != ippoolServer.Name &&
		ippool.ResourceVersion == ippoolServer.ResourceVersion {
		return fmt.Errorf("didn't get the same ippool back from the server \n%+v\n%+v", ippool, ippoolServer)
	}

	err = ippoolClient.Delete(ctx, name, metav1.DeleteOptions{})
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
				return &v3.BGPConfiguration{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	bgpConfigList, err := bgpConfigClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing bgpConfiguration (%s)", err)
	}
	if bgpConfigList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	bgpRes, err := bgpConfigClient.Create(ctx, bgpConfig, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the bgpConfiguration '%v' (%v)", bgpConfig, err)
	}
	if resName != bgpRes.Name {
		return fmt.Errorf("didn't get the same bgpConfig back from server\n%+v\n%+v", bgpConfig, bgpRes)
	}

	bgpRes, err = bgpConfigClient.Get(ctx, resName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting bgpConfiguration %s (%s)", resName, err)
	}

	err = bgpConfigClient.Delete(ctx, resName, metav1.DeleteOptions{})
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
				return &v3.BGPPeer{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	bgpPeerList, err := bgpPeerClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing bgpPeer (%s)", err)
	}
	if bgpPeerList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	bgpRes, err := bgpPeerClient.Create(ctx, bgpPeer, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the bgpPeer '%v' (%v)", bgpPeer, err)
	}
	if resName != bgpRes.Name {
		return fmt.Errorf("didn't get the same bgpPeer back from server\n%+v\n%+v", bgpPeer, bgpRes)
	}

	bgpRes, err = bgpPeerClient.Get(ctx, resName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting bgpPeer %s (%s)", resName, err)
	}

	err = bgpPeerClient.Delete(ctx, resName, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("BGPPeer should be deleted (%s)", err)
	}

	return nil
}

// TestProfileClient exercises the Profile client.
func TestProfileClient(t *testing.T) {
	// This matches the namespace that is created at test setup time in the Makefile.
	// TODO(doublek): Note that this currently only works for KDD mode.
	const name = "kns.namespace-1"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.Profile{}
			}, true)
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
	profile := &v3.Profile{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: calico.ProfileSpec{
			LabelsToApply: map[string]string{
				"aa": "bb",
			},
		},
	}
	ctx := context.Background()

	// start from scratch
	profileList, err := profileClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing profile (%s)", err)
	}
	if profileList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	// Profile creation is not supported.
	_, err = profileClient.Create(ctx, profile, metav1.CreateOptions{})
	if err == nil {
		return fmt.Errorf("profile should not be allowed to be created'%v' (%v)", profile, err)
	}

	profileRes, err := profileClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting profile %s (%s)", name, err)
	}

	if name != profileRes.Name {
		return fmt.Errorf("didn't get the same profile back from server\n%+v\n%+v", profile, profileRes)
	}

	// Profile deletion is not supported.
	err = profileClient.Delete(ctx, name, metav1.DeleteOptions{})
	if err == nil {
		return fmt.Errorf("Profile cannot be deleted (%s)", err)
	}

	return nil
}

// TestRemoteClusterConfigurationClient exercises the RemoteClusterConfiguration client.
func TestRemoteClusterConfigurationClient(t *testing.T) {
	const name = "test-remoteclusterconfig"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.RemoteClusterConfiguration{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	rccList, err := rccClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing remoteClusterConfiguration (%s)", err)
	}
	if rccList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	rccRes, err := rccClient.Create(ctx, rcc, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the remoteClusterConfiguration '%v' (%v)", rcc, err)
	}
	if resName != rccRes.Name {
		return fmt.Errorf("didn't get the same remoteClusterConfiguration back from server\n%+v\n%+v", rcc, rccRes)
	}

	rccRes, err = rccClient.Get(ctx, resName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting remoteClusterConfiguration %s (%s)", resName, err)
	}

	err = rccClient.Delete(ctx, resName, metav1.DeleteOptions{})
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
				return &v3.FelixConfiguration{}
			}, true)
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
	ctx := context.Background()

	// start from scratch
	felixConfigs, err := felixConfigClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing felixConfigs (%s)", err)
	}
	if felixConfigs.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	felixConfigServer, err := felixConfigClient.Create(ctx, felixConfig, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the felixConfig '%v' (%v)", felixConfig, err)
	}
	if name != felixConfigServer.Name {
		return fmt.Errorf("didn't get the same felixConfig back from the server \n%+v\n%+v", felixConfig, felixConfigServer)
	}

	felixConfigs, err = felixConfigClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing felixConfigs (%s)", err)
	}

	felixConfigServer, err = felixConfigClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting felixConfig %s (%s)", name, err)
	}
	if name != felixConfigServer.Name &&
		felixConfig.ResourceVersion == felixConfigServer.ResourceVersion {
		return fmt.Errorf("didn't get the same felixConfig back from the server \n%+v\n%+v", felixConfig, felixConfigServer)
	}

	err = felixConfigClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("felixConfig should be deleted (%s)", err)
	}

	return nil
}

// TestKubeControllersConfigurationClient exercises the KubeControllersConfiguration client.
func TestKubeControllersConfigurationClient(t *testing.T) {
	const name = "test-kubecontrollersconfig"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.KubeControllersConfiguration{}
			}, true)
			defer shutdownServer()
			if err := testKubeControllersConfigurationClient(client); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-kubecontrollersconfig test failed")
	}
}

func testKubeControllersConfigurationClient(client calicoclient.Interface) error {
	kubeControllersConfigClient := client.ProjectcalicoV3().KubeControllersConfigurations()
	kubeControllersConfig := &v3.KubeControllersConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Status: calico.KubeControllersConfigurationStatus{
			RunningConfig: calico.KubeControllersConfigurationSpec{
				Controllers: calico.ControllersConfig{
					Node: &calico.NodeControllerConfig{
						SyncLabels: calico.Enabled,
					},
				},
			},
		},
	}
	ctx := context.Background()

	// start from scratch
	kubeControllersConfigs, err := kubeControllersConfigClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing kubeControllersConfigs (%s)", err)
	}
	if kubeControllersConfigs.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	kubeControllersConfigServer, err := kubeControllersConfigClient.Create(ctx, kubeControllersConfig, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the kubeControllersConfig '%v' (%v)", kubeControllersConfig, err)
	}
	if kubeControllersConfigServer.Name != "default" {
		return fmt.Errorf("didn't get the same kubeControllersConfig back from the server \n%+v\n%+v", kubeControllersConfig, kubeControllersConfigServer)
	}
	if !reflect.DeepEqual(kubeControllersConfigServer.Status, calico.KubeControllersConfigurationStatus{}) {
		return fmt.Errorf("status was set on create to %#v", kubeControllersConfigServer.Status)
	}

	kubeControllersConfigs, err = kubeControllersConfigClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing kubeControllersConfigs (%s)", err)
	}
	if len(kubeControllersConfigs.Items) != 1 {
		return fmt.Errorf("expected 1 kubeControllersConfig got %d", len(kubeControllersConfigs.Items))
	}

	kubeControllersConfigServer, err = kubeControllersConfigClient.Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting kubeControllersConfig default (%s)", err)
	}
	if kubeControllersConfigServer.Name != "default" &&
		kubeControllersConfig.ResourceVersion == kubeControllersConfigServer.ResourceVersion {
		return fmt.Errorf("didn't get the same kubeControllersConfig back from the server \n%+v\n%+v", kubeControllersConfig, kubeControllersConfigServer)
	}

	kubeControllersConfigUpdate := kubeControllersConfigServer.DeepCopy()
	kubeControllersConfigUpdate.Spec.HealthChecks = calico.Enabled
	kubeControllersConfigUpdate.Status.EnvironmentVars = map[string]string{"FOO": "bar"}
	kubeControllersConfigServer, err = kubeControllersConfigClient.Update(ctx, kubeControllersConfigUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating kubeControllersConfig default (%s)", err)
	}
	if kubeControllersConfigServer.Spec.HealthChecks != kubeControllersConfigUpdate.Spec.HealthChecks {
		return errors.New("didn't update spec.content")
	}
	if kubeControllersConfigServer.Status.EnvironmentVars != nil {
		return errors.New("status was updated by Update()")
	}

	kubeControllersConfigUpdate = kubeControllersConfigServer.DeepCopy()
	kubeControllersConfigUpdate.Status.EnvironmentVars = map[string]string{"FIZZ": "buzz"}
	kubeControllersConfigUpdate.Labels = map[string]string{"foo": "bar"}
	kubeControllersConfigUpdate.Spec.HealthChecks = ""
	kubeControllersConfigServer, err = kubeControllersConfigClient.UpdateStatus(ctx, kubeControllersConfigUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating kubeControllersConfig default (%s)", err)
	}
	if !reflect.DeepEqual(kubeControllersConfigServer.Status, kubeControllersConfigUpdate.Status) {
		return fmt.Errorf("didn't update status. %v != %v", kubeControllersConfigUpdate.Status, kubeControllersConfigServer.Status)
	}
	if _, ok := kubeControllersConfigServer.Labels["foo"]; ok {
		return fmt.Errorf("updatestatus updated labels")
	}
	if kubeControllersConfigServer.Spec.HealthChecks == "" {
		return fmt.Errorf("updatestatus updated spec")
	}

	err = kubeControllersConfigClient.Delete(ctx, "default", metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("kubeControllersConfig should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().KubeControllersConfigurations().Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching KubeControllersConfigurations (%s)", err)
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

	// Create, then delete KubeControllersConfigurations
	kubeControllersConfigServer, err = kubeControllersConfigClient.Create(ctx, kubeControllersConfig, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the kubeControllersConfig '%v' (%v)", kubeControllersConfig, err)
	}
	err = kubeControllersConfigClient.Delete(ctx, "default", metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("kubeControllersConfig should be deleted (%s)", err)
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

// TestManagedClusterClient exercises the ManagedCluster client.
func TestManagedClusterClient(t *testing.T) {
	const name = "test-managedcluster"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			serverConfig := &TestServerConfig{
				etcdServerList: []string{"http://localhost:2379"},
				emptyObjFunc: func() runtime.Object {
					return &v3.ManagedCluster{}
				},
				enableManagedClusterCreateAPI: true,
				managedClustersCACertPath:     "../ca.crt",
				managedClustersCAKeyPath:      "../ca.key",
				managementClusterAddr:         "example.org:1234",
				applyTigeraLicense:            true,
			}

			client, shutdownServer := customizeFreshApiserverAndClient(t, serverConfig)

			defer shutdownServer()
			if err := testManagedClusterClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}
	ctx := context.Background()

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-managedcluster test failed")
	}

	t.Run(fmt.Sprintf("%s-Create API is disabled", name), func(t *testing.T) {
		serverConfig := &TestServerConfig{
			etcdServerList: []string{"http://localhost:2379"},
			emptyObjFunc: func() runtime.Object {
				return &v3.ManagedCluster{}
			},
			enableManagedClusterCreateAPI: false,
			managedClustersCACertPath:     "../ca.crt",
			managedClustersCAKeyPath:      "../ca.key",
			applyTigeraLicense:            true,
		}

		client, shutdownServer := customizeFreshApiserverAndClient(t, serverConfig)
		defer shutdownServer()

		managedClusterClient := client.ProjectcalicoV3().ManagedClusters()
		managedCluster := &v3.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       calico.ManagedClusterSpec{},
		}
		_, err := managedClusterClient.Create(ctx, managedCluster, metav1.CreateOptions{})

		if err == nil {
			t.Fatal("Expected API to be disabled")
		}
		if !strings.Contains(err.Error(), "ManagementCluster must be configured before adding ManagedClusters") {
			t.Fatalf("Expected API err to indicate that API is disabled. Received: %v", err)
		}
	})
}

func testManagedClusterClient(client calicoclient.Interface, name string) error {
	managedClusterClient := client.ProjectcalicoV3().ManagedClusters()
	managedCluster := &v3.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       calico.ManagedClusterSpec{},
	}

	expectedInitialStatus := calico.ManagedClusterStatus{
		Conditions: []calico.ManagedClusterStatusCondition{
			{
				Status: calico.ManagedClusterStatusValueUnknown,
				Type:   calico.ManagedClusterStatusTypeConnected,
			},
		},
	}
	ctx := context.Background()

	// start from scratch
	managedClusters, err := managedClusterClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing managedClusters (%s)", err)
	}
	if managedClusters.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}

	// ------------------------------------------------------------------------------------------
	managedClusterServer, err := managedClusterClient.Create(ctx, managedCluster, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the managedCluster '%v' (%v)", managedCluster, err)
	}
	if name != managedClusterServer.Name {
		return fmt.Errorf("didn't get the same managedCluster back from the server \n%+v\n%+v", managedCluster, managedClusterServer)
	}
	endpoint := regexp.MustCompile("managementClusterAddr:\\s\"example.org:1234\"")
	ca := regexp.MustCompile("management-cluster\\.crt:\\s\\w+")
	cert := regexp.MustCompile("managed-cluster\\.crt:\\s\\w+")
	key := regexp.MustCompile("managed-cluster\\.key:\\s\\w+")

	if len(managedClusterServer.Spec.InstallationManifest) == 0 {
		return fmt.Errorf("expected installationManifest to be populated when creating "+
			"%s \n%+v", managedCluster.Name, managedClusterServer)
	}

	if endpoint.FindStringIndex(managedClusterServer.Spec.InstallationManifest) == nil {
		return fmt.Errorf("expected installationManifest to contain %s when creating "+
			"%s \n%+v", "managementClusterAddr", managedCluster.Name, managedClusterServer)
	}

	if ca.FindStringIndex(managedClusterServer.Spec.InstallationManifest) == nil {
		return fmt.Errorf("expected installationManifest to contain %s when creating "+
			"%s \n%+v", "management-cluster.crt", managedCluster.Name, managedClusterServer)
	}

	if cert.FindStringIndex(managedClusterServer.Spec.InstallationManifest) == nil {
		return fmt.Errorf("expected installationManifest to contain %s when creating "+
			"%s \n%+v", "managed-cluster.crt", managedCluster.Name, managedClusterServer)
	}

	if key.FindStringIndex(managedClusterServer.Spec.InstallationManifest) == nil {
		return fmt.Errorf("expected installationManifest to contain %s when creating "+
			"%s \n%+v", "managed-cluster.key", managedCluster.Name, managedClusterServer)
	}

	fingerprint := managedClusterServer.ObjectMeta.Annotations["certs.tigera.io/active-fingerprint"]
	if len(fingerprint) == 0 {
		return fmt.Errorf("expected fingerprint when creating %s instead of \n%+v",
			managedCluster.Name, managedClusterServer)
	}

	// ------------------------------------------------------------------------------------------
	managedClusters, err = managedClusterClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing managedClusters (%s)", err)
	}
	if len(managedClusters.Items) != 1 {
		return fmt.Errorf("expected 1 managedCluster got %d", len(managedClusters.Items))
	}

	// ------------------------------------------------------------------------------------------
	managedClusterServer, err = managedClusterClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting managedCluster %s (%s)", name, err)
	}
	if name != managedClusterServer.Name &&
		managedCluster.ResourceVersion == managedClusterServer.ResourceVersion {
		return fmt.Errorf("didn't get the same managedCluster back from the server \n%+v\n%+v", managedCluster, managedClusterServer)
	}
	if !reflect.DeepEqual(managedClusterServer.Status, expectedInitialStatus) {
		return fmt.Errorf("status was set on create to %#v", managedClusterServer.Status)
	}
	if len(managedClusterServer.Spec.InstallationManifest) != 0 {
		return fmt.Errorf("expected installation manifest to be empty after creation instead of \n%+v", managedCluster)
	}
	// ------------------------------------------------------------------------------------------
	managedClusterUpdate := managedClusterServer.DeepCopy()
	managedClusterUpdate.Status.Conditions = []calico.ManagedClusterStatusCondition{
		{
			Message: "Connected to Managed Cluster",
			Reason:  "ConnectionSuccessful",
			Status:  calico.ManagedClusterStatusValueTrue,
			Type:    calico.ManagedClusterStatusTypeConnected,
		},
	}
	managedClusterServer, err = managedClusterClient.Update(ctx, managedClusterUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating managedCluster %s (%s)", name, err)
	}
	if !reflect.DeepEqual(managedClusterServer.Status, managedClusterUpdate.Status) {
		return fmt.Errorf("didn't update status %#v", managedClusterServer.Status)
	}
	// ------------------------------------------------------------------------------------------
	err = managedClusterClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("managedCluster should be deleted (%s)", err)
	}

	// Test watch
	w, err := client.ProjectcalicoV3().ManagedClusters().Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching ManagedClusters (%s)", err)
	}
	var events []watch.Event
	done := sync.WaitGroup{}
	done.Add(1)
	timeout := time.After(1000 * time.Millisecond)
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
		_, err = managedClusterClient.Create(ctx, mc, metav1.CreateOptions{})
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
				return &v3.ClusterInformation{}
			}, true)
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
	ctx := context.Background()

	ci, err := clusterInformationClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ClusterInformation (%s)", err)
	}
	if ci.Items == nil {
		return fmt.Errorf("items field should not be set to nil")
	}

	// Confirm it's not possible to edit the default cluster information.
	info := ci.Items[0]
	info.Spec.CalicoVersion = "fakeVersion"
	_, err = clusterInformationClient.Update(ctx, &info, metav1.UpdateOptions{})
	if err == nil {
		return fmt.Errorf("expected error updating default clusterinformation")
	}

	// Should also not be able to delete it.
	err = clusterInformationClient.Delete(ctx, "default", metav1.DeleteOptions{})
	if err == nil {
		return fmt.Errorf("expected error updating default clusterinformation")
	}

	// Confirm it's not possible to create a clusterInformation obj with name other than "default"
	invalidClusterInfo := &v3.ClusterInformation{ObjectMeta: metav1.ObjectMeta{Name: "test-clusterinformation"}}

	_, err = clusterInformationClient.Create(ctx, invalidClusterInfo, metav1.CreateOptions{})
	if err == nil {
		return fmt.Errorf("expected error creating invalidClusterInfo with name other than \"default\"")
	}

	return nil
}

// TestAuthenticationReviewsClient exercises the AuthenticationReviews client.
func TestAuthenticationReviewsClient(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.AuthenticationReview{}
			}, true)
			defer shutdownServer()
			if err := testAuthenticationReviewsClient(client); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run("test-authentication-reviews", rootTestFunc()) {
		t.Errorf("test-authentication-reviews failed")
	}
}

func testAuthenticationReviewsClient(client calicoclient.Interface) error {
	ar := v3.AuthenticationReview{}
	_, err := client.ProjectcalicoV3().AuthenticationReviews().Create(context.Background(), &ar, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	name := "name"
	groups := []string{name}
	extra := map[string][]string{name: groups}
	uid := "uid"

	ctx := request.NewContext()
	ctx = request.WithUser(ctx, &user.DefaultInfo{
		Name:   name,
		Groups: groups,
		Extra:  extra,
		UID:    uid,
	})

	auth := authenticationreview.NewREST()
	obj, err := auth.Create(ctx, auth.New(), nil, nil)
	if err != nil {
		return err
	}

	if obj == nil {
		return errors.New("expected an authentication review")
	}

	status := obj.(*v3.AuthenticationReview).Status
	if status.Name != name || status.Groups[0] != name || status.UID != uid || status.Extra[name][0] != name {
		return errors.New("unexpected user info from authentication review")
	}
	return nil
}

// TestAuthorizationReviewsClient exercises the AuthorizationReviews client.
func TestAuthorizationReviewsClient(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			pcs, client, shutdownServer := getFreshApiserverServerAndClient(t, func() runtime.Object {
				return &v3.AuthorizationReview{}
			})
			defer shutdownServer()
			if err := testAuthorizationReviewsClient(pcs, client); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !t.Run("test-authorization-reviews", rootTestFunc()) {
		t.Errorf("test-authorization-reviews failed")
	}
}

func testAuthorizationReviewsClient(pcs *apiserver.ProjectCalicoServer, client calicoclient.Interface) error {
	// Check we are able to create the authorization review.
	ar := v3.AuthorizationReview{}
	_, err := client.ProjectcalicoV3().AuthorizationReviews().Create(context.Background(), &ar, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create a user context.
	name := "name"
	groups := []string{name}
	extra := map[string][]string{name: groups}
	uid := "uid"

	ctx := request.NewContext()
	ctx = request.WithUser(ctx, &user.DefaultInfo{
		Name:   name,
		Groups: groups,
		Extra:  extra,
		UID:    uid,
	})

	// Create the authorization review REST backend using the instantiated RBAC helper.
	if pcs.RBACCalculator == nil {
		return fmt.Errorf("No RBAC calc")
	}
	auth := authorizationreview.NewREST(pcs.RBACCalculator)

	// For testing tier permissions.
	tierClient := client.ProjectcalicoV3().Tiers()
	tier := &v3.Tier{
		ObjectMeta: metav1.ObjectMeta{Name: "net-sec"},
	}

	_, err = tierClient.Create(ctx, tier, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to create tier: %v", err)
	}
	defer func() {
		_ = tierClient.Delete(ctx, "net-sec", metav1.DeleteOptions{})
	}()

	// Get the users permissions.
	req := &v3.AuthorizationReview{
		Spec: v3.AuthorizationReviewSpec{
			ResourceAttributes: []v3.AuthorizationReviewResourceAttributes{
				{
					APIGroup:  "",
					Resources: []string{"namespaces"},
					Verbs:     []string{"create", "get"},
				},
				{
					APIGroup:  "",
					Resources: []string{"pods"},
					// Try some duplicates to make sure they are contracted.
					Verbs: []string{"patch", "create", "delete", "patch", "delete"},
				},
			},
		},
	}

	// The user will currently have no permissions, so the returned status should contain an entry for each resource
	// type and verb combination, but contain no match entries for each.
	obj, err := auth.Create(ctx, req, nil, nil)
	if err != nil {
		return fmt.Errorf("Failed to create AuthorizationReview: %v", err)
	}

	if obj == nil {
		return errors.New("expected an AuthorizationReview")
	}

	status := obj.(*v3.AuthorizationReview).Status

	if err := checkAuthorizationReviewStatus(status, v3.AuthorizationReviewStatus{
		AuthorizedResourceVerbs: []v3.AuthorizedResourceVerbs{
			{
				APIGroup: "",
				Resource: "namespaces",
				Verbs:    []v3.AuthorizedResourceVerb{{Verb: "create"}, {Verb: "get"}},
			}, {
				APIGroup: "",
				Resource: "pods",
				Verbs:    []v3.AuthorizedResourceVerb{{Verb: "create"}, {Verb: "delete"}, {Verb: "patch"}},
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func checkAuthorizationReviewStatus(actual, expected v3.AuthorizationReviewStatus) error {
	if reflect.DeepEqual(actual, expected) {
		return nil
	}

	actualBytes, _ := json.Marshal(actual)
	expectedBytes, _ := json.Marshal(expected)

	return fmt.Errorf("Expected status: %s\nActual Status: %s", string(expectedBytes), string(actualBytes))
}

// TestPacketCaptureClient exercises the PacketCaptures client.
func TestPacketCaptureClient(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		const name = "test-packetcapture"
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.PacketCapture{}
			}, true)
			defer shutdownServer()
			if err := testPacketCapturesClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run("test-packet-captures", rootTestFunc()) {
		t.Errorf("test-packet-captures failed")
	}
}

func testPacketCapturesClient(client calicoclient.Interface, name string) error {
	ctx := context.Background()
	err := createEnterprise(client, ctx)
	if err == nil {
		return fmt.Errorf("Could not create a license")
	}

	ns := "default"
	packetCaptureClient := client.ProjectcalicoV3().PacketCaptures(ns)
	packetCapture := &v3.PacketCapture{ObjectMeta: metav1.ObjectMeta{Name: name}}

	// start from scratch
	packetCaptures, err := packetCaptureClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing packetCaptures (%s)", err)
	}
	if packetCaptures.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(packetCaptures.Items) > 0 {
		return fmt.Errorf("packetCaptures should not exist on start, had %v packetCaptures", len(packetCaptures.Items))
	}

	packetCaptureServer, err := packetCaptureClient.Create(ctx, packetCapture, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the packetCapture '%v' (%v)", packetCapture, err)
	}

	updatedPacketCapture := packetCaptureServer.DeepCopy()
	updatedPacketCapture.Labels = map[string]string{"foo": "bar"}
	packetCaptureServer, err = packetCaptureClient.Update(ctx, updatedPacketCapture, metav1.UpdateOptions{})
	if nil != err {
		return fmt.Errorf("error in updating the packetCapture '%v' (%v)", packetCapture, err)
	}

	updatedPacketCaptureWithStatus := packetCaptureServer.DeepCopy()
	updatedPacketCaptureWithStatus.Status = calico.PacketCaptureStatus{
		Files: []calico.PacketCaptureFile{
			{
				Node:      "node",
				FileNames: []string{"file1", "file2"},
			},
		},
	}

	packetCaptureServer, err = packetCaptureClient.UpdateStatus(ctx, updatedPacketCaptureWithStatus, metav1.UpdateOptions{})
	if nil != err {
		return fmt.Errorf("error updating the packetCapture '%v' (%v)", packetCaptureServer, err)
	}
	if !reflect.DeepEqual(packetCaptureServer.Status, updatedPacketCaptureWithStatus.Status) {
		return fmt.Errorf("didn't update status %#v", updatedPacketCaptureWithStatus.Status)
	}

	// Should be listing the packetCapture.
	packetCaptures, err = packetCaptureClient.List(ctx, metav1.ListOptions{})

	if err != nil {
		return fmt.Errorf("error listing packetCaptures (%s)", err)
	}
	if 1 != len(packetCaptures.Items) {
		return fmt.Errorf("should have exactly one packetCapture, had %v packetCaptures", len(packetCaptures.Items))
	}

	packetCaptureServer, err = packetCaptureClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting packetCapture %s (%s)", name, err)
	}
	if name != packetCaptureServer.Name &&
		packetCapture.ResourceVersion == packetCaptureServer.ResourceVersion {
		return fmt.Errorf("didn't get the same packetCapture back from the server \n%+v\n%+v", packetCapture, packetCaptureServer)
	}

	// Watch Test:
	opts := v1.ListOptions{Watch: true}
	wIface, err := packetCaptureClient.Watch(ctx, opts)
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

	err = packetCaptureClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("packetCapture should be deleted (%s)", err)
	}

	wg.Wait()
	return nil
}

// TestDeepPacketInspectionClient exercises the DeepPacketInspection client.
func TestDeepPacketInspectionClient(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		const name = "test-deeppacketinspection"
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.DeepPacketInspection{}
			}, true)
			defer shutdownServer()
			if err := testDeepPacketInspectionClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run("test-deep-packet-inspections", rootTestFunc()) {
		t.Errorf("test-deep-packet-inspections failed")
	}
}

func testDeepPacketInspectionClient(client calicoclient.Interface, name string) error {
	ctx := context.Background()
	err := createEnterprise(client, ctx)
	if err == nil {
		return fmt.Errorf("Could not create a license")
	}

	ns := "default"
	deepPacketInspectionClient := client.ProjectcalicoV3().DeepPacketInspections(ns)
	deepPacketInspection := &v3.DeepPacketInspection{ObjectMeta: metav1.ObjectMeta{Name: name}}

	// start from scratch
	deepPacketInspections, err := deepPacketInspectionClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing deepPacketInspections (%s)", err)
	}
	if deepPacketInspections.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(deepPacketInspections.Items) > 0 {
		return fmt.Errorf("deepPacketInspection should not exist on start, had %v deepPacketInspection", len(deepPacketInspections.Items))
	}

	deepPacketInspectionServer, err := deepPacketInspectionClient.Create(ctx, deepPacketInspection, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the deepPacketInspection '%v' (%v)", deepPacketInspection, err)
	}

	updatedDeepPacketInspection := deepPacketInspectionServer.DeepCopy()
	updatedDeepPacketInspection.Labels = map[string]string{"foo": "bar"}
	updatedDeepPacketInspection.Spec = calico.DeepPacketInspectionSpec{Selector: "k8s-app == 'sample-app'"}
	deepPacketInspectionServer, err = deepPacketInspectionClient.Update(ctx, updatedDeepPacketInspection, metav1.UpdateOptions{})
	if nil != err {
		return fmt.Errorf("error in updating the deepPacketInspection '%v' (%v)", deepPacketInspection, err)
	}
	if !reflect.DeepEqual(deepPacketInspectionServer.Labels, updatedDeepPacketInspection.Labels) {
		return fmt.Errorf("didn't update label %#v", deepPacketInspectionServer.Labels)
	}
	if !reflect.DeepEqual(deepPacketInspectionServer.Spec, updatedDeepPacketInspection.Spec) {
		return fmt.Errorf("didn't update spec %#v", deepPacketInspectionServer.Spec)
	}

	// Should be listing the deepPacketInspection.
	deepPacketInspections, err = deepPacketInspectionClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing deepPacketInspections (%s)", err)
	}
	if 1 != len(deepPacketInspections.Items) {
		return fmt.Errorf("should have exactly one deepPacketInspection, had %v deepPacketInspections", len(deepPacketInspections.Items))
	}

	deepPacketInspectionServer, err = deepPacketInspectionClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting deepPacketInspection %s (%s)", name, err)
	}
	if name != deepPacketInspectionServer.Name &&
		deepPacketInspection.ResourceVersion == deepPacketInspectionServer.ResourceVersion {
		return fmt.Errorf("didn't get the same deepPacketInspection back from the server \n%+v\n%+v", deepPacketInspection, deepPacketInspectionServer)
	}

	// Watch Test:
	opts := v1.ListOptions{Watch: true}
	wIface, err := deepPacketInspectionClient.Watch(ctx, opts)
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

	err = deepPacketInspectionClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("deepPacketInspection should be deleted (%s)", err)
	}

	wg.Wait()
	return nil
}

// TestUISettingsGroupClient exercises the UISettingsGroup client.
func TestUISettingsGroupClient(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		const name = "test-uisettingsgroup"
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.UISettingsGroup{}
			}, true)
			defer shutdownServer()
			if err := testUISettingsGroupClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run("test-uisettingsgroup", rootTestFunc()) {
		t.Errorf("test-uisettingsgroup failed")
	}
}

func testUISettingsGroupClient(client calicoclient.Interface, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := createEnterprise(client, ctx)
	if err == nil {
		return fmt.Errorf("Could not create a license")
	}

	uiSettingsGroupClient := client.ProjectcalicoV3().UISettingsGroups()
	uiSettingsGroup := &v3.UISettingsGroup{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       v3.UISettingsGroupSpec{Description: "this is a settings group"},
	}

	// start from scratch
	uiSettingsGroups, err := uiSettingsGroupClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing uiSettingsGroups (%s)", err)
	}
	if uiSettingsGroups.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(uiSettingsGroups.Items) > 0 {
		return fmt.Errorf("uiSettingsGroup should not exist on start, had %v uiSettingsGroup", len(uiSettingsGroups.Items))
	}

	uiSettingsGroupServer, err := uiSettingsGroupClient.Create(ctx, uiSettingsGroup, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the uiSettingsGroup '%v' (%v)", uiSettingsGroup, err)
	}

	updatedUISettingsGroup := uiSettingsGroupServer.DeepCopy()
	updatedUISettingsGroup.Labels = map[string]string{"foo": "bar"}
	updatedUISettingsGroup.Spec.Description = "updated description"
	uiSettingsGroupServer, err = uiSettingsGroupClient.Update(ctx, updatedUISettingsGroup, metav1.UpdateOptions{})
	if nil != err {
		return fmt.Errorf("error in updating the uiSettingsGroup '%v' (%v)", uiSettingsGroup, err)
	}
	if !reflect.DeepEqual(uiSettingsGroupServer.Labels, updatedUISettingsGroup.Labels) {
		return fmt.Errorf("didn't update label %#v", uiSettingsGroupServer.Labels)
	}
	if !reflect.DeepEqual(uiSettingsGroupServer.Spec, updatedUISettingsGroup.Spec) {
		return fmt.Errorf("didn't update spec %#v", uiSettingsGroupServer.Spec)
	}

	// Should be listing the uiSettingsGroup.
	uiSettingsGroups, err = uiSettingsGroupClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing uiSettingss (%s)", err)
	}
	if len(uiSettingsGroups.Items) != 1 {
		return fmt.Errorf("should have exactly one uiSettingsGroup, had %v uiSettingss", len(uiSettingsGroups.Items))
	}

	uiSettingsGroupServer, err = uiSettingsGroupClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting uiSettingsGroup %s (%s)", name, err)
	}
	if name != uiSettingsGroupServer.Name &&
		uiSettingsGroup.ResourceVersion == uiSettingsGroupServer.ResourceVersion {
		return fmt.Errorf("didn't get the same uiSettingsGroup back from the server \n%+v\n%+v", uiSettingsGroup, uiSettingsGroupServer)
	}

	// Watch Test:
	opts := v1.ListOptions{Watch: true}
	wIface, err := uiSettingsGroupClient.Watch(ctx, opts)
	if nil != err {
		return fmt.Errorf("Error on watch")
	}

	err = uiSettingsGroupClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("uiSettingsGroup should be deleted (%s)", err)
	}

	select {
	case e := <-wIface.ResultChan():
		// Received the watch event.
		fmt.Println("Watch object: ", e)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TestUISettingsClient exercises the UISettings client.
func TestUISettingsClient(t *testing.T) {
	rootTestFunc := func() func(t *testing.T) {
		const name = "test-uisettings"
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.UISettings{}
			}, true)
			defer shutdownServer()
			if err := testUISettingsClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run("test-uisettings", rootTestFunc()) {
		t.Errorf("test-uisettings failed")
	}
}

func testUISettingsClient(client calicoclient.Interface, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := createEnterprise(client, ctx)
	if err == nil {
		return fmt.Errorf("Could not create a license")
	}

	groupName := "groupname-a"
	name = groupName + "." + name
	name2 := groupName + "." + name + ".2"

	uiSettingsClient := client.ProjectcalicoV3().UISettings()
	uiSettingsGroupClient := client.ProjectcalicoV3().UISettingsGroups()
	uiSettings := &v3.UISettings{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			OwnerReferences: []v1.OwnerReference{},
		},
		Spec: v3.UISettingsSpec{
			Group:       groupName,
			Description: "namespace 123",
			View:        nil,
			Layer: &v3.UIGraphLayer{
				Nodes: []v3.UIGraphNode{{
					Type:      "this",
					Name:      "name",
					Namespace: "namespace",
					ID:        "this/namespace/name",
				}},
				Icon: "svg-1",
			},
			Dashboard: nil,
		},
	}
	uiSettingsGroup := &v3.UISettingsGroup{
		ObjectMeta: metav1.ObjectMeta{Name: groupName},
		Spec: v3.UISettingsGroupSpec{
			Description: "my groupName",
		},
	}

	// start from scratch. Listing without specifying the groupName should be fine since we have full access across
	// all groups.
	uiSettingsList, err := uiSettingsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing uiSettings with no group selector (%s)", err)
	}
	if uiSettingsList.Items == nil {
		return fmt.Errorf("Items field should not be set to nil")
	}
	if len(uiSettingsList.Items) > 0 {
		return fmt.Errorf("uiSettingsGroup should not exist on start, had %v uiSettingsGroup", len(uiSettingsList.Items))
	}

	// Listing with the group name will fail because the group does not exist.
	uiSettingsList, err = uiSettingsClient.List(ctx, metav1.ListOptions{FieldSelector: "spec.group=" + groupName})
	if err == nil {
		return fmt.Errorf("expected error listing the uiSettings with group when group does not exist")
	}

	// Attempt to create UISettings without the groupName existing,
	_, err = uiSettingsClient.Create(ctx, uiSettings, metav1.CreateOptions{})
	if err == nil {
		return fmt.Errorf("expected error creating the uiSettings without group")
	}

	// Create a UISettingsGroup.
	uiSettingsGroupServer, err := uiSettingsGroupClient.Create(ctx, uiSettingsGroup, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating the uiSettingsGroup '%v' (%v)", uiSettingsGroup, err)
	}
	defer func() {
		uiSettingsGroupClient.Delete(ctx, groupName, metav1.DeleteOptions{})
	}()

	uiSettingsServer, err := uiSettingsClient.Create(ctx, uiSettings, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating the uiSettings '%v' (%v)", uiSettings, err)
	}
	defer func() {
		uiSettingsClient.Delete(ctx, name, metav1.DeleteOptions{})
	}()
	if len(uiSettingsServer.OwnerReferences) != 1 {
		return fmt.Errorf("expecting OwnerReferences to contain a single entry after create '%v'", uiSettingsServer.OwnerReferences)
	}
	if uiSettingsServer.OwnerReferences[0].Kind != "UISettingsGroup" ||
		uiSettingsServer.OwnerReferences[0].Name != groupName ||
		uiSettingsServer.OwnerReferences[0].APIVersion != "projectcalico.org/v3" ||
		uiSettingsServer.OwnerReferences[0].UID != uiSettingsGroupServer.UID {
		return fmt.Errorf("expecting OwnerReferences be the owning group after create: '%v'", uiSettingsServer.OwnerReferences)
	}
	if len(uiSettingsServer.Spec.User) != 0 {
		return fmt.Errorf("expecting User field not to be filled in: %v", uiSettingsServer.Spec.User)
	}

	// / Try updating without the owner reference. This should fail.
	updatedUISettings := uiSettingsServer.DeepCopy()
	updatedUISettings.Labels = map[string]string{"foo": "bar"}
	updatedUISettings.Spec.Description = "updated description"
	updatedUISettings.OwnerReferences = nil
	_, err = uiSettingsClient.Update(ctx, updatedUISettings, metav1.UpdateOptions{})
	if err == nil {
		return fmt.Errorf("expecting error updating UISettings without the owner reference (%v)", uiSettings)
	}

	// Set the owner references from the Get and try again.
	updatedUISettings.OwnerReferences = uiSettingsServer.OwnerReferences
	uiSettingsServer, err = uiSettingsClient.Update(ctx, updatedUISettings, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error in updating the uiSettings '%v' (%v)", uiSettings, err)
	}
	if !reflect.DeepEqual(uiSettingsServer.Labels, updatedUISettings.Labels) {
		return fmt.Errorf("didn't update label %#v", uiSettingsServer.Labels)
	}
	if !reflect.DeepEqual(uiSettingsServer.Spec, updatedUISettings.Spec) {
		return fmt.Errorf("didn't update spec %#v", uiSettingsServer.Spec)
	}
	if len(uiSettingsServer.OwnerReferences) != 1 {
		return fmt.Errorf("expecting OwnerReferences to contain a single entry after update '%v'", uiSettingsServer.OwnerReferences)
	}
	if uiSettingsServer.OwnerReferences[0].Kind != "UISettingsGroup" ||
		uiSettingsServer.OwnerReferences[0].Name != groupName ||
		uiSettingsServer.OwnerReferences[0].APIVersion != "projectcalico.org/v3" ||
		uiSettingsServer.OwnerReferences[0].UID != uiSettingsGroupServer.UID {
		return fmt.Errorf("expecting OwnerReferences be the owning group after update: '%v'", uiSettingsServer.OwnerReferences)
	}

	// List should include everything if not specifying the group.
	uiSettingsList, err = uiSettingsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing uiSettingss without group selector (%s)", err)
	}
	if len(uiSettingsList.Items) != 1 {
		return fmt.Errorf("should have exactly one uiSettings, had %v uiSettingss", len(uiSettingsList.Items))
	}

	// Should be listing the uiSettings by field selector.
	uiSettingsList, err = uiSettingsClient.List(ctx, metav1.ListOptions{FieldSelector: "spec.group=" + groupName})
	if err != nil {
		return fmt.Errorf("error listing uiSettingss with group selector (%s)", err)
	}
	if len(uiSettingsList.Items) != 1 {
		return fmt.Errorf("should have exactly one uiSettings, had %v uiSettingss", len(uiSettingsList.Items))
	}

	uiSettingsServer, err = uiSettingsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting uiSettings %s (%s)", name, err)
	}
	if name != uiSettingsServer.Name &&
		uiSettings.ResourceVersion == uiSettingsServer.ResourceVersion {
		return fmt.Errorf("didn't get the same uiSettings back from the server \n%+v\n%+v", uiSettings, uiSettingsServer)
	}

	// Modify the group to have the user filter.
	uiSettingsGroupServer.Spec.FilterType = "User"
	uiSettingsGroupServer, err = uiSettingsGroupClient.Update(ctx, uiSettingsGroupServer, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating the uiSettingsGroup '%v' (%v)", uiSettingsGroup, err)
	}

	// Create a second group that should be tied to the user.
	uiSettings.Name = name2
	uiSettingsServer2, err := uiSettingsClient.Create(ctx, uiSettings, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating the second uiSettings '%v' (%v)", uiSettings, err)
	}
	defer func() {
		uiSettingsClient.Delete(ctx, name2, metav1.DeleteOptions{})
	}()
	if len(uiSettingsServer2.Spec.User) == 0 {
		return fmt.Errorf("expecting User field to be filled in")
	}

	// List UISettings without group. This should return both settings.
	uiSettingsList, err = uiSettingsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing uiSettingss without group selector (%s)", err)
	}
	if len(uiSettingsList.Items) != 2 {
		return fmt.Errorf("should have exactly two uiSettings, had %v uiSettingss", len(uiSettingsList.Items))
	}

	// List UISettings with field selector shouold limit to user specific settings now.
	uiSettingsList, err = uiSettingsClient.List(ctx, metav1.ListOptions{FieldSelector: "spec.group=" + groupName})
	if err != nil {
		return fmt.Errorf("error listing uiSettingss with group selector (%s)", err)
	}
	if len(uiSettingsList.Items) != 1 {
		return fmt.Errorf("should have exactly one uiSettings, had %v uiSettingss", len(uiSettingsList.Items))
	}
	if uiSettingsList.Items[0].Name != name2 {
		return fmt.Errorf("should have received %v, instead received %v", name2, uiSettingsList.Items[0].Name)
	}

	// Watch Test. Deleting the second should work.
	opts := v1.ListOptions{Watch: true, FieldSelector: "spec.group=" + groupName}
	wIface, err := uiSettingsClient.Watch(ctx, opts)
	if err != nil {
		return fmt.Errorf("Error on watch")
	}

	err = uiSettingsClient.Delete(ctx, name2, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("uiSettings should be deleted (%s)", err)
	}

	select {
	case e := <-wIface.ResultChan():
		// Received the watch event.
		fmt.Println("Watch object: ", e)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TestCalicoNodeStatusClient exercises the CalicoNodeStatus client.
func TestCalicoNodeStatusClient(t *testing.T) {
	const name = "test-caliconodestatus"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.CalicoNodeStatus{}
			}, true)
			defer shutdownServer()
			if err := testCalicoNodeStatusClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-caliconodestatus test failed")
	}
}

func testCalicoNodeStatusClient(client calicoclient.Interface, name string) error {
	seconds := uint32(11)
	caliconodestatusClient := client.ProjectcalicoV3().CalicoNodeStatuses()
	caliconodestatus := &v3.CalicoNodeStatus{
		ObjectMeta: metav1.ObjectMeta{Name: name},

		Spec: v3.CalicoNodeStatusSpec{
			Node: "node1",
			Classes: []v3.NodeStatusClassType{
				v3.NodeStatusClassTypeAgent,
				v3.NodeStatusClassTypeBGP,
				v3.NodeStatusClassTypeRoutes,
			},
			UpdatePeriodSeconds: &seconds,
		},
	}
	ctx := context.Background()

	// start from scratch
	caliconodestatuses, err := caliconodestatusClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing caliconodestatuses (%s)", err)
	}
	if caliconodestatuses.Items == nil {
		return fmt.Errorf("items field should not be set to nil")
	}

	caliconodestatusNew, err := caliconodestatusClient.Create(ctx, caliconodestatus, metav1.CreateOptions{})
	if nil != err {
		return fmt.Errorf("error creating the object '%v' (%v)", caliconodestatus, err)
	}
	if name != caliconodestatusNew.Name {
		return fmt.Errorf("didn't get the same object back from the server \n%+v\n%+v", caliconodestatus, caliconodestatusNew)
	}

	caliconodestatusNew, err = caliconodestatusClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting object %s (%s)", name, err)
	}

	err = caliconodestatusClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil != err {
		return fmt.Errorf("object should be deleted (%s)", err)
	}

	return nil
}

// TestIPAMConfigClient exercises the IPAMConfig client.
func TestIPAMConfigClient(t *testing.T) {
	const name = "test-ipamconfig"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.IPAMConfiguration{}
			}, false)
			defer shutdownServer()
			if err := testIPAMConfigClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-ipamconfig test failed")
	}
}

func testIPAMConfigClient(client calicoclient.Interface, name string) error {
	ipamConfigClient := client.ProjectcalicoV3().IPAMConfigurations()
	ipamConfig := &v3.IPAMConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: name},

		Spec: v3.IPAMConfigurationSpec{
			StrictAffinity:   true,
			MaxBlocksPerHost: 28,
		},
	}
	ctx := context.Background()

	_, err := ipamConfigClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing IPAMConfigurations: %s", err)
	}

	ipamConfigNew, err := ipamConfigClient.Create(ctx, ipamConfig, metav1.CreateOptions{})
	if err == nil {
		return fmt.Errorf("should not be able to create ipam config %s ", ipamConfig.Name)
	}

	ipamConfig.Name = "default"
	ipamConfigNew, err = ipamConfigClient.Create(ctx, ipamConfig, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating the object '%v' (%v)", ipamConfig, err)
	}

	if ipamConfigNew.Name != ipamConfig.Name {
		return fmt.Errorf("didn't get the same object back from the server \n%+v\n%+v", ipamConfig, ipamConfigNew)
	}

	if ipamConfigNew.Spec.StrictAffinity != true || ipamConfig.Spec.MaxBlocksPerHost != 28 {
		return fmt.Errorf("didn't get the correct object back from the server \n%+v\n%+v", ipamConfig, ipamConfigNew)
	}

	ipamConfigNew, err = ipamConfigClient.Get(ctx, ipamConfig.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting object %s (%s)", ipamConfig.Name, err)
	}

	ipamConfigNew.Spec.StrictAffinity = false
	ipamConfigNew.Spec.MaxBlocksPerHost = 0

	_, err = ipamConfigClient.Update(ctx, ipamConfigNew, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating object %s (%s)", name, err)
	}

	ipamConfigUpdated, err := ipamConfigClient.Get(ctx, ipamConfig.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting object %s (%s)", ipamConfig.Name, err)
	}

	if ipamConfigUpdated.Spec.StrictAffinity != false || ipamConfigUpdated.Spec.MaxBlocksPerHost != 0 {
		return fmt.Errorf("didn't get the correct object back from the server \n%+v\n%+v", ipamConfigUpdated, ipamConfigNew)
	}

	err = ipamConfigClient.Delete(ctx, ipamConfig.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("object should be deleted (%s)", err)
	}

	return nil
}

// TestBlockAffinityClient exercises the BlockAffinity client.
func TestBlockAffinityClient(t *testing.T) {
	const name = "test-blockaffinity"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.BlockAffinity{}
			}, true)
			defer shutdownServer()
			if err := testBlockAffinityClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-blockaffinity test failed")
	}
}

func testBlockAffinityClient(client calicoclient.Interface, name string) error {
	blockAffinityClient := client.ProjectcalicoV3().BlockAffinities()
	blockAffinity := &v3.BlockAffinity{
		ObjectMeta: metav1.ObjectMeta{Name: name},

		Spec: v3.BlockAffinitySpec{
			CIDR:  "10.0.0.0/24",
			Node:  "node1",
			State: "pending",
		},
	}
	libV3BlockAffinity := &libapiv3.BlockAffinity{
		ObjectMeta: metav1.ObjectMeta{Name: name},

		Spec: libapiv3.BlockAffinitySpec{
			CIDR:    "10.0.0.0/24",
			Node:    "node1",
			State:   "pending",
			Deleted: "false",
		},
	}
	ctx := context.Background()

	// Calico libv3 client instantiation in order to get around the API create restrictions
	// TODO: Currently these tests only run on a Kubernetes datastore since profile creation
	// does not work in etcd. Figure out how to divide this configuration to etcd once that
	// is fixed.
	config := apiconfig.NewCalicoAPIConfig()
	config.Spec = apiconfig.CalicoAPIConfigSpec{
		DatastoreType: apiconfig.Kubernetes,
		EtcdConfig: apiconfig.EtcdConfig{
			EtcdEndpoints: "http://localhost:2379",
		},
		KubeConfig: apiconfig.KubeConfig{
			Kubeconfig: os.Getenv("KUBECONFIG"),
		},
	}
	apiClient, err := libclient.New(*config)
	if err != nil {
		return fmt.Errorf("unable to create Calico lib v3 client: %s", err)
	}

	_, err = blockAffinityClient.Create(ctx, blockAffinity, metav1.CreateOptions{})
	if err == nil {
		return fmt.Errorf("should not be able to create block affinity %s ", blockAffinity.Name)
	}

	// Create the block affinity using the libv3 client.
	_, err = apiClient.BlockAffinities().Create(ctx, libV3BlockAffinity, options.SetOptions{})
	if err != nil {
		return fmt.Errorf("error creating the object through the Calico v3 API '%v' (%v)", libV3BlockAffinity, err)
	}

	blockAffinityNew, err := blockAffinityClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting object %s (%s)", name, err)
	}

	blockAffinityList, err := blockAffinityClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing BlockAffinity (%s)", err)
	}
	if blockAffinityList.Items == nil {
		return fmt.Errorf("items field should not be set to nil")
	}

	blockAffinityNew.Spec.State = "confirmed"

	_, err = blockAffinityClient.Update(ctx, blockAffinityNew, metav1.UpdateOptions{})
	if err == nil {
		return fmt.Errorf("should not be able to update block affinity %s", blockAffinityNew.Name)
	}

	err = blockAffinityClient.Delete(ctx, name, metav1.DeleteOptions{})
	if nil == err {
		return fmt.Errorf("should not be able to delete block affinity %s", blockAffinity.Name)
	}

	// Test watch
	w, err := blockAffinityClient.Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error watching block affinities (%s)", err)
	}

	_, err = apiClient.BlockAffinities().Delete(ctx, name, options.DeleteOptions{ResourceVersion: blockAffinityNew.ResourceVersion})
	if err != nil {
		return fmt.Errorf("error deleting the object through the Calico v3 API '%v' (%v)", name, err)
	}

	// Verify watch
	var events []watch.Event
	timeout := time.After(500 * time.Millisecond)
	var timeoutErr error
	// watch for 2 events
	for i := 0; i < 2; i++ {
		select {
		case e := <-w.ResultChan():
			events = append(events, e)
		case <-timeout:
			timeoutErr = fmt.Errorf("timed out waiting for events")
			break
		}
	}
	if timeoutErr != nil {
		return timeoutErr
	}
	if len(events) != 2 {
		return fmt.Errorf("expected 2 watch events got %d", len(events))
	}

	return nil
}

// TestBGPFilterClient exercises the BGPFilter client.
func TestBGPFilterClient(t *testing.T) {
	const name = "test-bgpfilter"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.BGPFilter{}
			}, false)
			defer shutdownServer()
			if err := testBGPFilterClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-bgpfilter test failed")
	}
}

func testBGPFilterClient(client calicoclient.Interface, name string) error {
	bgpFilterClient := client.ProjectcalicoV3().BGPFilters()
	acceptRuleV4 := v3.BGPFilterRuleV4{
		CIDR:          "10.10.10.0/24",
		MatchOperator: v3.In,
		Action:        v3.Accept,
	}
	rejectRuleV4 := v3.BGPFilterRuleV4{
		CIDR:          "11.11.11.0/24",
		MatchOperator: v3.NotIn,
		Action:        v3.Reject,
	}
	acceptRuleV6 := v3.BGPFilterRuleV6{
		CIDR:          "dead:beef:1::/64",
		MatchOperator: v3.Equal,
		Action:        v3.Accept,
	}
	rejectRuleV6 := v3.BGPFilterRuleV6{
		CIDR:          "dead:beef:2::/64",
		MatchOperator: v3.NotEqual,
		Action:        v3.Reject,
	}
	bgpFilter := &v3.BGPFilter{
		ObjectMeta: metav1.ObjectMeta{Name: name},

		Spec: v3.BGPFilterSpec{
			ExportV4: []v3.BGPFilterRuleV4{acceptRuleV4},
			ImportV4: []v3.BGPFilterRuleV4{rejectRuleV4},
			ExportV6: []v3.BGPFilterRuleV6{acceptRuleV6},
			ImportV6: []v3.BGPFilterRuleV6{rejectRuleV6},
		},
	}
	ctx := context.Background()

	_, err := bgpFilterClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing BGPFilters: %s", err)
	}

	bgpFilterNew, err := bgpFilterClient.Create(ctx, bgpFilter, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating the object '%v' (%v)", bgpFilter, err)
	}

	if bgpFilterNew.Name != bgpFilter.Name {
		return fmt.Errorf("didn't get the same object back from the server \n%+v\n%+v", bgpFilter, bgpFilterNew)
	}

	if len(bgpFilterNew.Spec.ExportV4) != 1 || bgpFilterNew.Spec.ExportV4[0] != bgpFilter.Spec.ExportV4[0] || len(bgpFilterNew.Spec.ImportV4) != 1 || bgpFilterNew.Spec.ImportV4[0] != bgpFilter.Spec.ImportV4[0] || len(bgpFilterNew.Spec.ExportV6) != 1 || bgpFilterNew.Spec.ExportV6[0] != bgpFilter.Spec.ExportV6[0] || len(bgpFilterNew.Spec.ImportV6) != 1 || bgpFilterNew.Spec.ImportV6[0] != bgpFilter.Spec.ImportV6[0] {
		return fmt.Errorf("didn't get the correct object back from the server \n%+v\n%+v", bgpFilter, bgpFilterNew)
	}

	bgpFilterNew, err = bgpFilterClient.Get(ctx, bgpFilter.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting object %s (%s)", bgpFilter.Name, err)
	}

	bgpFilterNew.Spec.ExportV4 = nil

	_, err = bgpFilterClient.Update(ctx, bgpFilterNew, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating object %s (%s)", name, err)
	}

	bgpFilterUpdated, err := bgpFilterClient.Get(ctx, bgpFilter.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting object %s (%s)", bgpFilter.Name, err)
	}

	if bgpFilterUpdated.Spec.ExportV4 != nil {
		return fmt.Errorf("didn't get the correct object back from the server \n%+v\n%+v", bgpFilterUpdated, bgpFilterNew)
	}

	err = bgpFilterClient.Delete(ctx, bgpFilter.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("object should be deleted (%s)", err)
	}

	return nil
}

// TestExternalNetworkClient exercises the ExternalNetwork client.
func TestExternalNetworkClient(t *testing.T) {
	const name = "test-externalnetwork"
	rootTestFunc := func() func(t *testing.T) {
		return func(t *testing.T) {
			client, shutdownServer := getFreshApiserverAndClient(t, func() runtime.Object {
				return &v3.ExternalNetwork{}
			}, false)
			defer shutdownServer()
			if err := testExternalNetworkClient(client, name); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !t.Run(name, rootTestFunc()) {
		t.Errorf("test-externalnetwork test failed")
	}
}

func testExternalNetworkClient(client calicoclient.Interface, name string) error {
	externalNetworkClient := client.ProjectcalicoV3().ExternalNetworks()
	index := uint32(28)
	externalNetwork := &v3.ExternalNetwork{
		ObjectMeta: metav1.ObjectMeta{Name: name},

		Spec: v3.ExternalNetworkSpec{
			RouteTableIndex: &index,
		},
	}
	ctx := context.Background()

	_, err := externalNetworkClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing ExternalNetworks: %s", err)
	}

	externalNetworkNew, err := externalNetworkClient.Create(ctx, externalNetwork, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating the object '%v' (%v)", externalNetwork, err)
	}

	if externalNetworkNew.Name != externalNetwork.Name {
		return fmt.Errorf("didn't get the same object back from the server \n%+v\n%+v", externalNetwork, externalNetworkNew)
	}

	if *externalNetwork.Spec.RouteTableIndex != 28 {
		return fmt.Errorf("didn't get the correct object back from the server \n%+v\n%+v", externalNetwork, externalNetworkNew)
	}

	externalNetworkNew, err = externalNetworkClient.Get(ctx, externalNetwork.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting object %s (%s)", externalNetwork.Name, err)
	}

	index = 10
	externalNetworkNew.Spec.RouteTableIndex = &index

	_, err = externalNetworkClient.Update(ctx, externalNetworkNew, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating object %s (%s)", name, err)
	}

	externalNetworkUpdated, err := externalNetworkClient.Get(ctx, externalNetwork.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting object %s (%s)", externalNetwork.Name, err)
	}

	if *externalNetworkUpdated.Spec.RouteTableIndex != 10 {
		return fmt.Errorf("didn't get the correct object back from the server \n%+v\n%+v", externalNetworkUpdated, externalNetworkNew)
	}

	err = externalNetworkClient.Delete(ctx, externalNetwork.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("object should be deleted (%s)", err)
	}

	return nil
}
