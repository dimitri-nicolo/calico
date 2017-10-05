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

package calico

import (
	"fmt"
	"strings"
)

type errInvalidKey struct {
	k string
}

func (e errInvalidKey) Error() string {
	return fmt.Sprintf("invalid key '%s'", e.k)
}

// NamespaceAndNameFromKey returns the namespace and name for a given k8s etcd-style key.
// This function is intended to be used in a Calico based storage.Interface.
//
// The first return value is the namespace. The second return value is the name and will
// not be empty if the error is nil. The error will be non-nil if the key was malformed,
// in which case all other return values will be empty strings.
//
// Example Namespaced resource key: projectcalico.org/networkpolicies/default/my-first-policy
// OR projectcalico.org/globalpolicies/my-first-policy
func NamespaceAndNameFromKey(key string) (string, string, error) {
	spl := strings.Split(key, "/")
	splLen := len(spl)

	if splLen == 2 {
		// slice has neither name nor namespace
		return "", "", nil
	} else if splLen == 3 {
		// slice has namespace and no name
		return spl[2], "", nil
	} else if splLen == 4 {
		// slice has namespace and name
		return spl[2], spl[3], nil
	}

	return "", "", errInvalidKey{k: key}
}

func NameFromKey(key string, hasNamespace bool) (string, error) {
	spl := strings.Split(key, "/")
	splLen := len(spl)

	if splLen == 2 {
		// slice has no name
		return "", nil
	} else if splLen == 3 {
		// slice has name
		return spl[2], nil
	}

	return "", errInvalidKey{k: key}
}
