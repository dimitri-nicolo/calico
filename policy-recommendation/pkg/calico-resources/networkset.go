// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package calicoresources

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

// newPrivateNetworkNetworkSet returns a pointer to the 'private-network' staged network set.
func newPrivateNetworkNetworkSet(owner metav1.OwnerReference) *v3.GlobalNetworkSet {
	return &v3.GlobalNetworkSet{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				fmt.Sprintf("%s/scope", PolicyRecKeyName): string(PrivateNetworkScope),
			},
			Name:            PrivateNetworkSetName,
			OwnerReferences: []metav1.OwnerReference{owner},
			Labels: map[string]string{
				fmt.Sprintf("%s/kind", projectCalicoKeyName): string(NetworkSetScope),
				fmt.Sprintf("%s/name", projectCalicoKeyName): PrivateNetworkSetName,
			},
		},
		Spec: v3.GlobalNetworkSetSpec{
			// RFC1918 Subnets. To be filled in manually after the network set has
			// been created.
			// "10.0.0.0/8",
			// "172.16.0.0/12",
			// "192.168.0.0/16",
			Nets: []string{},
		},
	}
}

// MaybeCreatePrivateNetworkSet attempts to create a new private network set if does not already exist.
func MaybeCreatePrivateNetworkSet(ctx context.Context, calico calicoclient.ProjectcalicoV3Interface, owner metav1.OwnerReference) error {
	if _, err := calico.GlobalNetworkSets().Get(ctx, PrivateNetworkSetName, metav1.GetOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			// Network set doesn't exist, create a new one
			networkSet := newPrivateNetworkNetworkSet(owner)

			_, err = calico.GlobalNetworkSets().Create(ctx, networkSet, metav1.CreateOptions{})
			if err != nil {
				log.WithError(err).Errorf("failed to create global network set: %s.", PrivateNetworkSetName)
				return err
			}
			log.Infof("New global network set '%s' created", PrivateNetworkSetName)
		} else {
			log.WithError(err).Errorf("Failed to get network set: %s", PrivateNetworkSetName)
			return err
		}
	}

	return nil
}
