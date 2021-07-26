// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package fv_test

import (
	"testing"

	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	. "github.com/projectcalico/calicoctl/v3/tests/fv/utils"
	"github.com/projectcalico/libcalico-go/lib/logutils"
)

func init() {
	log.AddHook(logutils.ContextHook{})
	log.SetFormatter(&logutils.Formatter{})
}

func TestCaptureArgs(t *testing.T) {
	RegisterTestingT(t)

	const usageCalicoctl = `Usage:
  calicoctl [options] <command> [<args>...]`

	var tables = []struct {
		args           []string
		expectedOutput string
		shouldFail     bool
	}{
		{[]string{"captured-packets"}, "Invalid option", true},
		{[]string{"captured-packets", "--any_command"}, "Invalid option", true},
		{[]string{"captured-packets", "any_command"}, "Invalid option", true},
		{[]string{"captured-packets", "copy"}, "Invalid option", true},
		{[]string{"captured-packets", "clean"}, "Invalid option", true},
		{[]string{"--help", "captured-packets"}, usageCalicoctl, false},
		{[]string{"-h", "captured-packets"}, usageCalicoctl, false},
	}

	for _, entry := range tables {
		log.Infof("Running calicoctl with %v", entry.args)
		out, err := CalicoctlMayFail(entry.args...)
		if entry.shouldFail {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(out).To(ContainSubstring(entry.expectedOutput))
	}
}
