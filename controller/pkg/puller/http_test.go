package puller

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

func TestQuery(t *testing.T) {
	g := NewGomegaWithT(t)

	input := feed.IPSet{
		"1.2.3.4",
		"5.6.7.8 ",
		"2.0.0.0/8",
		"2.3.4.5/32 ",
		"2000::1",
		"2000::/5",
	}
	expected := feed.IPSet{
		"1.2.3.4/32",
		"5.6.7.8/32",
		"2.0.0.0/8",
		"2.3.4.5/32",
		"2000::1/128",
		"2000::/5",
	}
	timeout := time.Second

	client := &http.Client{}
	resp := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(strings.Join([]string(append(input, "# comment", "", " ", "junk", "junk/")), "\n"))),
	}
	client.Transport = &mockRT{
		resp: resp,
	}
	f := feed.NewFeed("test", "test-namespace")
	header := http.Header{}
	u := &url.URL{}

	puller := NewHTTPPuller(f, client, u, header, 0, 0).(*httpPuller)

	snapshots := make(chan feed.IPSet)

	statser := statser.NewStatser()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go func() {
		defer close(snapshots)
		puller.query(ctx, snapshots, statser, 1, 0)
	}()

	select {
	case snapshot, ok := <-snapshots:
		g.Expect(ok).Should(BeTrue(), "Received a snapshot")
		g.Expect(snapshot).Should(HaveLen(len(expected)))
		for idx, actual := range snapshot {
			g.Expect(actual).Should(Equal(expected[idx]))
		}
	case <-time.Tick(timeout):
		t.Fatal("query timed out")
	}

	status := statser.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}), "Sync was not successful")
	g.Expect(status.LastSuccessfulSearch).Should(Equal(time.Time{}), "Search was not successful")
	g.Expect(status.ErrorConditions).Should(HaveLen(0), "Status errors were not reported")
}

func TestQueryHTTPError(t *testing.T) {
	g := NewGomegaWithT(t)

	timeout := time.Second

	client := &http.Client{}
	err := TemporaryError("test error")
	rt := &mockRT{
		err: err,
	}
	client.Transport = rt

	f := feed.NewFeed("test", "test-namespace")
	header := http.Header{}
	u := &url.URL{}

	puller := NewHTTPPuller(f, client, u, header, 0, 0).(*httpPuller)

	snapshots := make(chan feed.IPSet)

	statser := statser.NewStatser()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	attempts := uint(5)
	go func() {
		defer close(snapshots)
		puller.query(ctx, snapshots, statser, attempts, 0)
	}()

	select {
	case _, ok := <-snapshots:
		g.Expect(ok).Should(BeFalse(), "Should not receive a snapshot")
	case <-time.Tick(timeout):
		t.Fatal("query timed out")
	}

	g.Expect(rt.c).Should(Equal(attempts), "Retried max times")

	status := statser.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}), "Sync was not successful")
	g.Expect(status.LastSuccessfulSearch).Should(Equal(time.Time{}), "Search was not successful")
	g.Expect(status.ErrorConditions).Should(HaveLen(1), "1 error should have been reported")
	g.Expect(status.ErrorConditions[0].Type).Should(Equal(statserType), "Error condition type is set correctly")
}

func TestQueryHTTPStatus404(t *testing.T) {
	g := NewGomegaWithT(t)

	timeout := time.Second

	client := &http.Client{}
	rt := &mockRT{
		resp: &http.Response{
			StatusCode: 404,
		},
	}
	client.Transport = rt

	f := feed.NewFeed("test", "test-namespace")
	header := http.Header{}
	u := &url.URL{}

	puller := NewHTTPPuller(f, client, u, header, 0, 0).(*httpPuller)

	snapshots := make(chan feed.IPSet)

	statser := statser.NewStatser()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	attempts := uint(5)
	go func() {
		defer close(snapshots)
		puller.query(ctx, snapshots, statser, attempts, 0)
	}()

	select {
	case _, ok := <-snapshots:
		g.Expect(ok).Should(BeFalse(), "Should not receive a snapshot")
	case <-time.Tick(timeout):
		t.Fatal("query timed out")
	}

	g.Expect(rt.c).Should(Equal(uint(1)), "Does not retry on error 404")

	status := statser.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}), "Sync was not successful")
	g.Expect(status.LastSuccessfulSearch).Should(Equal(time.Time{}), "Search was not successful")
	g.Expect(status.ErrorConditions).Should(HaveLen(1), "1 error should have been reported")
	g.Expect(status.ErrorConditions[0].Type).Should(Equal(statserType), "Error condition type is set correctly")
}

func TestQueryHTTPStatus500(t *testing.T) {
	g := NewGomegaWithT(t)

	timeout := time.Second

	client := &http.Client{}
	rt := &mockRT{
		resp: &http.Response{
			StatusCode: 500,
		},
	}
	client.Transport = rt

	f := feed.NewFeed("test", "test-namespace")
	header := http.Header{}
	u := &url.URL{}

	puller := NewHTTPPuller(f, client, u, header, 0, 0).(*httpPuller)

	snapshots := make(chan feed.IPSet)

	statser := statser.NewStatser()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	attempts := uint(5)
	go func() {
		defer close(snapshots)
		puller.query(ctx, snapshots, statser, attempts, 0)
	}()

	select {
	case _, ok := <-snapshots:
		g.Expect(ok).Should(BeFalse(), "Should not receive a snapshot")
	case <-time.Tick(timeout):
		t.Fatal("query timed out")
	}

	g.Expect(rt.c).Should(Equal(attempts))

	status := statser.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}), "Sync was not successful")
	g.Expect(status.LastSuccessfulSearch).Should(Equal(time.Time{}), "Search was not successful")
	g.Expect(status.ErrorConditions).Should(HaveLen(1), "1 error should have been reported")
	g.Expect(status.ErrorConditions[0].Type).Should(Equal(statserType), "Error condition type is set correctly")
}

type mockRT struct {
	resp *http.Response
	err  error
	c    uint
}

func (r *mockRT) RoundTrip(*http.Request) (*http.Response, error) {
	r.c++
	return r.resp, r.err
}
