// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
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
		"junk&stuff # not a valid domain name, but still possible to query for",
		"-junk.com # also not a valid name, but still possible to query for",
		"mølmer-sørensen.gate",
		"xn--mlmer-srensen-bnbg.gate",
	}
	expected := db.IPSetSpec{
		"www.badguys.co.uk",
		"we-love-malware.io",
		"z.f.com",
		"hax4u.ru",
		"com",
		"www.botnet.qq",
		"junk&stuff",
		"-junk.com",
		"mølmer-sørensen.gate",
		"mølmer-sørensen.gate",
	}

	client := &http.Client{}
	resp := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(strings.Join(append(input, "# comment", "", " "), "\n"))),
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

func TestQueryDomainNameSet_WithGNS(t *testing.T) {
	g := NewGomegaWithT(t)

	input := db.DomainNameSetSpec{
		"www.badguys.co.uk",
		"we-love-malware.io ",
	}
	expected := db.IPSetSpec{
		"www.badguys.co.uk",
		"we-love-malware.io",
	}

	client := &http.Client{}
	resp := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(strings.Join(input, "\n"))),
	}
	client.Transport = &MockRoundTripper{
		Response: resp,
	}
	s := &statser.MockStatser{}
	edn := elastic.NewMockDomainNameSetsController()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	f := testGTFDomainNameSet.DeepCopy()
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{Labels: map[string]string{"key": "value"}}
	puller := NewDomainNameSetHTTPPuller(f, &db.MockSets{}, &MockConfigMap{}, &MockSecrets{}, client, edn).(*httpPuller)

	go func() {
		err := puller.query(ctx, s, 1, 0)
		g.Expect(err).ShouldNot(HaveOccurred())
	}()

	g.Eventually(edn.Sets).Should(HaveKey(testGTFDomainNameSet.Name))
	dset, ok := edn.Sets()[testGTFDomainNameSet.Name]
	g.Expect(ok).Should(BeTrue(), "Received a snapshot")
	g.Expect(dset).Should(HaveLen(len(expected)))
	for idx, actual := range dset {
		g.Expect(actual).Should(Equal(expected[idx]))
	}

	status := s.Status()
	// Pull should work as expected, but drop an error about GlobalNetworkSetSync
	g.Expect(status.LastSuccessfulSync.Time).ShouldNot(Equal(time.Time{}), "Sync time was set")
	g.Expect(status.LastSuccessfulSearch.Time).Should(Equal(time.Time{}), "Search time was not set")
	g.Expect(status.ErrorConditions).
		Should(ConsistOf([]v3.ErrorCondition{{Type: statser.GlobalNetworkSetSyncFailed, Message: "sync not supported for domain name set"}}))

	// Update the feed to remove the GNS and re-query
	puller.SetFeed(&testGTFDomainNameSet)
	go func() {
		err := puller.query(ctx, s, 1, 0)
		g.Expect(err).ShouldNot(HaveOccurred())
	}()

	g.Eventually(func() []v3.ErrorCondition { return s.Status().ErrorConditions }).Should(HaveLen(0), "should clear GNS error")
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
	g.Expect(canonicalizeDNSName("xn--Mlmer-Srensen-bnbg.gate")).Should(Equal("mølmer-sørensen.gate"))
	g.Expect(canonicalizeDNSName("mølmer-sørensen.gate")).Should(Equal("mølmer-sørensen.gate"))

	// www.Æther.com --- with capital, should be normalized to lowercase
	g.Expect(canonicalizeDNSName("www.xn--ther-9ja.com")).Should(Equal("www.æther.com"))

	// Names already in unicode should be normalized to lowercase
	g.Expect(canonicalizeDNSName("www.Æther.com")).Should(Equal("www.æther.com"))

	// Names with corrupted punycode should just be normalized with case and dots
	g.Expect(canonicalizeDNSName("xn--Mlmer-Srensen-bnb&..gate")).Should(Equal("xn--mlmer-srensen-bnb&.gate"))
}

func TestSyncGNSFromDB_DomainNameSet(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	feed := testGTFDomainNameSet.DeepCopy()
	feed.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{Labels: map[string]string{"key": "value"}}
	dnSet := &db.MockSets{
		Value: db.DomainNameSetSpec{"baddos.ooo"},
	}
	s := &statser.MockStatser{}

	puller := NewDomainNameSetHTTPPuller(feed, dnSet, &MockConfigMap{ConfigMapData: configMapData}, &MockSecrets{SecretsData: secretsData}, nil, nil).(*httpPuller)

	puller.gnsHandler.syncFromDB(ctx, s)

	g.Expect(s.Status().ErrorConditions).
		Should(ConsistOf([]v3.ErrorCondition{{Type: statser.GlobalNetworkSetSyncFailed, Message: "sync not supported for domain name set"}}))

	// modify to remove GNS sync and resync
	puller.SetFeed(&testGTFDomainNameSet)
	puller.gnsHandler.syncFromDB(ctx, s)
	g.Expect(s.Status().ErrorConditions).To(HaveLen(0))
}
