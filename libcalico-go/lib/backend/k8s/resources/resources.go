// Copyright (c) 2017-2020,2022 Tigera, Inc. All rights reserved.

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

package resources

import (
	"strings"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/json"
)

const (
	labelsAnnotation      = "projectcalico.org/labels"
	annotationsAnnotation = "projectcalico.org/annotations"
	metadataAnnotation    = "projectcalico.org/metadata"
)

// Interface that all Kubernetes and Calico resources implement.
type Resource interface {
	runtime.Object
	metav1.ObjectMetaAccessor
}

// Interface that all Kubernetes and Calico resource lists implement.
type ResourceList interface {
	runtime.Object
	metav1.ListMetaAccessor
}

// Function signature for conversion function to convert a K8s resource to a
// KVPair equivalent.
type ConvertK8sResourceToKVPair func(Resource) (*model.KVPair, error)
type ConvertK8sResourceToKVPairs func(Resource) ([]*model.KVPair, error)

// ConvertK8sResourceOneToOneAdapter converts a ConvertK8sResourceToKVPair function to a ConvertK8sResourceToKVPairs function
func ConvertK8sResourceOneToOneAdapter(oneToOne ConvertK8sResourceToKVPair) ConvertK8sResourceToKVPairs {
	return func(r Resource) ([]*model.KVPair, error) {
		kvp, err := oneToOne(r)
		if err != nil {
			return nil, err
		} else if kvp != nil {
			return []*model.KVPair{kvp}, nil
		}

		return nil, nil
	}
}

// Store Calico Metadata in the k8s resource annotations for non-CRD backed resources.
// Currently this just stores Annotations and Labels and drops all other metadata
// attributes.
func SetK8sAnnotationsFromCalicoMetadata(k8sRes Resource, calicoRes Resource) {
	a := k8sRes.GetObjectMeta().GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	if labels := calicoRes.GetObjectMeta().GetLabels(); len(labels) > 0 {
		if lann, err := json.Marshal(labels); err != nil {
			log.WithError(err).Warning("unable to store labels as an annotation")
		} else {
			a[labelsAnnotation] = string(lann)
		}
	} else {
		// There are no Calico labels - nil out the k8s res.
		delete(a, labelsAnnotation)
	}
	if annotations := calicoRes.GetObjectMeta().GetAnnotations(); len(annotations) > 0 {
		if aann, err := json.Marshal(annotations); err != nil {
			log.WithError(err).Warning("unable to store annotations as an annotation")
		} else {
			a[annotationsAnnotation] = string(aann)
		}
	} else {
		// There are no Calico annotations - nil out the k8s res.
		delete(a, annotationsAnnotation)
	}
	k8sRes.GetObjectMeta().SetAnnotations(a)
}

// Extract the Calico resource Metadata from the k8s resource annotations for non-CRD
// backed resources.  This extracts the Annotations and Labels stored as a annotation,
// and fills in the CreationTimestamp and UID from the k8s resource.
func SetCalicoMetadataFromK8sAnnotations(calicoRes Resource, k8sRes Resource) {
	com := calicoRes.GetObjectMeta()
	kom := k8sRes.GetObjectMeta()
	com.SetResourceVersion(kom.GetResourceVersion())
	com.SetCreationTimestamp(kom.GetCreationTimestamp())
	com.SetUID(kom.GetUID())
	a := kom.GetAnnotations()
	if a == nil {
		return
	}

	if lann, ok := a[labelsAnnotation]; ok {
		var labels map[string]string
		if err := json.Unmarshal([]byte(lann), &labels); err != nil {
			log.WithError(err).Warning("unable to parse labels annotation")
		} else {
			com.SetLabels(labels)
		}
	}
	if aann, ok := a[annotationsAnnotation]; ok {
		var annotations map[string]string
		if err := json.Unmarshal([]byte(aann), &annotations); err != nil {
			log.WithError(err).Warning("unable to parse annotations annotation")
		} else {
			com.SetAnnotations(annotations)
		}
	}
}

