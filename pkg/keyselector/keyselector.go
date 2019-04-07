// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package keyselector

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/sethelper"
)

// This file implements a key selector. This acts as a bridge between the resources that are configured with one or more
// keys (e.g. endpoints with Key addresses) and other resources that act as clients for these keys (e.g. service Keys).
//
// TODO (note): This is very similar to the LabelSelector and we could have used that to handle the simple match start/stop
// processing. However, this implementation does not require selector processing and so a little more efficient, and
// and eventually we may want additional data from the stored info (e.g. who owns an Key, and which Keys are not accounted
// for in the clients - both of these translate to useful report information)
//
// TODO(rlb): This is effecively the same processing as the policy rule selector manager, except we'd need to implement
// additional hooks to notify of "first" match and "last" match on start or stopped respectively. We should update the
// rule selector manager to use this.

// Callbacks. This is the notify the owner that there is a client using one of their keys.
type MatchStarted func(owner, client resources.ResourceID, key string, firstKey bool)
type MatchStopped func(owner, client resources.ResourceID, key string, lastKey bool)

// KeyManager interface.
type Interface interface {
	RegisterCallbacks(kinds []schema.GroupVersionKind, started MatchStarted, stopped MatchStopped)
	SetOwnerKeys(owner resources.ResourceID, keys set.Set)
	SetClientKeys(client resources.ResourceID, keys set.Set)
	DeleteOwner(owner resources.ResourceID)
	DeleteClient(client resources.ResourceID)
}

// New creates a new KeyManager.
func New() Interface {
	keym := &keySelector{
		keysByOwner:       make(map[resources.ResourceID]set.Set),
		keysByClient:      make(map[resources.ResourceID]set.Set),
		clientsByKey:      make(map[string]resources.Set),
		ownersByKey:       make(map[string]resources.Set),
		keysByOwnerClient: make(map[ownerClient]set.Set),
	}
	return keym
}

// keySelector implements the KeyManager interface.
type keySelector struct {
	// The cross referencing.
	keysByOwner       map[resources.ResourceID]set.Set
	keysByClient      map[resources.ResourceID]set.Set
	ownersByKey       map[string]resources.Set
	clientsByKey      map[string]resources.Set
	keysByOwnerClient map[ownerClient]set.Set

	// Callbacks
	cbs []callbacksWithKind
}

type callbacksWithKind struct {
	started MatchStarted
	stopped MatchStopped
	kind    schema.GroupVersionKind
}

type ownerClient struct {
	owner  resources.ResourceID
	client resources.ResourceID
}

// RegisterCallbacks registers client callbacks with this manager.
func (ls *keySelector) RegisterCallbacks(kinds []schema.GroupVersionKind, started MatchStarted, stopped MatchStopped) {
	for _, kind := range kinds {
		ls.cbs = append(ls.cbs, callbacksWithKind{
			started: started,
			stopped: stopped,
			kind:    kind,
		})
	}
}

// SetOwnerKeys sets owners keys.
func (m *keySelector) SetOwnerKeys(owner resources.ResourceID, keys set.Set) {
	// Start by finding the delta sets of Keys.
	currentSet := m.keysByOwner[owner]
	if currentSet == nil {
		currentSet = set.New()
	}
	if keys == nil {
		delete(m.keysByOwner, owner)
		keys = set.New()
	} else {
		m.keysByOwner[owner] = keys
	}

	sethelper.IterDifferences(currentSet, keys,
		// Key address is removed from the owners list.
		func(item interface{}) error {
			key := item.(string)

			// Update the ownersByKey set.
			owners := m.ownersByKey[key]
			owners.Discard(owner)
			if owners.Len() == 0 {
				delete(m.ownersByKey, key)
			}

			// Notify links to clients.
			clients := m.clientsByKey[key]
			if clients == nil {
				return nil
			}
			clients.Iter(func(client resources.ResourceID) error {
				m.onKeyMatchStopped(owner, client, key)
				return nil
			})
			return nil
		},
		// New Key address is added to the owners list.
		func(item interface{}) error {
			key := item.(string)

			// Update the ownersByKey set.
			owners := m.ownersByKey[key]
			if owners == nil {
				owners = resources.NewSet()
				m.ownersByKey[key] = owners
			}
			owners.Add(owner)

			// Notify links to clients.
			clients := m.clientsByKey[key]
			if clients == nil {
				return nil
			}
			clients.Iter(func(client resources.ResourceID) error {
				m.onKeyMatchStarted(owner, client, key)
				return nil
			})
			return nil
		},
	)
}

