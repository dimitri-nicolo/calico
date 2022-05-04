package api_error_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApiError(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ApiError Suite")
}
