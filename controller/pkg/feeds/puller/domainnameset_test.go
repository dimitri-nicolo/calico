// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
)

func TestQueryDomainNameSet(t *testing.T) {
	g := NewGomegaWithT(t)

	input := db.DomainNameSetSpec{
		"www.badguys.co.uk",
		"we-love-malware.io ",
		"z.f.com # a comment after a valid address",
		"  hax4u.ru",
		"com # a top-level-domain is technically a valid domain name",
		"wWw.bOTnET..qQ. # should normalize case and dots",
	}
	expected := db.IPSetSpec{
		"www.badguys.co.uk",
		"we-love-malware.io",
		"z.f.com",
		"hax4u.ru",
		"com",
		"www.botnet.qq",
	}

	client := &http.Client{}
	resp := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(strings.Join(append(input, "# comment", "", " ", "junk&stuff", "-junk.com"), "\n"))),
	}
	client.Transport = &MockRoundTripper{
		Response: resp,
	}
	s := &statser.MockStatser{}
	edn := elastic.NewMockDomainNameSetsController()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	puller := NewDomainNameSetHTTPPuller(&testGTFDomainNameSet, &db.MockSets{}, &MockConfigMap{}, &MockSecrets{}, client, edn).(*httpPuller)

	go func() {
		err := puller.query(ctx, s, 1, 0)
		g.Expect(err).ShouldNot(HaveOccurred())
	}()

	g.Eventually(edn.Sets).Should(HaveKey(testGTFDomainNameSet.Name))
	dset, ok := edn.Sets()[testGlobalThreatFeed.Name]
	g.Expect(ok).Should(BeTrue(), "Received a snapshot")
	g.Expect(dset).Should(HaveLen(len(expected)))
	for idx, actual := range dset {
		g.Expect(actual).Should(Equal(expected[idx]))
	}

	status := s.Status()
	g.Expect(status.LastSuccessfulSync.Time).ShouldNot(Equal(time.Time{}), "Sync time was set")
	g.Expect(status.LastSuccessfulSearch.Time).Should(Equal(time.Time{}), "Search time was not set")
	g.Expect(status.ErrorConditions).Should(HaveLen(0), "Statser errors were not reported")
}

func TestGetStartupDelayDomainNameSet(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	edn := elastic.NewMockDomainNameSetsController()
	puller := NewDomainNameSetHTTPPuller(&testGTFDomainNameSet, &db.MockSets{
		Time: time.Now().Add(-time.Hour),
	}, &MockConfigMap{ConfigMapData: configMapData}, &MockSecrets{SecretsData: secretsData}, nil, edn).(*httpPuller)

	delay := puller.getStartupDelay(ctx)

	g.Expect(delay).Should(BeNumerically("~", puller.period-time.Hour, time.Minute))
}

func TestCanonicalizeDNSName(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(canonicalizeDNSName("tigera.io")).Should(Equal("tigera.io"))
	g.Expect(canonicalizeDNSName(".tigera.io.")).Should(Equal("tigera.io"))
	g.Expect(canonicalizeDNSName("..tigera..io..")).Should(Equal("tigera.io"))
	g.Expect(canonicalizeDNSName("tIgeRa.Io")).Should(Equal("tigera.io"))
}
