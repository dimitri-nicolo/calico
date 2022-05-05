package fv_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fv Suite")
}
