// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package keyselector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/tigera/compliance/pkg/resources"
)

var (
	c1 = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeEndpoints,
		NameNamespace: resources.NameNamespace{
			Name: "1",
		},
	}
	c2 = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeEndpoints,
		NameNamespace: resources.NameNamespace{
			Name: "2",
		},
	}
	o1 = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeHostEndpoints,
		NameNamespace: resources.NameNamespace{
			Name: "1",
		},
	}
	o2 = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeHostEndpoints,
		NameNamespace: resources.NameNamespace{
			Name: "2",
		},
	}
)

type cb struct {
	owner  resources.ResourceID
	client resources.ResourceID
	key    string
}

type tester struct {
	k            Interface
	matchStarted set.Set
	matchStopped set.Set
}

func newTester() *tester {
	t := &tester{
		k: New(),
	}
	t.k.RegisterCallbacks(
		[]schema.GroupVersionKind{resources.ResourceTypeHostEndpoints, resources.ResourceTypeEndpoints},
		t.onMatchStarted, t.onMatchStopped,
	)
	return t
}

func (t *tester) onMatchStarted(owner, client resources.ResourceID, key string) {
	t.matchStarted.Add(cb{owner, client, key})
}

func (t *tester) onMatchStopped(owner, client resources.ResourceID, key string) {
	t.matchStopped.Add(cb{owner, client, key})
}

func (t *tester) setClientKeys(client resources.ResourceID, keys set.Set) {
	if client.GroupVersionKind != resources.ResourceTypeEndpoints {
		panic("Error in test code, passing in wrong client type")
	}
	t.matchStarted = set.New()
	t.matchStopped = set.New()
	t.k.SetClientKeys(client, keys)
}

func (t *tester) setOwnerKeys(owner resources.ResourceID, keys set.Set) {
	if owner.GroupVersionKind != resources.ResourceTypeHostEndpoints {
		panic("Error in test code, passing in wrong owner type")
	}
	t.matchStarted = set.New()
	t.matchStopped = set.New()
	t.k.SetOwnerKeys(owner, keys)
}

func (t *tester) deleteClient(client resources.ResourceID) {
	if client.GroupVersionKind != resources.ResourceTypeEndpoints {
		panic("Error in test code, passing in wrong client type")
	}
	t.matchStarted = set.New()
	t.matchStopped = set.New()
	t.k.DeleteClient(client)
}

func (t *tester) deleteOwner(owner resources.ResourceID) {
	if owner.GroupVersionKind != resources.ResourceTypeHostEndpoints {
		panic("Error in test code, passing in wrong owner type")
	}
	t.matchStarted = set.New()
	t.matchStopped = set.New()
	t.k.DeleteOwner(owner)
}

func (t *tester) ExpectEmpty() {
	Expect(t.k.(*keySelector).clientsByKey).To(HaveLen(0))
	Expect(t.k.(*keySelector).keysByClient).To(HaveLen(0))
	Expect(t.k.(*keySelector).ownersByKey).To(HaveLen(0))
	Expect(t.k.(*keySelector).keysByOwner).To(HaveLen(0))

}

