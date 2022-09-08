// Code generated by mockery v2.14.0. DO NOT EDIT.

package k8s

import (
	apiserverinternalv1alpha1 "k8s.io/client-go/kubernetes/typed/apiserverinternal/v1alpha1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"

	appsv1beta1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"

	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"

	authenticationv1beta1 "k8s.io/client-go/kubernetes/typed/authentication/v1beta1"

	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"

	authorizationv1beta1 "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"

	autoscalingv1 "k8s.io/client-go/kubernetes/typed/autoscaling/v1"

	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"

	batchv1beta1 "k8s.io/client-go/kubernetes/typed/batch/v1beta1"

	certificatesv1 "k8s.io/client-go/kubernetes/typed/certificates/v1"

	certificatesv1beta1 "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"

	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"

	coordinationv1beta1 "k8s.io/client-go/kubernetes/typed/coordination/v1beta1"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	discovery "k8s.io/client-go/discovery"

	discoveryv1 "k8s.io/client-go/kubernetes/typed/discovery/v1"

	discoveryv1beta1 "k8s.io/client-go/kubernetes/typed/discovery/v1beta1"

	eventsv1 "k8s.io/client-go/kubernetes/typed/events/v1"

	eventsv1beta1 "k8s.io/client-go/kubernetes/typed/events/v1beta1"

	extensionsv1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"

	flowcontrolv1beta1 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta1"

	flowcontrolv1beta2 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta2"

	mock "github.com/stretchr/testify/mock"

	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"

	networkingv1beta1 "k8s.io/client-go/kubernetes/typed/networking/v1beta1"

	nodev1 "k8s.io/client-go/kubernetes/typed/node/v1"

	nodev1alpha1 "k8s.io/client-go/kubernetes/typed/node/v1alpha1"

	nodev1beta1 "k8s.io/client-go/kubernetes/typed/node/v1beta1"

	policyv1 "k8s.io/client-go/kubernetes/typed/policy/v1"

	policyv1beta1 "k8s.io/client-go/kubernetes/typed/policy/v1beta1"

	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"

	rbacv1alpha1 "k8s.io/client-go/kubernetes/typed/rbac/v1alpha1"

	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"

	schedulingv1 "k8s.io/client-go/kubernetes/typed/scheduling/v1"

	schedulingv1alpha1 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha1"

	schedulingv1beta1 "k8s.io/client-go/kubernetes/typed/scheduling/v1beta1"

	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"

	storagev1alpha1 "k8s.io/client-go/kubernetes/typed/storage/v1alpha1"

	storagev1beta1 "k8s.io/client-go/kubernetes/typed/storage/v1beta1"

	v1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"

	v1alpha1 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1alpha1"

	v1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"

	v1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"

	v2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"

	v2beta1 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta1"

	v2beta2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"

	v3 "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

// MockClientSet is an autogenerated mock type for the ClientSet type
type MockClientSet struct {
	mock.Mock
}

// AdmissionregistrationV1 provides a mock function with given fields:
func (_m *MockClientSet) AdmissionregistrationV1() v1.AdmissionregistrationV1Interface {
	ret := _m.Called()

	var r0 v1.AdmissionregistrationV1Interface
	if rf, ok := ret.Get(0).(func() v1.AdmissionregistrationV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.AdmissionregistrationV1Interface)
		}
	}

	return r0
}

// AdmissionregistrationV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) AdmissionregistrationV1beta1() v1beta1.AdmissionregistrationV1beta1Interface {
	ret := _m.Called()

	var r0 v1beta1.AdmissionregistrationV1beta1Interface
	if rf, ok := ret.Get(0).(func() v1beta1.AdmissionregistrationV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1beta1.AdmissionregistrationV1beta1Interface)
		}
	}

	return r0
}

// AppsV1 provides a mock function with given fields:
func (_m *MockClientSet) AppsV1() appsv1.AppsV1Interface {
	ret := _m.Called()

	var r0 appsv1.AppsV1Interface
	if rf, ok := ret.Get(0).(func() appsv1.AppsV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(appsv1.AppsV1Interface)
		}
	}

	return r0
}

// AppsV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) AppsV1beta1() appsv1beta1.AppsV1beta1Interface {
	ret := _m.Called()

	var r0 appsv1beta1.AppsV1beta1Interface
	if rf, ok := ret.Get(0).(func() appsv1beta1.AppsV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(appsv1beta1.AppsV1beta1Interface)
		}
	}

	return r0
}