// SetClientKeys sets clients keys.
func (m *keySelector) SetClientKeys(client resources.ResourceID, keys set.Set) {
	// Start by finding the delta sets of Keys.
	currentSet := m.keysByClient[client]
	if currentSet == nil {
		currentSet = set.New()
	}
	if keys == nil {
		delete(m.keysByClient, client)
		keys = set.New()
	} else {
		m.keysByClient[client] = keys
	}

	sethelper.IterDifferences(currentSet, keys,
		// Key address is removed from the clients list.
		func(item interface{}) error {
			key := item.(string)

			// Update the clientsByKey set.
			clients := m.clientsByKey[key]
			clients.Discard(client)
			if clients.Len() == 0 {
				delete(m.clientsByKey, key)
			}

			// Notify links to owners.
			owners := m.ownersByKey[key]
			if owners == nil {
				return nil
			}
			owners.Iter(func(owner resources.ResourceID) error {
				m.onKeyMatchStopped(owner, client, key)
				return nil
			})
			return nil
		},
		// New Key address is added to the clients list.
		func(item interface{}) error {
			key := item.(string)

			// Update the clientsByKey set.
			clients := m.clientsByKey[key]
			if clients == nil {
				clients = resources.NewSet()
				m.clientsByKey[key] = clients
			}
			clients.Add(client)

			// Notify links to owners.
			owners := m.ownersByKey[key]
			if owners == nil {
				return nil
			}
			owners.Iter(func(owner resources.ResourceID) error {
				m.onKeyMatchStarted(owner, client, key)
				return nil
			})
			return nil
		},
	)
}

func (m *keySelector) DeleteOwner(owner resources.ResourceID) {
	m.SetOwnerKeys(owner, nil)
}

func (m *keySelector) DeleteClient(client resources.ResourceID) {
	m.SetClientKeys(client, nil)
}

// onMatchStarted is called from the InheritIndex helper when a selector-endpoint match has
// started.
func (c *keySelector) onKeyMatchStarted(owner, client resources.ResourceID, key string) {
	var firstKey bool
	oc := ownerClient{owner: owner, client: client}
	keys := c.keysByOwnerClient[oc]
	if keys == nil {
		keys = set.New()
		c.keysByOwnerClient[oc] = keys
		firstKey = true
	}
	keys.Add(key)

	for i := range c.cbs {
		if c.cbs[i].kind == owner.GroupVersionKind || c.cbs[i].kind == client.GroupVersionKind {
			c.cbs[i].started(owner, client, key, firstKey)
		}
	}
}

// onMatchStopped is called from the InheritIndex helper when a selector-endpoint match has
// stopped.
func (c *keySelector) onKeyMatchStopped(owner, client resources.ResourceID, key string) {
	var lastKey bool
	oc := ownerClient{owner: owner, client: client}
	keys := c.keysByOwnerClient[oc]
	keys.Discard(key)
	if keys.Len() == 0 {
		delete(c.keysByOwnerClient, oc)
		lastKey = true
	}

	for i := range c.cbs {
		if c.cbs[i].kind == owner.GroupVersionKind || c.cbs[i].kind == client.GroupVersionKind {
			c.cbs[i].stopped(owner, client, key, lastKey)
		}
	}
}
