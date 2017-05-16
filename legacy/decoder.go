/*
Copyright 2016 The Kubernetes Authors.

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

package legacy

import (
	"encoding/json"
	"strings"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimejson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/pkg/api"
)

// Decoder to amend and read legacy structured libcalico
// data.
type Decoder struct {
	*runtimejson.Serializer
}

func NewDecoder() runtime.Decoder {
	serializer := runtimejson.NewSerializer(runtimejson.DefaultMetaFactory, api.Scheme, api.Scheme, false)
	return &Decoder{serializer}
}

// Decode amends data if needed and returns the rest of the Decoded data.
func (d *Decoder) Decode(originalData []byte, gvk *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	_, isPolicy := into.(*calico.Policy)
	if isPolicy {
		policyParse := string(originalData[:])
		if strings.Contains(policyParse, "creationTimestamp") != true {
			var m model.Policy
			err := json.Unmarshal(originalData, &m)
			if err != nil {
				return nil, nil, err
			}
			// regroup the message to make it decodable into into.
			var out calico.Policy
			out.Spec = m
			originalData, err = json.Marshal(out)
			if err != nil {
				return nil, nil, err
			}
		}
	}                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     
	return d.Serializer.Decode(originalData, gvk, into)
}
