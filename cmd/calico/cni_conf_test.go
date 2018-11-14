// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package main_test

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
)

var _ = Describe("CNI config template tests", func() {
	It("should be valid JSON", func() {
		f, err := ioutil.ReadFile("../../windows-packaging/TigeraCalico/cni.conf.template")
		Expect(err).NotTo(HaveOccurred())

		var data map[string]interface{}
		err = json.Unmarshal(f, &data)
		Expect(err).NotTo(HaveOccurred())
	})
})
