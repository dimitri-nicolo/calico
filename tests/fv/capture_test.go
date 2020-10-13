// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"testing"

	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	. "github.com/projectcalico/calicoctl/tests/fv/utils"
	"github.com/projectcalico/libcalico-go/lib/logutils"
)

func init() {
	log.AddHook(logutils.ContextHook{})
	log.SetFormatter(&logutils.Formatter{})
}


func TestCaptureArgs(t *testing.T) {
	RegisterTestingT(t)

	const usage = `Usage:
  calicoctl captured-packets (copy |clean ) <NAME>
                [--config=<CONFIG>] [--namespace=<NS>] [--all-namespaces] [--dest=<DEST>]
`
	var tables = []struct {
		args []string
		expectedOutput string
		shouldFail bool
	}{
		{[]string{"captured-packets"}, "Invalid option", true},
		{[]string{"captured-packets", "--any_command"}, "Invalid option", true},
		{[]string{"captured-packets", "copy"}, "Invalid option", true},
		{[]string{"captured-packets", "clean"}, "Invalid option", true},
		{[]string{"captured-packets", "--help"}, usage, false},
		{[]string{"captured-packets", "-h"}, usage, false},
		{[]string{"captured-packets", "copy", "-h"}, usage, false},
		{[]string{"captured-packets", "clean", "-h"}, usage, false},
	}

	for _, entry := range tables {
		log.Infof("Running calicoctl with %v", entry.args)
		out, err := CalicoctlMayFail(true, entry.args...)
		if entry.shouldFail {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(out).To(ContainSubstring(entry.expectedOutput))
	}
}