// Store Calico Metadata in the k8s resource annotations for CRD backed resources.
// This should store all Metadata except for those stored in Annotations and Labels and
// store them in annotations.
func ConvertCalicoResourceToK8sResource(resIn Resource) (Resource, error) {
	rom := resIn.GetObjectMeta()

	// Make sure to remove data that is passed to Kubernetes so it is not duplicated in
	// the annotations.
	romCopy := &metav1.ObjectMeta{}
	rom.(*metav1.ObjectMeta).DeepCopyInto(romCopy)
	romCopy.Name = ""
	romCopy.Namespace = ""
	romCopy.ResourceVersion = ""
	romCopy.Labels = nil
	romCopy.Annotations = nil
	romCopy.UID = ""

	// Marshal the data and store the json representation in the annotations.
	metadataBytes, err := json.Marshal(romCopy)
	if err != nil {
		return nil, err
	}
	annotations := rom.GetAnnotations()
	if len(annotations) == 0 {
		annotations = make(map[string]string)
	}
	annotations[metadataAnnotation] = string(metadataBytes)

	// Make sure to clear out all of the Calico Metadata out of the ObjectMeta except for
	// Name, Namespace, ResourceVersion, Labels, and Annotations. Annotations is already
	// copied so it can be set separately.
	meta := &metav1.ObjectMeta{}
	meta.Name = rom.GetName()
	meta.Namespace = rom.GetNamespace()
	meta.ResourceVersion = rom.GetResourceVersion()
	meta.Labels = rom.GetLabels()

	// Reverse the UID for UISettings to ensure garbage collection works correctly.
	switch resIn.(type) {
	case *v3.UISettings:
		meta.UID = ReverseUID(rom.GetUID())
	default:
		meta.UID = rom.GetUID()
	}

	resOut := resIn.DeepCopyObject().(Resource)
	romOut := resOut.GetObjectMeta()
	meta.DeepCopyInto(romOut.(*metav1.ObjectMeta))
	romOut.SetAnnotations(annotations)

	return resOut, nil
}

// Retrieve all of the Calico Metadata from the k8s resource annotations for CRD backed
// resources. This should remove the relevant Calico Metadata annotation when it has finished.
func ConvertK8sResourceToCalicoResource(res Resource) error {
	rom := res.GetObjectMeta()
	annotations := rom.GetAnnotations()
	if len(annotations) == 0 {
		// Make no changes if there are no annotations to read Calico Metadata out of.
		return nil
	}
	if _, ok := annotations[metadataAnnotation]; !ok {
		// No changes if there are no annotations stored on the Resource.
		return nil
	}

	meta := &metav1.ObjectMeta{}
	err := json.Unmarshal([]byte(annotations[metadataAnnotation]), meta)
	if err != nil {
		return err
	}

	// Clear out the annotations
	delete(annotations, metadataAnnotation)
	if len(annotations) == 0 {
		annotations = nil
	}

	// Manually write in the data not stored in the annotations: Name, Namespace, ResourceVersion,
	// Labels, and Annotations so that they do not get overwritten.
	meta.Name = rom.GetName()
	meta.Namespace = rom.GetNamespace()
	meta.ResourceVersion = rom.GetResourceVersion()
	meta.Labels = rom.GetLabels()
	meta.Annotations = annotations

	// Reverse the UID for UISettings to ensure garbage collection works correctly.
	switch res.(type) {
	case *v3.UISettings:
		meta.UID = ReverseUID(rom.GetUID())
	default:
		meta.UID = rom.GetUID()
	}

	// If no creation timestamp was stored in the metadata annotation, use the one from the CR.
	// The timestamp is normally set in the clientv3 code. However, for objects that bypass
	// the v3 client (e.g., IPAM), we won't have generated a creation timestamp and the field
	// is required on update calls.
	if meta.GetCreationTimestamp().Time.IsZero() {
		meta.SetCreationTimestamp(rom.GetCreationTimestamp())
	}

	// Overwrite the K8s metadata with the Calico metadata.
	meta.DeepCopyInto(rom.(*metav1.ObjectMeta))

	return nil
}

// ReverseUID reverses the segments of a UID to create another UID.
//
// We use this to map between the CRD and v3 resource types. It is not possible to use the same UID as this breaks
// garbage collection of the v3 resource types. We need to be able to deterministically map between the v3 and CRD
// UID and this seems like a reasonable approach. We could potentially store in the annotation (so both CRD and v3
// have this annotation and we switch annotation w/ the metadata UID) - however that doesn't work for deletes where
// the metadata is unavailable, but a UID may have been supplied to handle atomicity.
func ReverseUID(uid types.UID) types.UID {
	parts := strings.Split(string(uid), "-")
	for ii := range parts {
		parts[ii] = ReverseString(parts[ii])
	}
	return types.UID(strings.Join(parts, "-"))
}

func ReverseString(s string) string {
	r := []rune(s)
	for ii, jj := 0, len(r)-1; ii < len(r)/2; ii, jj = ii+1, jj-1 {
		r[ii], r[jj] = r[jj], r[ii]
	}
	return string(r)
}
