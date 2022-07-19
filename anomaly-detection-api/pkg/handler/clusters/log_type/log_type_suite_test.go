package log_type_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLogType(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LogType Suite")
}
