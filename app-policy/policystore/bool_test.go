package policystore

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestGetBoolFromConfig(t *testing.T) {
	RegisterTestingT(t)
	m := map[string]string{
		"value1": "true",
		"value2": "false",
		"value3": "foobarbaz",
	}
	Expect(getBoolFromConfig(m, "missing", false)).To(BeFalse())
	Expect(getBoolFromConfig(m, "missing", true)).To(BeTrue())
	Expect(getBoolFromConfig(m, "value1", false)).To(BeTrue())
	Expect(getBoolFromConfig(m, "value2", true)).To(BeFalse())
	Expect(getBoolFromConfig(m, "value3", false)).To(BeFalse())
	Expect(getBoolFromConfig(m, "value3", true)).To(BeTrue())
}