// AppsV1beta2 provides a mock function with given fields:
func (_m *MockClientSet) AppsV1beta2() v1beta2.AppsV1beta2Interface {
	ret := _m.Called()

	var r0 v1beta2.AppsV1beta2Interface
	if rf, ok := ret.Get(0).(func() v1beta2.AppsV1beta2Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1beta2.AppsV1beta2Interface)
		}
	}

	return r0
}

// AuthenticationV1 provides a mock function with given fields:
func (_m *MockClientSet) AuthenticationV1() authenticationv1.AuthenticationV1Interface {
	ret := _m.Called()

	var r0 authenticationv1.AuthenticationV1Interface
	if rf, ok := ret.Get(0).(func() authenticationv1.AuthenticationV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(authenticationv1.AuthenticationV1Interface)
		}
	}

	return r0
}

// AuthenticationV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) AuthenticationV1beta1() authenticationv1beta1.AuthenticationV1beta1Interface {
	ret := _m.Called()

	var r0 authenticationv1beta1.AuthenticationV1beta1Interface
	if rf, ok := ret.Get(0).(func() authenticationv1beta1.AuthenticationV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(authenticationv1beta1.AuthenticationV1beta1Interface)
		}
	}

	return r0
}

// AuthorizationV1 provides a mock function with given fields:
func (_m *MockClientSet) AuthorizationV1() authorizationv1.AuthorizationV1Interface {
	ret := _m.Called()

	var r0 authorizationv1.AuthorizationV1Interface
	if rf, ok := ret.Get(0).(func() authorizationv1.AuthorizationV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(authorizationv1.AuthorizationV1Interface)
		}
	}

	return r0
}

// AuthorizationV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) AuthorizationV1beta1() authorizationv1beta1.AuthorizationV1beta1Interface {
	ret := _m.Called()

	var r0 authorizationv1beta1.AuthorizationV1beta1Interface
	if rf, ok := ret.Get(0).(func() authorizationv1beta1.AuthorizationV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(authorizationv1beta1.AuthorizationV1beta1Interface)
		}
	}

	return r0
}

// AutoscalingV1 provides a mock function with given fields:
func (_m *MockClientSet) AutoscalingV1() autoscalingv1.AutoscalingV1Interface {
	ret := _m.Called()

	var r0 autoscalingv1.AutoscalingV1Interface
	if rf, ok := ret.Get(0).(func() autoscalingv1.AutoscalingV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(autoscalingv1.AutoscalingV1Interface)
		}
	}

	return r0
}

// AutoscalingV2 provides a mock function with given fields:
func (_m *MockClientSet) AutoscalingV2() v2.AutoscalingV2Interface {
	ret := _m.Called()

	var r0 v2.AutoscalingV2Interface
	if rf, ok := ret.Get(0).(func() v2.AutoscalingV2Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v2.AutoscalingV2Interface)
		}
	}

	return r0
}

// AutoscalingV2beta1 provides a mock function with given fields:
func (_m *MockClientSet) AutoscalingV2beta1() v2beta1.AutoscalingV2beta1Interface {
	ret := _m.Called()

	var r0 v2beta1.AutoscalingV2beta1Interface
	if rf, ok := ret.Get(0).(func() v2beta1.AutoscalingV2beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v2beta1.AutoscalingV2beta1Interface)
		}
	}

	return r0
}

// AutoscalingV2beta2 provides a mock function with given fields:
func (_m *MockClientSet) AutoscalingV2beta2() v2beta2.AutoscalingV2beta2Interface {
	ret := _m.Called()

	var r0 v2beta2.AutoscalingV2beta2Interface
	if rf, ok := ret.Get(0).(func() v2beta2.AutoscalingV2beta2Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v2beta2.AutoscalingV2beta2Interface)
		}
	}

	return r0
}

// BatchV1 provides a mock function with given fields:
func (_m *MockClientSet) BatchV1() batchv1.BatchV1Interface {
	ret := _m.Called()

	var r0 batchv1.BatchV1Interface
	if rf, ok := ret.Get(0).(func() batchv1.BatchV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(batchv1.BatchV1Interface)
		}
	}

	return r0
}

// BatchV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) BatchV1beta1() batchv1beta1.BatchV1beta1Interface {
	ret := _m.Called()

	var r0 batchv1beta1.BatchV1beta1Interface
	if rf, ok := ret.Get(0).(func() batchv1beta1.BatchV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(batchv1beta1.BatchV1beta1Interface)
		}
	}

	return r0
}

