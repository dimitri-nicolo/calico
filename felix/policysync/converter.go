package policysync

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/proto"
)

// Wrap - converts an update interface to known ToDataplane types we care about. Exported for use in dikastes as well
func Wrap(update interface{}) (*proto.ToDataplane, error) {
	switch e := update.(type) {
	case *proto.InSync:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_InSync{
				InSync: e,
			},
		}), nil
	case *proto.WorkloadEndpointUpdate:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_WorkloadEndpointUpdate{
				WorkloadEndpointUpdate: e,
			},
		}), nil
	case *proto.WorkloadEndpointRemove:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_WorkloadEndpointRemove{
				WorkloadEndpointRemove: e,
			},
		}), nil
	case *proto.ActiveProfileUpdate:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_ActiveProfileUpdate{
				ActiveProfileUpdate: e,
			},
		}), nil
	case *proto.ActiveProfileRemove:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_ActiveProfileRemove{
				ActiveProfileRemove: e,
			},
		}), nil
	case *proto.ActivePolicyUpdate:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_ActivePolicyUpdate{
				ActivePolicyUpdate: e,
			},
		}), nil
	case *proto.ActivePolicyRemove:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_ActivePolicyRemove{
				ActivePolicyRemove: e,
			},
		}), nil
	case *proto.ServiceAccountUpdate:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_ServiceAccountUpdate{
				ServiceAccountUpdate: e,
			},
		}), nil
	case *proto.ServiceAccountRemove:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_ServiceAccountRemove{
				ServiceAccountRemove: e,
			},
		}), nil
	case *proto.NamespaceUpdate:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_NamespaceUpdate{
				NamespaceUpdate: e,
			},
		}), nil
	case *proto.NamespaceRemove:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_NamespaceRemove{
				NamespaceRemove: e,
			},
		}), nil
	case *proto.IPSetUpdate:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_IpsetUpdate{
				IpsetUpdate: e,
			},
		}), nil
	case *proto.IPSetDeltaUpdate:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_IpsetDeltaUpdate{
				IpsetDeltaUpdate: e,
			},
		}), nil
	case *proto.IPSetRemove:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_IpsetRemove{
				IpsetRemove: e,
			},
		}), nil
	case *proto.RouteUpdate:
		log.Info("perhost: route update")
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_RouteUpdate{
				RouteUpdate: e,
			},
		}), nil
	case *proto.RouteRemove:
		log.Info("perhost: route remove")
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_RouteRemove{
				RouteRemove: e,
			},
		}), nil
	case *proto.ConfigUpdate:
		return (&proto.ToDataplane{
			Payload: &proto.ToDataplane_ConfigUpdate{
				ConfigUpdate: e,
			},
		}), nil
	default:
		return nil, fmt.Errorf("unknown type: %T", e)
	}
}
