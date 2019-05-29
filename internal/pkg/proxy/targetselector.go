package proxy

import (
	"net/http"
	"net/url"
)

func XTarget() func (r *http.Request) string {
	return func (r *http.Request) string {
		return r.Header.Get("x-target")
	}
}

func Path() func (r *http.Request) string {
	return func (r *http.Request) string {
		return "api"
	}
}

func CreateStaticTargets() map[string]*url.URL {
	targets := make(map[string]*url.URL)

	targets["api"] = parse("http://localhost:8001")
	targets["es"] = parse("http://localhost:8002")

	return targets
}

func CreateStaticTargetsForAgent() map[string]*url.URL {
	targets := make(map[string]*url.URL)

	targets["api"] = parse("http://localhost:3000")

	return targets
}

func parse(rawUrl string) *url.URL{
	url, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}

	return url
}