// CertificatesV1 provides a mock function with given fields:
func (_m *MockClientSet) CertificatesV1() certificatesv1.CertificatesV1Interface {
	ret := _m.Called()

	var r0 certificatesv1.CertificatesV1Interface
	if rf, ok := ret.Get(0).(func() certificatesv1.CertificatesV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(certificatesv1.CertificatesV1Interface)
		}
	}

	return r0
}

// CertificatesV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) CertificatesV1beta1() certificatesv1beta1.CertificatesV1beta1Interface {
	ret := _m.Called()

	var r0 certificatesv1beta1.CertificatesV1beta1Interface
	if rf, ok := ret.Get(0).(func() certificatesv1beta1.CertificatesV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(certificatesv1beta1.CertificatesV1beta1Interface)
		}
	}

	return r0
}

// CoordinationV1 provides a mock function with given fields:
func (_m *MockClientSet) CoordinationV1() coordinationv1.CoordinationV1Interface {
	ret := _m.Called()

	var r0 coordinationv1.CoordinationV1Interface
	if rf, ok := ret.Get(0).(func() coordinationv1.CoordinationV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(coordinationv1.CoordinationV1Interface)
		}
	}

	return r0
}

// CoordinationV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) CoordinationV1beta1() coordinationv1beta1.CoordinationV1beta1Interface {
	ret := _m.Called()

	var r0 coordinationv1beta1.CoordinationV1beta1Interface
	if rf, ok := ret.Get(0).(func() coordinationv1beta1.CoordinationV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(coordinationv1beta1.CoordinationV1beta1Interface)
		}
	}

	return r0
}

// CoreV1 provides a mock function with given fields:
func (_m *MockClientSet) CoreV1() corev1.CoreV1Interface {
	ret := _m.Called()

	var r0 corev1.CoreV1Interface
	if rf, ok := ret.Get(0).(func() corev1.CoreV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(corev1.CoreV1Interface)
		}
	}

	return r0
}

// Discovery provides a mock function with given fields:
func (_m *MockClientSet) Discovery() discovery.DiscoveryInterface {
	ret := _m.Called()

	var r0 discovery.DiscoveryInterface
	if rf, ok := ret.Get(0).(func() discovery.DiscoveryInterface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(discovery.DiscoveryInterface)
		}
	}

	return r0
}

// DiscoveryV1 provides a mock function with given fields:
func (_m *MockClientSet) DiscoveryV1() discoveryv1.DiscoveryV1Interface {
	ret := _m.Called()

	var r0 discoveryv1.DiscoveryV1Interface
	if rf, ok := ret.Get(0).(func() discoveryv1.DiscoveryV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(discoveryv1.DiscoveryV1Interface)
		}
	}

	return r0
}

// DiscoveryV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) DiscoveryV1beta1() discoveryv1beta1.DiscoveryV1beta1Interface {
	ret := _m.Called()

	var r0 discoveryv1beta1.DiscoveryV1beta1Interface
	if rf, ok := ret.Get(0).(func() discoveryv1beta1.DiscoveryV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(discoveryv1beta1.DiscoveryV1beta1Interface)
		}
	}

	return r0
}

// EventsV1 provides a mock function with given fields:
func (_m *MockClientSet) EventsV1() eventsv1.EventsV1Interface {
	ret := _m.Called()

	var r0 eventsv1.EventsV1Interface
	if rf, ok := ret.Get(0).(func() eventsv1.EventsV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(eventsv1.EventsV1Interface)
		}
	}

	return r0
}

// EventsV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) EventsV1beta1() eventsv1beta1.EventsV1beta1Interface {
	ret := _m.Called()

	var r0 eventsv1beta1.EventsV1beta1Interface
	if rf, ok := ret.Get(0).(func() eventsv1beta1.EventsV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(eventsv1beta1.EventsV1beta1Interface)
		}
	}

	return r0
}

// ExtensionsV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) ExtensionsV1beta1() extensionsv1beta1.ExtensionsV1beta1Interface {
	ret := _m.Called()

	var r0 extensionsv1beta1.ExtensionsV1beta1Interface
	if rf, ok := ret.Get(0).(func() extensionsv1beta1.ExtensionsV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(extensionsv1beta1.ExtensionsV1beta1Interface)
		}
	}

	return r0
}

