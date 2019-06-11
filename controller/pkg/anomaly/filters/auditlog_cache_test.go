// Copyright 2019 Tigera Inc. All rights reserved.

package filters

import (
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

func TestAuditLogCache(t *testing.T) {
	g := NewWithT(t)

	al := &db.MockAuditLog{}
	cache := newAuditLogCache(al)

	// Test that the cache passes objects neither Created nor Deleted
	k1 := auditKey{timestamp: time.Now()}

	filtered, err := cache.isKeyFiltered(k1)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeFalse())
	g.Expect(cache.cache).Should(HaveLen(1))
	g.Expect(cache.cache).Should(HaveKeyWithValue(k1.String(), false))

	// Test that the cache filters Created objects
	k2 := auditKey{timestamp: k1.timestamp.Add(time.Hour)}
	al.CreatedOk = true

	filtered, err = cache.isKeyFiltered(k2)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeTrue())
	g.Expect(cache.cache).Should(HaveLen(2))
	g.Expect(cache.cache).Should(HaveKeyWithValue(k2.String(), true))

	// Test that the cache filters Deleted objects
	k3 := auditKey{timestamp: k2.timestamp.Add(time.Hour)}
	al.CreatedOk = false
	al.DeletedOk = true

	filtered, err = cache.isKeyFiltered(k3)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeTrue())
	g.Expect(cache.cache).Should(HaveLen(3))
	g.Expect(cache.cache).Should(HaveKeyWithValue(k3.String(), true))

	// Test that the cache does not store Created errors
	k4 := auditKey{timestamp: k3.timestamp.Add(time.Hour)}
	al.CreatedErr = errors.New("test")

	_, err = cache.isKeyFiltered(k4)
	g.Expect(err).Should(HaveOccurred())
	g.Expect(cache.cache).Should(HaveLen(3))

	// Test that the cache does not store Deleted errors
	k5 := auditKey{timestamp: k4.timestamp.Add(time.Hour)}
	al.CreatedOk = false
	al.CreatedErr = nil
	al.DeletedErr = errors.New("test")

	_, err = cache.isKeyFiltered(k5)
	g.Expect(err).Should(HaveOccurred())
	g.Expect(cache.cache).Should(HaveLen(3))

	// Test that the cache is used for false values
	al.CreatedArgs = nil
	al.DeletedArgs = nil
	filtered, err = cache.isKeyFiltered(k1)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeFalse())
	g.Expect(al.CreatedArgs).Should(BeNil())
	g.Expect(al.DeletedArgs).Should(BeNil())

	// Test that the cache is used for true values
	filtered, err = cache.isKeyFiltered(k2)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeTrue())
	g.Expect(al.CreatedArgs).Should(BeNil())
	g.Expect(al.DeletedArgs).Should(BeNil())

	// Test the areKeysFiltered function

	// All false
	filtered, err = cache.areKeysFiltered(k1, k1)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeFalse())

	// One true
	filtered, err = cache.areKeysFiltered(k2)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeTrue())

	// Second true
	filtered, err = cache.areKeysFiltered(k1, k2)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeTrue())

	// First true
	filtered, err = cache.areKeysFiltered(k2, k1)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeTrue())

	// Second errors
	al.CreatedErr = errors.New("test")
	_, err = cache.areKeysFiltered(k1, k4)
	g.Expect(err).Should(HaveOccurred())

	// First errors
	_, err = cache.areKeysFiltered(k4, k1)
	g.Expect(err).Should(HaveOccurred())

	// First true, second errors but is never checked
	filtered, err = cache.areKeysFiltered(k2, k4)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(filtered).Should(BeTrue())
}
