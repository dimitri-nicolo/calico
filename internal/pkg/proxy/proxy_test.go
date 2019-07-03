package proxy_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/proxy"
)

func init() {
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)
}

var _ = Describe("Proxy", func() {
	Describe("When proxy is empty", func() {
		It("should return error to any request", func() {
			p, err := proxy.New(nil)
			Expect(err).NotTo(HaveOccurred())

			r, err := http.NewRequest("GET", "http://host/path", nil)
			Expect(err).NotTo(HaveOccurred())

			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)

			res := w.Result()
			Expect(res.StatusCode).To(Equal(404))
		})

		It("should fail to configure with a bad target", func() {
			_, err := proxy.New([]proxy.Target{{}})
			Expect(err).To(HaveOccurred())
		})

		It("should fail to configure with a bad path", func() {
			_, err := proxy.New([]proxy.Target{
				{
					Path: "",
					Dest: &url.URL{
						Scheme: "http",
						Host:   "some",
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("should fail to configure with the same path twice", func() {
			_, err := proxy.New([]proxy.Target{
				{
					Path: "/",
					Dest: &url.URL{
						Scheme: "http",
						Host:   "some",
					},
				},
				{
					Path: "/",
					Dest: &url.URL{
						Scheme: "http",
						Host:   "other",
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("When proxy is configured", func() {
		t := &transport{
			func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: body("")}, nil
			},
		}

		p, _ := proxy.New([]proxy.Target{
			{
				Path: "/path/",
				Dest: &url.URL{
					Scheme: "http",
					Host:   "some",
				},
				Transport: t,
			},
		})

		It("should be redirected to the root", func() {
			r, err := http.NewRequest("GET", "http://host/path", nil)
			Expect(err).NotTo(HaveOccurred())
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)

			res := w.Result()
			Expect(res.StatusCode).To(Equal(301))
			Expect(res.Header.Get("Location")).To(Equal("/path/"))
		})

		It("should reach the root", func() {
			r, err := http.NewRequest("GET", "http://host/path/", nil)
			Expect(err).NotTo(HaveOccurred())
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)

			res := w.Result()
			Expect(res.StatusCode).To(Equal(200))
		})

		It("should reach a sub tree", func() {
			r, err := http.NewRequest("GET", "http://host/path/sub/tree", nil)
			Expect(err).NotTo(HaveOccurred())
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)

			res := w.Result()
			Expect(res.StatusCode).To(Equal(200))
		})

		It("should fail to reach unconfigured target", func() {
			r, err := http.NewRequest("GET", "http://host/unconfigured", nil)
			Expect(err).NotTo(HaveOccurred())
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)

			res := w.Result()
			Expect(res.StatusCode).To(Equal(404))
		})
	})

	Describe("When some targets have a token", func() {
		token := "some-token"
		noToken := "no token"

		withToken := &transport{
			func(r *http.Request) (*http.Response, error) {
				h := r.Header.Get("Authorization")
				if h == "" {
					return &http.Response{StatusCode: 400, Body: body("no token")}, nil
				}
				if h != "Bearer "+token {
					return &http.Response{StatusCode: 400, Body: body("bad token")}, nil
				}
				return &http.Response{StatusCode: 200, Body: body(token)}, nil
			},
		}

		withoutToken := &transport{
			func(r *http.Request) (*http.Response, error) {
				h := r.Header.Get("Authorization")
				if h != "" {
					return &http.Response{StatusCode: 400, Body: body("unexpected token " + h)}, nil
				}
				return &http.Response{StatusCode: 200, Body: body(noToken)}, nil
			},
		}

		p, _ := proxy.New([]proxy.Target{
			{
				Path: "/token",
				Dest: &url.URL{
					Scheme: "http",
					Host:   "some",
				},
				Token:     token,
				Transport: withToken,
			},
			{
				Path: "/",
				Dest: &url.URL{
					Scheme: "http",
					Host:   "other",
				},
				Transport: withoutToken,
			},
		})

		It("should get token if configured with one", func() {
			r, err := http.NewRequest("GET", "http://host/token", nil)
			Expect(err).NotTo(HaveOccurred())
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)

			res := w.Result()
			Expect(res.StatusCode).To(Equal(200))

			msg, err := ioutil.ReadAll(res.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(msg)).To(Equal(token))
		})

		It("should not get token if unconfigured", func() {
			r, err := http.NewRequest("GET", "http://host/path", nil)
			Expect(err).NotTo(HaveOccurred())
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)

			res := w.Result()
			Expect(res.StatusCode).To(Equal(200))

			msg, err := ioutil.ReadAll(res.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(msg)).To(Equal(noToken))
		})
	})
})

type transport struct {
	rt func(*http.Request) (*http.Response, error)
}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.rt(r)
}

func body(msg string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(msg))
}