// FlowcontrolV1alpha1 provides a mock function with given fields:
func (_m *MockClientSet) FlowcontrolV1alpha1() v1alpha1.FlowcontrolV1alpha1Interface {
	ret := _m.Called()

	var r0 v1alpha1.FlowcontrolV1alpha1Interface
	if rf, ok := ret.Get(0).(func() v1alpha1.FlowcontrolV1alpha1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1alpha1.FlowcontrolV1alpha1Interface)
		}
	}

	return r0
}

// FlowcontrolV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) FlowcontrolV1beta1() flowcontrolv1beta1.FlowcontrolV1beta1Interface {
	ret := _m.Called()

	var r0 flowcontrolv1beta1.FlowcontrolV1beta1Interface
	if rf, ok := ret.Get(0).(func() flowcontrolv1beta1.FlowcontrolV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(flowcontrolv1beta1.FlowcontrolV1beta1Interface)
		}
	}

	return r0
}

// FlowcontrolV1beta2 provides a mock function with given fields:
func (_m *MockClientSet) FlowcontrolV1beta2() flowcontrolv1beta2.FlowcontrolV1beta2Interface {
	ret := _m.Called()

	var r0 flowcontrolv1beta2.FlowcontrolV1beta2Interface
	if rf, ok := ret.Get(0).(func() flowcontrolv1beta2.FlowcontrolV1beta2Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(flowcontrolv1beta2.FlowcontrolV1beta2Interface)
		}
	}

	return r0
}

// InternalV1alpha1 provides a mock function with given fields:
func (_m *MockClientSet) InternalV1alpha1() apiserverinternalv1alpha1.InternalV1alpha1Interface {
	ret := _m.Called()

	var r0 apiserverinternalv1alpha1.InternalV1alpha1Interface
	if rf, ok := ret.Get(0).(func() apiserverinternalv1alpha1.InternalV1alpha1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apiserverinternalv1alpha1.InternalV1alpha1Interface)
		}
	}

	return r0
}

// NetworkingV1 provides a mock function with given fields:
func (_m *MockClientSet) NetworkingV1() networkingv1.NetworkingV1Interface {
	ret := _m.Called()

	var r0 networkingv1.NetworkingV1Interface
	if rf, ok := ret.Get(0).(func() networkingv1.NetworkingV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(networkingv1.NetworkingV1Interface)
		}
	}

	return r0
}

// NetworkingV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) NetworkingV1beta1() networkingv1beta1.NetworkingV1beta1Interface {
	ret := _m.Called()

	var r0 networkingv1beta1.NetworkingV1beta1Interface
	if rf, ok := ret.Get(0).(func() networkingv1beta1.NetworkingV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(networkingv1beta1.NetworkingV1beta1Interface)
		}
	}

	return r0
}

// NodeV1 provides a mock function with given fields:
func (_m *MockClientSet) NodeV1() nodev1.NodeV1Interface {
	ret := _m.Called()

	var r0 nodev1.NodeV1Interface
	if rf, ok := ret.Get(0).(func() nodev1.NodeV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(nodev1.NodeV1Interface)
		}
	}

	return r0
}

// NodeV1alpha1 provides a mock function with given fields:
func (_m *MockClientSet) NodeV1alpha1() nodev1alpha1.NodeV1alpha1Interface {
	ret := _m.Called()

	var r0 nodev1alpha1.NodeV1alpha1Interface
	if rf, ok := ret.Get(0).(func() nodev1alpha1.NodeV1alpha1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(nodev1alpha1.NodeV1alpha1Interface)
		}
	}

	return r0
}

// NodeV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) NodeV1beta1() nodev1beta1.NodeV1beta1Interface {
	ret := _m.Called()

	var r0 nodev1beta1.NodeV1beta1Interface
	if rf, ok := ret.Get(0).(func() nodev1beta1.NodeV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(nodev1beta1.NodeV1beta1Interface)
		}
	}

	return r0
}

// PolicyV1 provides a mock function with given fields:
func (_m *MockClientSet) PolicyV1() policyv1.PolicyV1Interface {
	ret := _m.Called()

	var r0 policyv1.PolicyV1Interface
	if rf, ok := ret.Get(0).(func() policyv1.PolicyV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(policyv1.PolicyV1Interface)
		}
	}

	return r0
}

// PolicyV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) PolicyV1beta1() policyv1beta1.PolicyV1beta1Interface {
	ret := _m.Called()

	var r0 policyv1beta1.PolicyV1beta1Interface
	if rf, ok := ret.Get(0).(func() policyv1beta1.PolicyV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(policyv1beta1.PolicyV1beta1Interface)
		}
	}

	return r0
}

