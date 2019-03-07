package util

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestRunLoop(t *testing.T) {
	g := NewGomegaWithT(t)

	maxDuration := time.Millisecond * 10
	period := 100 * time.Microsecond

	ctx, cancel := context.WithTimeout(context.TODO(), maxDuration)
	defer cancel()

	c := 0
	RunLoop(ctx, func() { c++ }, period)

	g.Expect(c).Should(BeNumerically("~", maxDuration/period, 1))
}
