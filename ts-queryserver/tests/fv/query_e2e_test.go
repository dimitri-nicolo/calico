// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.
package fv

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes/fake"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/calicoctl/calicoctl/resourcemgr"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	libapi "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/backend"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/config"
	queryhdr "github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/query"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/server"
)

var _ = testutils.E2eDatastoreDescribe("Query tests", testutils.DatastoreEtcdV3, func(config apiconfig.CalicoAPIConfig) {

	DescribeTable("Query tests (e2e with server)",
		func(tqds []testQueryData, crossCheck func(tqd testQueryData, addr string, netClient *http.Client)) {
			By("Creating a v3 client interface")
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			By("Cleaning the datastore")
			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			err = be.Clean()
			Expect(err).NotTo(HaveOccurred())

			// Choose an arbitrary port for the server to listen on.
			By("Choosing an arbitrary available local port for the queryserver")
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			Expect(err).NotTo(HaveOccurred())
			addr := listener.Addr().String()
			listener.Close()

			// Get server configuration variables meant for FVs.
			servercfg := getDummyConfigFromEnvFv(addr, "", "")

			fakeK8sClient := fake.NewSimpleClientset()
			mh := &mockHandler{}

			By("Starting the queryserver")
			srv := server.NewServer(fakeK8sClient, &config, servercfg, mh)
			err = srv.Start()
			Expect(err).NotTo(HaveOccurred())
			defer srv.Stop()

			var configured map[model.ResourceKey]resourcemgr.ResourceObject
			var netClient = &http.Client{Timeout: time.Second * 10}
			for _, tqd := range tqds {
				By(fmt.Sprintf("Creating the resources for test: %s", tqd.description))
				configured = createResources(c, tqd.resources, configured)

				By(fmt.Sprintf("Running query for test: %s", tqd.description))
				queryFn := getQueryFunction(tqd, addr, netClient)
				Eventually(queryFn).Should(Equal(tqd.response), tqd.description)
				Consistently(queryFn).Should(Equal(tqd.response), tqd.description)

				if crossCheck != nil {
					By("Running a cross-check query")
					crossCheck(tqd, addr, netClient)
				}
			}
		},

		Entry("Summary queries", summaryTestQueryData(), nil),
		Entry("Node queries", nodeTestQueryData(), nil),
		Entry("Endpoint queries", endpointTestQueryData(), crossCheckEndpointQuery),
		Entry("Policy queries", policyTestQueryData(), crossCheckPolicyQuery),
	)
})

func getQueryFunction(tqd testQueryData, addr string, netClient *http.Client) func() interface{} {
	By(fmt.Sprintf("Calculating the URL for the test: %s", tqd.description))
	qurl := calculateQueryUrl(addr, tqd.query)

	By(fmt.Sprintf("Creating the query function for test: %s", tqd.description))
	return func() interface{} {
		// Return the result if we have it, otherwise the error, this allows us to use Eventually to
		// check both values and errors.
		log.WithField("url", qurl).Debug("Running query")
		r, err := netClient.Get(qurl)
		if err != nil {
			return err
		}
		defer r.Body.Close()
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		bodyString := string(bodyBytes)
		if r.StatusCode != http.StatusOK {
			return errorResponse{
				text: strings.TrimSpace(bodyString),
				code: r.StatusCode,
			}
		}

		if _, ok := tqd.response.(errorResponse); ok {
			// We are expecting an error but didn't get one, we'll have to return an error containing
			// the raw json.
			return fmt.Errorf("expecting error but command was successful: %s", bodyString)
		}

		// The response body should be json and the same type as the expected response object.
		ro := reflect.New(reflect.TypeOf(tqd.response).Elem()).Interface()
		err = json.Unmarshal(bodyBytes, ro)
		if err != nil {
			return fmt.Errorf("unmarshal error: %v: %v: %v", reflect.TypeOf(ro), err, bodyString)
		}
		return ro
	}
}

func calculateQueryUrl(addr string, query interface{}) string {
	var parms []string
	u := "http://" + addr + "/"
	switch qt := query.(type) {
	case client.QueryEndpointsReq:
		u += "endpoints"
		if qt.Endpoint != nil {
			u = u + "/" + getNameFromResource(qt.Endpoint)
			break
		}
		parms = appendStringParm(parms, queryhdr.QuerySelector, qt.Selector)
		parms = appendStringParm(parms, queryhdr.QueryUnprotected, strconv.FormatBool(qt.Unprotected))
		parms = appendStringParm(parms, queryhdr.QueryUnlabelled, fmt.Sprint(qt.Unlabelled))
		parms = appendStringParm(parms, queryhdr.QueryNode, qt.Node)
		parms = appendResourceParm(parms, queryhdr.QueryPolicy, qt.Policy)
		parms = appendStringParm(parms, queryhdr.QueryRuleDirection, qt.RuleDirection)
		parms = appendStringParm(parms, queryhdr.QueryRuleIndex, fmt.Sprint(qt.RuleIndex))
		parms = appendStringParm(parms, queryhdr.QueryRuleEntity, qt.RuleEntity)
		parms = appendStringParm(parms, queryhdr.QueryRuleNegatedSelector, fmt.Sprint(qt.RuleNegatedSelector))
		parms = appendPageParms(parms, qt.Page)
		parms = appendSortParms(parms, qt.Sort)
	case client.QueryPoliciesReq:
		u += "policies"
		if qt.Policy != nil {
			u = u + "/" + getNameFromResource(qt.Policy)
			break
		}
		parms = appendResourceParm(parms, queryhdr.QueryEndpoint, qt.Endpoint)
		parms = appendResourceParm(parms, queryhdr.QueryNetworkSet, qt.NetworkSet)
		parms = appendStringParm(parms, queryhdr.QueryTier, qt.Tier)
		parms = appendStringParm(parms, queryhdr.QueryUnmatched, fmt.Sprint(qt.Unmatched))
		for k, v := range qt.Labels {
			parms = append(parms, queryhdr.QueryLabelPrefix+k+"="+v)
		}
		parms = appendPageParms(parms, qt.Page)
		parms = appendSortParms(parms, qt.Sort)
	case client.QueryNodesReq:
		u += "nodes"
		if qt.Node != nil {
			u = u + "/" + getNameFromResource(qt.Node)
			break
		}
		parms = appendPageParms(parms, qt.Page)
		parms = appendSortParms(parms, qt.Sort)
	case client.QueryClusterReq:
		u += "summary?from=now-15m&to=now-0m"
	}

	if len(parms) == 0 {
		return u
	}
	return u + "?" + strings.Join(parms, "&")
}