// ProjectcalicoV3 provides a mock function with given fields:
func (_m *MockClientSet) ProjectcalicoV3() v3.ProjectcalicoV3Interface {
	ret := _m.Called()

	var r0 v3.ProjectcalicoV3Interface
	if rf, ok := ret.Get(0).(func() v3.ProjectcalicoV3Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v3.ProjectcalicoV3Interface)
		}
	}

	return r0
}

// RbacV1 provides a mock function with given fields:
func (_m *MockClientSet) RbacV1() rbacv1.RbacV1Interface {
	ret := _m.Called()

	var r0 rbacv1.RbacV1Interface
	if rf, ok := ret.Get(0).(func() rbacv1.RbacV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rbacv1.RbacV1Interface)
		}
	}

	return r0
}

// RbacV1alpha1 provides a mock function with given fields:
func (_m *MockClientSet) RbacV1alpha1() rbacv1alpha1.RbacV1alpha1Interface {
	ret := _m.Called()

	var r0 rbacv1alpha1.RbacV1alpha1Interface
	if rf, ok := ret.Get(0).(func() rbacv1alpha1.RbacV1alpha1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rbacv1alpha1.RbacV1alpha1Interface)
		}
	}

	return r0
}

// RbacV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) RbacV1beta1() rbacv1beta1.RbacV1beta1Interface {
	ret := _m.Called()

	var r0 rbacv1beta1.RbacV1beta1Interface
	if rf, ok := ret.Get(0).(func() rbacv1beta1.RbacV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rbacv1beta1.RbacV1beta1Interface)
		}
	}

	return r0
}

// SchedulingV1 provides a mock function with given fields:
func (_m *MockClientSet) SchedulingV1() schedulingv1.SchedulingV1Interface {
	ret := _m.Called()

	var r0 schedulingv1.SchedulingV1Interface
	if rf, ok := ret.Get(0).(func() schedulingv1.SchedulingV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(schedulingv1.SchedulingV1Interface)
		}
	}

	return r0
}

// SchedulingV1alpha1 provides a mock function with given fields:
func (_m *MockClientSet) SchedulingV1alpha1() schedulingv1alpha1.SchedulingV1alpha1Interface {
	ret := _m.Called()

	var r0 schedulingv1alpha1.SchedulingV1alpha1Interface
	if rf, ok := ret.Get(0).(func() schedulingv1alpha1.SchedulingV1alpha1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(schedulingv1alpha1.SchedulingV1alpha1Interface)
		}
	}

	return r0
}

// SchedulingV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) SchedulingV1beta1() schedulingv1beta1.SchedulingV1beta1Interface {
	ret := _m.Called()

	var r0 schedulingv1beta1.SchedulingV1beta1Interface
	if rf, ok := ret.Get(0).(func() schedulingv1beta1.SchedulingV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(schedulingv1beta1.SchedulingV1beta1Interface)
		}
	}

	return r0
}

// StorageV1 provides a mock function with given fields:
func (_m *MockClientSet) StorageV1() storagev1.StorageV1Interface {
	ret := _m.Called()

	var r0 storagev1.StorageV1Interface
	if rf, ok := ret.Get(0).(func() storagev1.StorageV1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(storagev1.StorageV1Interface)
		}
	}

	return r0
}

// StorageV1alpha1 provides a mock function with given fields:
func (_m *MockClientSet) StorageV1alpha1() storagev1alpha1.StorageV1alpha1Interface {
	ret := _m.Called()

	var r0 storagev1alpha1.StorageV1alpha1Interface
	if rf, ok := ret.Get(0).(func() storagev1alpha1.StorageV1alpha1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(storagev1alpha1.StorageV1alpha1Interface)
		}
	}

	return r0
}

// StorageV1beta1 provides a mock function with given fields:
func (_m *MockClientSet) StorageV1beta1() storagev1beta1.StorageV1beta1Interface {
	ret := _m.Called()

	var r0 storagev1beta1.StorageV1beta1Interface
	if rf, ok := ret.Get(0).(func() storagev1beta1.StorageV1beta1Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(storagev1beta1.StorageV1beta1Interface)
		}
	}

	return r0
}

type mockConstructorTestingTNewMockClientSet interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockClientSet creates a new instance of MockClientSet. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockClientSet(t mockConstructorTestingTNewMockClientSet) *MockClientSet {
	mock := &MockClientSet{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
