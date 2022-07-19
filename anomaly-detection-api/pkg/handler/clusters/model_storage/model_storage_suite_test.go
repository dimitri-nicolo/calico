package model_storage_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestModelStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ModelStorage Suite")
}
