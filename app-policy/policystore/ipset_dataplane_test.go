package policystore

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/ipsets"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

func TestAddOrReplaceIPSet(t *testing.T) {
	RegisterTestingT(t)

	t.Run("should not add IP set if UpdateType is not DOMAIN", func(t *testing.T) {
		// Initialize the policy store manager and dataplane
		policyStoreManager := NewPolicyStoreManager()
		ipFamily := ipsets.IPFamilyV4
		dataplane := NewDomainIPSetsDataplane(ipFamily, policyStoreManager)

		metadata := ipsets.IPSetMetadata{
			SetID:      "non-domain-set",
			Type:       ipsets.IPSetTypeHashIP,
			UpdateType: proto.IPSetUpdate_IP,
		}
		members := []string{"192.168.1.1"}

		// Call AddOrReplaceIPSet with non-DOMAIN UpdateType
		dataplane.AddOrReplaceIPSet(metadata, members)

		// Notify the policy store manager that the IP set is in sync
		policyStoreManager.OnInSync()

		// Verify that the IP set was not added
		policyStoreManager.DoWithReadLock(func(store *PolicyStore) {
			_, exists := store.IPSetByID[metadata.SetID]
			Expect(exists).To(BeFalse(), "Expected IP set '%s' not to be added", metadata.SetID)
		})
	})

	t.Run("should add new IP set with members when it does not exist", func(t *testing.T) {
		// Initialize the policy store manager and dataplane
		policyStoreManager := NewPolicyStoreManager()
		ipFamily := ipsets.IPFamilyV4
		dataplane := NewDomainIPSetsDataplane(ipFamily, policyStoreManager)

		metadata := ipsets.IPSetMetadata{
			SetID:      "test-set",
			Type:       ipsets.IPSetTypeHashIP,
			UpdateType: proto.IPSetUpdate_DOMAIN,
		}
		initialMembers := []string{"192.168.1.1", "192.168.1.2"}

		// Call AddOrReplaceIPSet with initial members
		dataplane.AddOrReplaceIPSet(metadata, initialMembers)

		// Notify the policy store manager that the IP set is in sync
		policyStoreManager.OnInSync()

		// Verify that the IP set was added with correct members
		policyStoreManager.DoWithReadLock(func(store *PolicyStore) {
			ipset, exists := store.IPSetByID[metadata.SetID]
			Expect(exists).To(BeTrue(), "Expected IP set '%s' to be added", metadata.SetID)

			expectedMembers := set.New[string]()
			expectedMembers.AddAll(initialMembers)
			actualMembers := ipset.Members()
			actualMembersSet := set.FromArray(actualMembers)
			Expect(expectedMembers).To(Equal(actualMembersSet), "Expected members %v, got %v", expectedMembers, actualMembersSet)
		})
	})

	t.Run("should replace existing IP set with new members", func(t *testing.T) {
		// Initialize the policy store manager and dataplane
		policyStoreManager := NewPolicyStoreManager()
		ipFamily := ipsets.IPFamilyV4
		dataplane := NewDomainIPSetsDataplane(ipFamily, policyStoreManager)

		metadata := ipsets.IPSetMetadata{
			SetID:      "test-set",
			Type:       ipsets.IPSetTypeHashIP,
			UpdateType: proto.IPSetUpdate_DOMAIN,
		}
		initialMembers := []string{"192.168.1.1", "192.168.1.2"}
		newMembers := []string{"192.168.1.3", "192.168.1.4"}

		// Add initial IP set
		dataplane.AddOrReplaceIPSet(metadata, initialMembers)

		// Replace IP set with new members
		dataplane.AddOrReplaceIPSet(metadata, newMembers)

		// Notify the policy store manager that the IP set is in sync
		policyStoreManager.OnInSync()

		// Verify that the IP set contains only new members
		policyStoreManager.DoWithReadLock(func(store *PolicyStore) {
			ipset, exists := store.IPSetByID[metadata.SetID]
			Expect(exists).To(BeTrue(), "Expected IP set '%s' to exist", metadata.SetID)

			expectedMembers := set.New[string]()
			expectedMembers.AddAll(newMembers)
			actualMembers := ipset.Members()
			actualMembersSet := set.FromArray(actualMembers)
			Expect(expectedMembers).To(Equal(actualMembersSet), "Expected members %v, got %v", expectedMembers, actualMembersSet)
		})
	})
}

func TestAddMembers(t *testing.T) {
	// Initialize the policy store manager and dataplane
	policyStoreManager := NewPolicyStoreManager()
	ipFamily := ipsets.IPFamilyV4
	dataplane := NewDomainIPSetsDataplane(ipFamily, policyStoreManager)

	// Define the IP set metadata and initial members
	metadata := ipsets.IPSetMetadata{
		SetID:      "test-set",
		Type:       ipsets.IPSetTypeHashIP,
		UpdateType: proto.IPSetUpdate_DOMAIN,
	}
	initialMembers := []string{"192.168.1.1", "192.168.1.2"}

	// Call AddOrReplaceIPSet with initial members
	dataplane.AddOrReplaceIPSet(metadata, initialMembers)

	// Define additional members to add
	newMembers := []string{"192.168.1.3", "192.168.1.4"}

	// Call AddMembers with new members
	dataplane.AddMembers(metadata.SetID, newMembers)

	// Notify the policy store manager that the IP set is in sync
	policyStoreManager.OnInSync()

	// Verify that the IP set now contains both initial and new members
	policyStoreManager.DoWithReadLock(func(store *PolicyStore) {
		ipset, exists := store.IPSetByID[metadata.SetID]
		if !exists {
			t.Fatalf("Expected IP set '%s' to exist after adding new members", metadata.SetID)
		}

		expectedMembers := set.New[string]()
		expectedMembers.AddAll(initialMembers)
		expectedMembers.AddAll(newMembers)
		actualMembers := ipset.Members()
		actualMembersSet := set.FromArray(actualMembers)

		if !expectedMembers.Equals(actualMembersSet) {
			t.Errorf("IP set members do not match after adding new members. Expected: %v, Got: %v", expectedMembers, actualMembers)
		}
	})
}

