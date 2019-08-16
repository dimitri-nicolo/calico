// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestConfig(t *testing.T) {
	g := NewWithT(t)

	matches, err := filepath.Glob("test_data/*/*.json")
	g.Expect(err).ShouldNot(HaveOccurred())

	for _, fn := range matches {
		fmt.Printf("Processing %s\n", fn)
		f, err := os.Open(fn)
		g.Expect(err).ShouldNot(HaveOccurred())
		defer f.Close()

		b, err := ioutil.ReadAll(f)
		g.Expect(err).ShouldNot(HaveOccurred())

		tcs := []struct {
			Expected string             `json:"expected"`
			Input    elastic.RecordSpec `json:"input"`
		}{}
		err = json.Unmarshal(b, &tcs)
		g.Expect(err).ShouldNot(HaveOccurred())

		for idx, tc := range tcs {
			fmt.Printf("tc %d\n", idx)
			g.Expect(tc.Input.Id).ShouldNot(BeEmpty())
			g.Expect(Jobs).Should(HaveKey(tc.Input.Id))
			g.Expect(Jobs[tc.Input.Id].Detectors).Should(HaveKey(tc.Input.DetectorIndex))

			actual := &strings.Builder{}
			err = Jobs[tc.Input.Id].Detectors[tc.Input.DetectorIndex].Execute(actual, tc.Input)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(actual.String()).Should(Equal(tc.Expected))
		}
	}
}
