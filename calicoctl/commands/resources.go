// Copyright (c) 2016,2019-2020 Tigera, Inc. All rights reserved.

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
// limitations under the License.

package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	licClient "github.com/tigera/licensing/client"

	"github.com/projectcalico/calicoctl/calicoctl/commands/argutils"
	"github.com/projectcalico/calicoctl/calicoctl/commands/clientmgr"
	"github.com/projectcalico/calicoctl/calicoctl/resourcemgr"
	yaml "github.com/projectcalico/go-yaml-wrapper"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	calicoErrors "github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/options"
)

type action int

const (
	actionApply action = iota
	actionCreate
	actionUpdate
	actionDelete
	actionGetOrList
	actionPatch
)

// Convert loaded resources to a slice of resources for easier processing.
// The loaded resources may be a slice containing resources and resource lists, or
// may be a single resource or a single resource list.  This function handles the
// different possible options to convert to a single slice of resources.
func convertToSliceOfResources(loaded interface{}) ([]resourcemgr.ResourceObject, error) {
	res := []resourcemgr.ResourceObject{}
	log.Infof("Converting resource to slice: %v", loaded)
	switch r := loaded.(type) {
	case []runtime.Object:
		for i := 0; i < len(r); i++ {
			r, err := convertToSliceOfResources(r[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r...)
		}

	case resourcemgr.ResourceObject:
		res = append(res, r)

	case resourcemgr.ResourceListObject:
		ret, err := meta.ExtractList(r)
		if err != nil {
			return nil, err
		}

		for _, v := range ret {
			res = append(res, v.(resourcemgr.ResourceObject))
		}
	}

	log.Infof("Returning slice: %v", res)
	return res, nil
}

// commandResults contains the results from executing a CLI command
type commandResults struct {
	// Whether the input file was invalid.
	fileInvalid bool

	// The number of resources that are being configured.
	numResources int

	// The number of resources that were actually configured.  This will
	// never be 0 without an associated error.
	numHandled int

	// The associated error.
	err error

	// The single type of resource that is being configured, or blank
	// if multiple resource types are being configured in a single shot.
	singleKind string

	// The results returned from each invocation
	resources []runtime.Object

	// Errors associated with individual resources
	resErrs []error

	// The Calico API client used for the requests (useful if required
	// again).
	client client.Interface
}

// executeConfigCommand is main function called by all of the resource management commands
// in calicoctl (apply, create, replace, get, delete and patch).  This provides common function
// for all these commands:
// 	-  Load resources from file (or if not specified determine the resource from
// 	   the command line options).
// 	-  Convert the loaded resources into a list of resources (easier to handle)
// 	-  Process each resource individually, fanning out to the appropriate methods on
//	   the client interface, collate results and exit on the first error.
func executeConfigCommand(args map[string]interface{}, action action) commandResults {
	var resources []resourcemgr.ResourceObject

	singleKind := false

	log.Info("Executing config command")

	if filename := args["--filename"]; filename != nil {
		// Filename is specified, load the resource from file and convert to a slice
		// of resources for easier handling.
		r, err := resourcemgr.CreateResourcesFromFile(filename.(string))
		if err != nil {
			return commandResults{err: err, fileInvalid: true}
		}

		resources, err = convertToSliceOfResources(r)
		if err != nil {
			return commandResults{err: err}
		}
	} else {
		// Filename is not specific so extract the resource from the arguments. This
		// is only useful for delete, get and patch functions - but we don't need to check that
		// here since the command syntax requires a filename for the other resource
		// management commands.
		var err error
		singleKind = true
		resources, err = resourcemgr.GetResourcesFromArgs(args)
		if err != nil {
			return commandResults{err: err}
		}
	}

	if len(resources) == 0 {
		return commandResults{err: errors.New("no resources specified")}
	}

	if log.GetLevel() >= log.DebugLevel {
		for _, v := range resources {
			log.Debugf("Resource: %s", v.GetObjectKind().GroupVersionKind().String())
		}

		d, err := yaml.Marshal(resources)
		if err != nil {
			return commandResults{err: err}
		}
		log.Debugf("Data: %s", string(d))
	}

	// Load the client config and connect.
	cf := args["--config"].(string)
	client, err := clientmgr.NewClient(cf)
	if err != nil {
		fmt.Printf("Failed to create Calico API client: %s\n", err)
		os.Exit(1)
	}
	log.Infof("Client: %v", client)

	// Initialise the command results with the number of resources and the name of the
	// kind of resource (if only dealing with a single resource).
	results := commandResults{client: client}
	var kind string
	count := make(map[string]int)
	for _, r := range resources {
		kind = r.GetObjectKind().GroupVersionKind().Kind
		count[kind] = count[kind] + 1
		results.numResources = results.numResources + 1
	}
	if len(count) == 1 || singleKind {
		results.singleKind = kind
	}

	// For commands that modify config, first attempt to initialize the datastore.
	switch action {
	case actionApply, actionCreate, actionUpdate:
		tryEnsureInitialized(context.Background(), client)
	}

	// Now execute the command on each resource in order, exiting as soon as we hit an
	// error.
	export := argutils.ArgBoolOrFalse(args, "--export")
	nameSpecified := false
	emptyName := false
	switch a := args["<NAME>"].(type) {
	case string:
		nameSpecified = len(a) > 0
		_, ok := args["<NAME>"]
		emptyName = !ok || !nameSpecified
	case []string:
		nameSpecified = len(a) > 0
		for _, v := range a {
			if v == "" {
				emptyName = true
			}
		}
	}

	if emptyName {
		return commandResults{err: fmt.Errorf("resource name may not be empty")}
	}

	for _, r := range resources {
		res, err := executeResourceAction(args, client, r, action)
		if err != nil {
			switch action {
			case actionApply, actionCreate, actionDelete, actionGetOrList:
				results.resErrs = append(results.resErrs, err)
				continue
			default:
				results.err = err
			}
		}

		// Remove the cluster specific metadata if the "--export" flag is specified
		// Skip removing cluster specific metadata if this is is called as a "list"
		// operation (no specific name is specified).
		if export && nameSpecified {
			for i := range res {
				rom := res[i].(v1.ObjectMetaAccessor).GetObjectMeta()
				rom.SetNamespace("")
				rom.SetUID("")
				rom.SetResourceVersion("")
				rom.SetCreationTimestamp(v1.Time{})
				rom.SetDeletionTimestamp(nil)
				rom.SetDeletionGracePeriodSeconds(nil)
				rom.SetClusterName("")
			}
		}

		results.resources = append(results.resources, res...)
		results.numHandled = results.numHandled + len(res)
	}

	return results
}

// executeResourceAction fans out the specific resource action to the appropriate method
// on the ResourceManager for the specific resource.
func executeResourceAction(args map[string]interface{}, client client.Interface, resource resourcemgr.ResourceObject, action action) ([]runtime.Object, error) {
	ctx := context.Background()

	// If the resource is Tier/TierList or RemoteClusterConfiguration/RemoteClusterConfigurationList then check the
	// license first then continue only if the license is valid.
	resGVK := resource.GetObjectKind().GroupVersionKind().String()
	if resGVK == api.NewTier().GroupVersionKind().String() || resGVK == api.NewTierList().GroupVersionKind().String() ||
		resGVK == api.NewRemoteClusterConfiguration().GroupVersionKind().String() || resGVK == api.NewRemoteClusterConfigurationList().GroupVersionKind().String() {

		lic, err := client.LicenseKey().Get(ctx, "default", options.GetOptions{ResourceVersion: ""})
		if err != nil {
			// License not found (not applied) or datastore down.
			switch err.(type) {
			case calicoErrors.ErrorResourceDoesNotExist:
				return nil, fmt.Errorf("not licensed for this feature. No valid license was found for your environment. Contact Tigera support or email licensing@tigera.io")
			default:
				// For any other error - datastore error, unauthorized, etc., we just return the error as-is.
				return nil, err
			}
		} else {
			log.Info("License resource found")
		}

		claims, err := licClient.Decode(*lic)
		if err != nil {
			// This means the license is there but is corrupted or has been messed with.
			return nil, fmt.Errorf("license is corrupted. Please contact Tigera support or email licensing@tigera.io")
		}

		if licStatus := claims.Validate(); licStatus != licClient.Valid {
			// If the license is expired (but within grace period) then show this warning banner, but continue to work.
			// in CNX v2.1, grace period is infinite.
			fmt.Println("[WARNING] Your license has expired. Please update your license to restore normal operations.")
			fmt.Println("Contact Tigera support or email licensing@tigera.io")
			fmt.Println()
		} else {
			log.Info("License is valid")
		}
	}

	rm := resourcemgr.GetResourceManager(resource)

	err := handleNamespace(resource, rm, args)
	if err != nil {
		return nil, err
	}

	var resOut runtime.Object
	ctx = context.Background()

	switch action {
	case actionApply:
		resOut, err = rm.Apply(ctx, client, resource)
	case actionCreate:
		resOut, err = rm.Create(ctx, client, resource)
	case actionUpdate:
		resOut, err = rm.Update(ctx, client, resource)
	case actionDelete:
		resOut, err = rm.Delete(ctx, client, resource)
	case actionGetOrList:
		resOut, err = rm.GetOrList(ctx, client, resource)
	case actionPatch:
		patch := args["--patch"].(string)
		resOut, err = rm.Patch(ctx, client, resource, patch)
	}

	// Skip over some errors depending on command line options.
	if err != nil {
		skip := false
		switch err.(type) {
		case calicoErrors.ErrorResourceAlreadyExists:
			skip = argutils.ArgBoolOrFalse(args, "--skip-exists")
		case calicoErrors.ErrorResourceDoesNotExist:
			skip = argutils.ArgBoolOrFalse(args, "--skip-not-exists")
		}
		if skip {
			resOut = resource
			err = nil
		}
	}

	return []runtime.Object{resOut}, err
}

// tryEnsureInitialized is called from any write action (apply, create, update). This
// attempts to initialize the datastore. We do not fail the user action if this fails
// since the users access permissions may be restricted to only allow modification
// of certain resource types.
func tryEnsureInitialized(ctx context.Context, client client.Interface) {
	if err := client.EnsureInitialized(ctx, "", "", ""); err != nil {
		log.WithError(err).Info("Unable to initialize datastore")
	}
}

// handleNamespace fills in the namespace information in the resource (if required),
// and validates the namespace depending on whether or not a namespace should be
// provided based on the resource kind.
func handleNamespace(resource resourcemgr.ResourceObject, rm resourcemgr.ResourceManager, args map[string]interface{}) error {
	allNs := argutils.ArgBoolOrFalse(args, "--all-namespaces")
	cliNs := argutils.ArgStringOrBlank(args, "--namespace")
	resNs := resource.GetObjectMeta().GetNamespace()

	if rm.IsNamespaced() {
		switch {
		case allNs && cliNs != "":
			// Check if --namespace and --all-namespaces flags are used together.
			return fmt.Errorf("cannot use both --namespace and --all-namespaces flags at the same time")
		case resNs == "" && cliNs != "":
			// If resource doesn't have a namespace specified
			// but it's passed in through the -n flag then use that one.
			resource.GetObjectMeta().SetNamespace(cliNs)
		case resNs != "" && allNs:
			// If --all-namespaces is used then we must set namespace to "" so
			// list operation can list resources from all the namespaces.
			resource.GetObjectMeta().SetNamespace("")
		case resNs == "" && allNs:
			// no-op
		case resNs == "" && cliNs == "" && !allNs:
			// Set the namespace to "default" if not specified.
			resource.GetObjectMeta().SetNamespace("default")
		case resNs != "" && cliNs == "":
			// Use the namespace specified in the resource, which is already set.
		case resNs != cliNs:
			// If both resource and the CLI pass in the namespace but they don't match then return an error.
			return fmt.Errorf("resource namespace does not match client namespace. %s != %s", resNs, cliNs)
		}
	} else if resNs != "" {
		return fmt.Errorf("namespace should not be specified for a non-namespaced resource. %s is not a namespaced resource",
			resource.GetObjectKind().GroupVersionKind().Kind)
	} else if allNs || cliNs != "" {
		return fmt.Errorf("%s is not namespaced", resource.GetObjectKind().GroupVersionKind().Kind)
	}

	return nil
}