var _ = Describe("label selector checks", func() {
	It("simple matches between client and owner and then remove owner", func() {
		t := newTester()

		By("Setting client1 key A")
		t.setClientKeys(c1, set.From("A"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(BeZero())

		By("Setting owner1 key A")
		t.setOwnerKeys(o1, set.From("A"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(Equal(1))
		Expect(t.matchStarted.Contains(cb{o1, c1, "A"})).To(BeTrue())

		By("Deleting owner1")
		t.deleteOwner(o1)
		Expect(t.matchStopped.Len()).To(Equal(1))
		Expect(t.matchStarted.Len()).To(BeZero())
		Expect(t.matchStopped.Contains(cb{o1, c1, "A"})).To(BeTrue())

		By("Deleting client1")
		t.deleteClient(c1)
		Expect(t.matchStarted.Len()).To(BeZero())
		Expect(t.matchStopped.Len()).To(BeZero())

		By("Checking internal data")
		t.ExpectEmpty()
	})

	It("simple matches between client and owner and then remove owner", func() {
		t := newTester()

		By("Setting owner1 key A")
		t.setOwnerKeys(o1, set.From("A"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(BeZero())

		By("Setting client1 key A")
		t.setClientKeys(c1, set.From("A"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(Equal(1))
		Expect(t.matchStarted.Contains(cb{o1, c1, "A"})).To(BeTrue())

		By("Deleting client1")
		t.deleteClient(c1)
		Expect(t.matchStarted.Len()).To(BeZero())
		Expect(t.matchStopped.Len()).To(Equal(1))
		Expect(t.matchStopped.Contains(cb{o1, c1, "A"})).To(BeTrue())

		By("Deleting owner1")
		t.deleteOwner(o1)
		Expect(t.matchStarted.Len()).To(BeZero())
		Expect(t.matchStopped.Len()).To(BeZero())

		By("Checking internal data")
		t.ExpectEmpty()
	})

	It("simple matches multiple clients to owner then remove owner", func() {
		t := newTester()

		By("Setting owner1 key A")
		t.setOwnerKeys(o1, set.From("A"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(BeZero())

		By("Setting client1 key A")
		t.setClientKeys(c1, set.From("A"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(Equal(1))
		Expect(t.matchStarted.Contains(cb{o1, c1, "A"})).To(BeTrue())

		By("Setting client2 key A")
		t.setClientKeys(c2, set.From("A"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(Equal(1))
		Expect(t.matchStarted.Contains(cb{o1, c2, "A"})).To(BeTrue())

		By("Deleting owner1")
		t.deleteOwner(o1)
		Expect(t.matchStopped.Len()).To(Equal(2))
		Expect(t.matchStarted.Len()).To(BeZero())
		Expect(t.matchStopped.Contains(cb{o1, c1, "A"})).To(BeTrue())
		Expect(t.matchStopped.Contains(cb{o1, c2, "A"})).To(BeTrue())

		By("Deleting client1 and client2")
		t.deleteClient(c1)
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(BeZero())
		t.deleteClient(c2)
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(BeZero())

		By("Checking internal data")
		t.ExpectEmpty()
	})

	It("multi-way matches", func() {
		t := newTester()

		By("Setting owner1 key A")
		t.setOwnerKeys(o1, set.From("A"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(BeZero())

		By("Setting client1 keys A and B")
		t.setClientKeys(c1, set.From("A", "B"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(Equal(1))
		Expect(t.matchStarted.Contains(cb{o1, c1, "A"})).To(BeTrue())

		By("Setting owner2 keys A and B")
		t.setOwnerKeys(o2, set.From("A", "B"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(Equal(2))
		Expect(t.matchStarted.Contains(cb{o2, c1, "A"})).To(BeTrue())
		Expect(t.matchStarted.Contains(cb{o2, c1, "B"})).To(BeTrue())

		By("Updating owner1 key B")
		t.setOwnerKeys(o1, set.From("B"))
		Expect(t.matchStopped.Len()).To(Equal(1))
		Expect(t.matchStarted.Len()).To(Equal(1))
		Expect(t.matchStopped.Contains(cb{o1, c1, "A"})).To(BeTrue())
		Expect(t.matchStarted.Contains(cb{o1, c1, "B"})).To(BeTrue())

		By("Setting client2 keys B")
		t.setClientKeys(c2, set.From("B"))
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(Equal(2))
		Expect(t.matchStarted.Contains(cb{o1, c2, "B"})).To(BeTrue())
		Expect(t.matchStarted.Contains(cb{o2, c2, "B"})).To(BeTrue())

		By("Deleting client1")
		t.deleteClient(c1)
		Expect(t.matchStopped.Len()).To(Equal(3))
		Expect(t.matchStarted.Len()).To(BeZero())
		Expect(t.matchStopped.Contains(cb{o1, c1, "B"})).To(BeTrue())
		Expect(t.matchStopped.Contains(cb{o2, c1, "A"})).To(BeTrue())
		Expect(t.matchStopped.Contains(cb{o2, c1, "B"})).To(BeTrue())

		By("Deleting owner1")
		t.deleteOwner(o1)
		Expect(t.matchStopped.Len()).To(Equal(1))
		Expect(t.matchStarted.Len()).To(BeZero())
		Expect(t.matchStopped.Contains(cb{o1, c2, "B"})).To(BeTrue())

		By("Deleting owner2")
		t.deleteOwner(o2)
		Expect(t.matchStopped.Len()).To(Equal(1))
		Expect(t.matchStarted.Len()).To(BeZero())
		Expect(t.matchStopped.Contains(cb{o2, c2, "B"})).To(BeTrue())

		By("Deleting client2")
		t.deleteClient(c2)
		Expect(t.matchStopped.Len()).To(BeZero())
		Expect(t.matchStarted.Len()).To(BeZero())

		By("Checking internal data")
		t.ExpectEmpty()
	})
})
