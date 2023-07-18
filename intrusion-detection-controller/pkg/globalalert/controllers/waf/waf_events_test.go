package waf

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WAF event cache", func() {
	var (
		wafLog   v1.WAFLog
		rawLog   []byte
		wafCache WafEventsCache
		err      error
	)

	BeforeEach(func() {
		wafCache = WafEventsCache{
			lastWafTimestamp: time.Now(), // subject to change
		}
		f := mustOpen("test_files/waf_log.json")
		defer f.Close()
		rawLog, err = io.ReadAll(f)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(rawLog, &wafLog)
		Expect(err).NotTo(HaveOccurred())

	})

	Context("NewWAFEvent", func() {
		It("create a new WAF event", func() {
			expected := v1.Event{
				Type:         query.WafEventType,
				Origin:       "Web Application Firewall",
				Time:         v1.NewEventDate(wafLog.Timestamp),
				Name:         "WAF Event",
				Description:  "Some traffic inside your cluster triggered some Web Application Firewall rules",
				Severity:     80,
				Host:         "lorcan-bz-wodc-kadm-node-1",
				Protocol:     "HTTP/1.1",
				SourceIP:     &wafLog.Source.IP,
				SourceName:   "-",
				DestIP:       &wafLog.Destination.IP,
				DestName:     "nginx-svc",
				MitreIDs:     &[]string{"T1190"},
				Mitigations:  &[]string{"Review the source of this event - an attacker could be inside your cluster attempting to exploit your web application. Calico network policy can be used to block the connection if the activity is not expected"},
				AttackVector: "Network",
				MitreTactic:  "Initial Access",
				Record:       wafLog,
			}
			generatedEvent := NewWafEvent(wafLog)
			Expect(generatedEvent).To(Equal(expected))

		})
	})

	Context("Testing WAF Event Cache", func() {
		It("Testing WAF Evennt Cache Add Pass", func() {

			wafCache.Add(wafLog)

			isPresent := wafCache.Contains(wafLog)
			Expect(isPresent).To(BeTrue())
		})

		It("Test WAF Event Cache Add Fail", func() {

			isPresent := wafCache.Contains(wafLog)
			Expect(isPresent).To(BeFalse())
		})
	})
})

func mustOpen(name string) io.ReadCloser {
	f, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	return f
}