func appendPageParms(parms []string, page *client.Page) []string {
	if page == nil {
		return append(parms, queryhdr.QueryNumPerPage+"="+queryhdr.AllResults)
	}
	return append(parms,
		fmt.Sprintf("%s=%d", queryhdr.QueryPageNum, page.PageNum),
		fmt.Sprintf("%s=%d", queryhdr.QueryNumPerPage, page.NumPerPage),
	)
}

func appendSortParms(parms []string, sort *client.Sort) []string {
	if sort == nil {
		return parms
	}
	for _, f := range sort.SortBy {
		parms = append(parms, fmt.Sprintf("%s=%s", queryhdr.QuerySortBy, f))
	}
	return append(parms, fmt.Sprintf("%s=%v", queryhdr.QueryReverseSort, sort.Reverse))
}

func appendStringParm(parms []string, key, value string) []string {
	if value == "" {
		return parms
	}
	return append(parms, key+"="+url.QueryEscape(value))
}

func appendResourceParm(parms []string, key string, value model.Key) []string {
	if value == nil {
		return parms
	}
	return append(parms, key+"="+getNameFromResource(value))
}

func getNameFromResource(k model.Key) string {
	rk := k.(model.ResourceKey)
	if rk.Namespace != "" {
		return rk.Namespace + "/" + rk.Name
	}
	return rk.Name
}

func crossCheckPolicyQuery(tqd testQueryData, addr string, netClient *http.Client) {
	qpr, ok := tqd.response.(*client.QueryPoliciesResp)
	if !ok {
		// Don't attempt to cross check errored queries since we have nothing to cross-check.
		return
	}
	for _, p := range qpr.Items {
		policy := p.Name
		if p.Namespace != "" {
			policy = p.Namespace + "/" + policy
		}

		By(fmt.Sprintf("Running endpoint query for policy: %s", policy))
		qurl := "http://" + addr + "/endpoints?policy=" + policy + "&page=all"

		r, err := netClient.Get(qurl)
		Expect(err).NotTo(HaveOccurred())
		defer r.Body.Close()
		bodyBytes, err := io.ReadAll(r.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.StatusCode).To(Equal(http.StatusOK))
		output := client.QueryEndpointsResp{}
		err = json.Unmarshal(bodyBytes, &output)
		Expect(err).NotTo(HaveOccurred())
		var numWeps, numHeps int
		for _, i := range output.Items {
			if i.Kind == libapi.KindWorkloadEndpoint {
				numWeps++
			} else {
				numHeps++
			}
		}
		Expect(numHeps).To(Equal(p.NumHostEndpoints))
		Expect(numWeps).To(Equal(p.NumWorkloadEndpoints))
	}
}

func crossCheckEndpointQuery(tqd testQueryData, addr string, netClient *http.Client) {
	qpr, ok := tqd.response.(*client.QueryEndpointsResp)
	if !ok {
		// Don't attempt to cross check errored queries since we have nothing to cross-check.
		return
	}
	for _, p := range qpr.Items {
		endpoint := p.Name
		if p.Namespace != "" {
			endpoint = p.Namespace + "/" + endpoint
		}

		By(fmt.Sprintf("Running policy query for endpoint: %s", endpoint))
		qurl := "http://" + addr + "/policies?endpoint=" + endpoint + "&page=all"

		r, err := netClient.Get(qurl)
		Expect(err).NotTo(HaveOccurred())
		defer r.Body.Close()
		bodyBytes, err := io.ReadAll(r.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.StatusCode).To(Equal(http.StatusOK))
		output := client.QueryPoliciesResp{}
		err = json.Unmarshal(bodyBytes, &output)
		Expect(err).NotTo(HaveOccurred())
		var numNps, numGnps int
		for _, i := range output.Items {
			if i.Kind == apiv3.KindNetworkPolicy {
				numNps++
			} else {
				numGnps++
			}
		}
		Expect(numNps).To(Equal(p.NumNetworkPolicies))
		Expect(numGnps).To(Equal(p.NumGlobalNetworkPolicies))
	}
}

// getDummyConfigFromEnvFv returns the server configuration variables meant for FV tests.
func getDummyConfigFromEnvFv(addr, webKey, webCert string) *config.Config {
	config := &config.Config{
		ListenAddr: addr,
		TLSCert:    webCert,
		TLSKey:     webKey,
	}

	return config
}

type mockHandler struct {
}

func (mh *mockHandler) AuthenticationHandler(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		handlerFunc.ServeHTTP(w, req)
	}
}

//TODO(rlb):
// - reorder policies
// - re-node a HostEndpoint
