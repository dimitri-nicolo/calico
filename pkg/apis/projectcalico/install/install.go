// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package install

import (
	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(projectcalico.AddToScheme(scheme))
	utilruntime.Must(v3.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v3.SchemeGroupVersion))
}
