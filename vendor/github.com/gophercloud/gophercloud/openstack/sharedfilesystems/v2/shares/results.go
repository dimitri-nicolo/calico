package shares

import (
	"encoding/json"
	"time"

	"github.com/gophercloud/gophercloud"
)

// Share contains all information associated with an OpenStack Share
type Share struct {
	// The availability zone of the share
	AvailabilityZone string `json:"availability_zone"`
	// A description of the share
	Description string `json:"description,omitempty"`
	// DisplayDescription is inherited from BlockStorage API.
	// Both Description and DisplayDescription can be used
	DisplayDescription string `json:"display_description,omitempty"`
	// DisplayName is inherited from BlockStorage API
	// Both DisplayName and Name can be used
	DisplayName string `json:"display_name,omitempty"`
	// Indicates whether a share has replicas or not.
	HasReplicas bool `json:"has_replicas"`
	// The host name of the share
	Host string `json:"host"`
	// The UUID of the share
	ID string `json:"id"`
	// Indicates the visibility of the share
	IsPublic bool `json:"is_public,omitempty"`
	// Share links for pagination
	Links []map[string]string `json:"links"`
	// Key, value -pairs of custom metadata
	Metadata map[string]string `json:"metadata,omitempty"`
	// The name of the share
	Name string `json:"name,omitempty"`
	// The UUID of the project to which this share belongs to
	ProjectID string `json:"project_id"`
	// The share replication type
	ReplicationType string `json:"replication_type,omitempty"`
	// The UUID of the share network
	ShareNetworkID string `json:"share_network_id"`
	// The shared file system protocol
	ShareProto string `json:"share_proto"`
	// The UUID of the share server
	ShareServerID string `json:"share_server_id"`
	// The UUID of the share type.
	ShareType string `json:"share_type"`
	// The name of the share type.
	ShareTypeName string `json:"share_type_name"`
	// Size of the share in GB
	Size int `json:"size"`
	// UUID of the snapshot from which to create the share
	SnapshotID string `json:"snapshot_id"`
	// The share status
	Status string `json:"status"`
	// The task state, used for share migration
	TaskState string `json:"task_state"`
	// The type of the volume
	VolumeType string `json:"volume_type,omitempty"`
	// The UUID of the consistency group this share belongs to
	ConsistencyGroupID string `json:"consistency_group_id"`
	// Used for filtering backends which either support or do not support share snapshots
	SnapshotSupport          bool   `json:"snapshot_support"`
	SourceCgsnapshotMemberID string `json:"source_cgsnapshot_member_id"`
	// Timestamp when the share was created
	CreatedAt time.Time `json:"-"`
}

func (r *Share) UnmarshalJSON(b []byte) error {
	type tmp Share
	var s struct {
		tmp
		CreatedAt gophercloud.JSONRFC3339MilliNoZ `json:"created_at"`
	}
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*r = Share(s.tmp)

	r.CreatedAt = time.Time(s.CreatedAt)

	return nil
}

type commonResult struct {
	gophercloud.Result
}

// Extract will get the Share object from the commonResult
func (r commonResult) Extract() (*Share, error) {
	var s struct {
		Share *Share `json:"share"`
	}
	err := r.ExtractInto(&s)
	return s.Share, err
}

// CreateResult contains the response body and error from a Create request.
type CreateResult struct {
	commonResult
}

// DeleteResult contains the response body and error from a Delete request.
type DeleteResult struct {
	gophercloud.ErrResult
}

// GetResult contains the response body and error from a Get request.
type GetResult struct {
	commonResult
}

// GetExportLocationsResult contains the result body and error from an
// GetExportLocations request.
type GetExportLocationsResult struct {
	gophercloud.Result
}

// ExportLocation contains all information associated with a share export location
type ExportLocation struct {
	// The export location path that should be used for mount operation.
	Path string `json:"path"`
	// The UUID of the share instance that this export location belongs to.
	ShareInstanceID string `json:"share_instance_id"`
	// Defines purpose of an export location.
	// If set to true, then it is expected to be used for service needs
	// and by administrators only.
	// If it is set to false, then this export location can be used by end users.
	IsAdminOnly bool `json:"is_admin_only"`
	// The share export location UUID.
	ID string `json:"id"`
	// Drivers may use this field to identify which export locations are
	// most efficient and should be used preferentially by clients.
	// By default it is set to false value. New in version 2.14
	Preferred bool `json:"preferred"`
}

// Extract will get the Export Locations from the commonResult
func (r GetExportLocationsResult) Extract() ([]ExportLocation, error) {
	var s struct {
		ExportLocations []ExportLocation `json:"export_locations"`
	}
	err := r.ExtractInto(&s)
	return s.ExportLocations, err
}

// AccessRight contains all information associated with an OpenStack share
// Grant Access Response
type AccessRight struct {
	// The UUID of the share to which you are granted or denied access.
	ShareID string `json:"share_id"`
	// The access rule type that can be "ip", "cert" or "user".
	AccessType string `json:"access_type,omitempty"`
	// The value that defines the access that can be a valid format of IP, cert or user.
	AccessTo string `json:"access_to,omitempty"`
	// The access credential of the entity granted share access.
	AccessKey string `json:"access_key,omitempty"`
	// The access level to the share is either "rw" or "ro".
	AccessLevel string `json:"access_level,omitempty"`
	// The state of the access rule
	State string `json:"state,omitempty"`
	// The access rule ID.
	ID string `json:"id"`
}

// Extract will get the GrantAccess object from the commonResult
func (r GrantAccessResult) Extract() (*AccessRight, error) {
	var s struct {
		AccessRight *AccessRight `json:"access"`
	}
	err := r.ExtractInto(&s)
	return s.AccessRight, err
}

// GrantAccessResult contains the result body and error from an GrantAccess request.
type GrantAccessResult struct {
	gophercloud.Result
}
