package statser

import (
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestStatusClearError(t *testing.T) {
	g := NewGomegaWithT(t)

	expected := []ErrorCondition{
		{
			Type:    "keep",
			Message: "should be kept",
		},
	}
	status := &Status{
		ErrorConditions: expected,
	}
	status.Error("remove", errors.New("should not be kept"))
	status.ClearError("remove")

	g.Expect(status.ErrorConditions).Should(Equal(expected), "Only expected error conditions should be kept")
}

func TestStatusSuccessfulSearch(t *testing.T) {
	g := NewGomegaWithT(t)

	status := &Status{}
	status.SuccessfulSearch()
	g.Expect(status.LastSuccessfulSearch).ShouldNot(Equal(time.Time{}), "LastSuccessfulSearch set")
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}), "LastSuccessfulSync not set")
}

func TestStatusSuccessfulSync(t *testing.T) {
	g := NewGomegaWithT(t)

	status := &Status{}
	status.SuccessfulSync()

	g.Expect(status.LastSuccessfulSync).ShouldNot(Equal(time.Time{}), "LastSuccessfulSync set")
	g.Expect(status.LastSuccessfulSearch).Should(Equal(time.Time{}), "LastSuccessfulSearch not set")
}