func TestRemoveMembers(t *testing.T) {
	// Initialize the policy store manager and dataplane
	policyStoreManager := NewPolicyStoreManager()
	ipFamily := ipsets.IPFamilyV4
	dataplane := NewDomainIPSetsDataplane(ipFamily, policyStoreManager)

	// Define the IP set metadata and initial members
	metadata := ipsets.IPSetMetadata{
		SetID:      "test-set",
		Type:       ipsets.IPSetTypeHashIP,
		UpdateType: proto.IPSetUpdate_DOMAIN,
	}
	initialMembers := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}

	// Call AddOrReplaceIPSet with initial members
	dataplane.AddOrReplaceIPSet(metadata, initialMembers)

	// Define members to remove
	removedMembers := []string{"192.168.1.3", "192.168.1.4"}

	// Call RemoveMembers with members to remove
	dataplane.RemoveMembers(metadata.SetID, removedMembers)

	// Notify the policy store manager that the IP set is in sync
	policyStoreManager.OnInSync()

	// Verify that the IP set now contains only the remaining members
	policyStoreManager.DoWithReadLock(func(store *PolicyStore) {
		ipset, exists := store.IPSetByID[metadata.SetID]
		if !exists {
			t.Fatalf("Expected IP set '%s' to exist after removing members", metadata.SetID)
		}

		expectedMembers := set.New[string]()
		expectedMembers.AddAll([]string{"192.168.1.1", "192.168.1.2"})
		actualMembers := ipset.Members()
		actualMembersSet := set.FromArray(actualMembers)

		if !expectedMembers.Equals(actualMembersSet) {
			t.Errorf("IP set members do not match after removing members. Expected: %v, Got: %v", expectedMembers, actualMembers)
		}
	})
}

func TestRemoveIPSet(t *testing.T) {
	// Initialize the policy store manager and dataplane
	policyStoreManager := NewPolicyStoreManager()
	ipFamily := ipsets.IPFamilyV4
	dataplane := NewDomainIPSetsDataplane(ipFamily, policyStoreManager)

	// Define the IP set metadata and initial members
	metadata := ipsets.IPSetMetadata{
		SetID:      "test-set",
		Type:       ipsets.IPSetTypeHashIP,
		UpdateType: proto.IPSetUpdate_DOMAIN,
	}
	initialMembers := []string{"192.168.1.1", "192.168.1.2"}

	// Call AddOrReplaceIPSet with initial members
	dataplane.AddOrReplaceIPSet(metadata, initialMembers)

	// Call RemoveIPSet to remove the IP set
	dataplane.RemoveIPSet(metadata.SetID)

	// Notify the policy store manager that the IP set is in sync
	policyStoreManager.OnInSync()

	// Verify that the IP set no longer exists
	policyStoreManager.DoWithReadLock(func(store *PolicyStore) {
		_, exists := store.IPSetByID[metadata.SetID]
		if exists {
			t.Fatalf("Expected IP set '%s' to be removed", metadata.SetID)
		}
	})
}

func TestGetIPFamily(t *testing.T) {
	// Initialize the policy store manager and dataplane
	policyStoreManager := NewPolicyStoreManager()
	ipFamily := ipsets.IPFamilyV4
	dataplane := NewDomainIPSetsDataplane(ipFamily, policyStoreManager)

	// Verify that the IP family is returned correctly
	if dataplane.GetIPFamily() != ipFamily {
		t.Errorf("Expected IP family to be %v, but got %v", ipFamily, dataplane.GetIPFamily())
	}
}

func TestGetTypeOf(t *testing.T) {
	// Initialize the policy store manager and dataplane
	policyStoreManager := NewPolicyStoreManager()
	ipFamily := ipsets.IPFamilyV4
	dataplane := NewDomainIPSetsDataplane(ipFamily, policyStoreManager)

	// Define the IP set metadata and initial members
	metadata := ipsets.IPSetMetadata{
		SetID:      "test-set",
		Type:       ipsets.IPSetTypeHashIP,
		UpdateType: proto.IPSetUpdate_DOMAIN,
	}
	initialMembers := []string{"192.168.1.1", "192.168.1.2"}

	// Call AddOrReplaceIPSet with initial members
	dataplane.AddOrReplaceIPSet(metadata, initialMembers)

	// Notify the policy store manager that the IP set is in sync
	policyStoreManager.OnInSync()

	// Verify that the type of the IP set is returned correctly
	ipsetType, err := dataplane.GetTypeOf(metadata.SetID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ipsetType != ipsets.IPSetTypeHashIP {
		t.Errorf("Expected IP set type to be %v, but got %v", ipsets.IPSetTypeHashIP, ipsetType)
	}
}
