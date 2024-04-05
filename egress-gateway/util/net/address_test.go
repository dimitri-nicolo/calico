package net

import (
	"net"
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseEgressPodIPs(t *testing.T) {
	RegisterTestingT(t)

	for _, tc := range []struct {
		input          string
		expectedOutput net.IP
	}{
		{
			input:          "",
			expectedOutput: nil,
		},
		{
			input:          "whatisip",
			expectedOutput: nil,
		},
		{
			input:          "192.168.0.1",
			expectedOutput: net.ParseIP("192.168.0.1"),
		},
		{
			input:          "192.168.0.1,badipaddress",
			expectedOutput: net.ParseIP("192.168.0.1"),
		},
		{
			input:          "2001::1",
			expectedOutput: nil,
		},
		{
			input:          "2001::1,a.b.c.d",
			expectedOutput: nil,
		},
		{
			input:          "10.10.10.10,2001::1",
			expectedOutput: net.ParseIP("10.10.10.10"),
		},
		{
			input:          "2001::1,10.10.10.10",
			expectedOutput: net.ParseIP("10.10.10.10"),
		},
		{
			input:          "192.168.1.1,10.10.10.10",
			expectedOutput: net.ParseIP("192.168.1.1"),
		},
	} {
		Expect(ParseEgressPodIPs(tc.input)).To(Equal(tc.expectedOutput))
	}
}
