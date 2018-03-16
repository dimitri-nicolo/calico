package fv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calicoctl/calicoctl/resourcemgr"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/testutils"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
	"github.com/tigera/calicoq/web/queryserver/handlers"
	"github.com/tigera/calicoq/web/queryserver/server"
)

var _ = testutils.E2eDatastoreDescribe("Node tests", testutils.DatastoreEtcdV3, func(config apiconfig.CalicoAPIConfig) {

	DescribeTable("Query tests",
		func(tqds []testQueryData) {
			By("Creating a v3 client interface")
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			By("Cleaning the datastore")
			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			be.Clean()

			// Choose an arbitrary port for the server to listen on.
			By("Choosing an arbitrary available local port for the queryserver")
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			addr := listener.Addr().String()
			listener.Close()

			By("Starting the queryserver")
			server.Start(addr, &config, "", "")
			defer server.Stop()

			var configured map[model.ResourceKey]resourcemgr.ResourceObject
			var netClient = &http.Client{Timeout: time.Second * 10}
			for _, tqd := range tqds {
				By(fmt.Sprintf("Creating the resources for test: %s", tqd.description))
				configured = createResources(c, tqd.resources, configured)

				By(fmt.Sprintf("Calculating the URL for the test: %s", tqd.description))
				url := calculateQueryUrl(addr, tqd.query)

				By(fmt.Sprintf("Running query for test: %s", tqd.description))
				queryFn := func() interface{} {
					// Return the result if we have it, otherwise the error, this allows us to use Eventually to
					// check both values and errors.
					log.WithField("url", url).Debug("Running query")
					r, err := netClient.Get(url)
					if err != nil {
						return err
					}
					defer r.Body.Close()
					bodyBytes, err := ioutil.ReadAll(r.Body)
					if err != nil {
						return err
					}
					bodyString := string(bodyBytes)
					if r.StatusCode != http.StatusOK {
						return errors.New(strings.TrimSpace(bodyString))
					}

					// The response body should be json and the same type as the expected response object.
					ro := reflect.New(reflect.TypeOf(tqd.response).Elem()).Interface()
					err = json.Unmarshal(bodyBytes, ro)
					if err != nil {
						return fmt.Errorf("unmarshal error: %v: %v: %v", reflect.TypeOf(ro), err, bodyString)
					}
					return ro
				}
				Eventually(queryFn).Should(Equal(tqd.response))
				Consistently(queryFn).Should(Equal(tqd.response))

				By(fmt.Sprintf("Reapplying the same resources for test: %s", tqd.description))
				configured = createResources(c, tqd.resources, configured)

				By(fmt.Sprintf("Re-running the same query for test: %s", tqd.description))
				Eventually(queryFn).Should(Equal(tqd.response))
				Consistently(queryFn).Should(Equal(tqd.response))
			}
		},

		Entry("Summary queries", summaryTestQueryData()),
		Entry("Node queries", nodeTestQueryData()),
		Entry("Endpoint queries", endpointTestQueryData()),
		Entry("Policy queries", policyTestQueryData()),
	)
})

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
		parms = appendStringParm(parms, handlers.QuerySelector, qt.Selector)
		parms = appendStringParm(parms, handlers.QueryNode, qt.Node)
		parms = appendResourceParm(parms, handlers.QueryPolicy, qt.Policy)
		parms = appendStringParm(parms, handlers.QueryRuleDirection, qt.RuleDirection)
		parms = appendStringParm(parms, handlers.QueryRuleIndex, fmt.Sprint(qt.RuleIndex))
		parms = appendStringParm(parms, handlers.QueryRuleEntity, qt.RuleEntity)
		parms = appendStringParm(parms, handlers.QueryRuleNegatedSelector, fmt.Sprint(qt.RuleNegatedSelector))
		parms = appendPageParms(parms, qt.Page)
		parms = appendSortParms(parms, qt.Sort)
	case client.QueryPoliciesReq:
		u += "policies"
		if qt.Policy != nil {
			u = u + "/" + getNameFromResource(qt.Policy)
			break
		}
		parms = appendResourceParm(parms, handlers.QueryEndpoint, qt.Endpoint)
		parms = appendStringParm(parms, handlers.QueryTier, qt.Tier)
		parms = appendStringParm(parms, handlers.QueryUnmatched, fmt.Sprint(qt.Unmatched))
		for k, v := range qt.Labels {
			parms = append(parms, handlers.QueryLabelPrefix+k+"="+v)
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
		u += "summary"
	}

	if len(parms) == 0 {
		return u
	}
	return u + "?" + strings.Join(parms, "&")
}

func appendPageParms(parms []string, page *client.Page) []string {
	if page == nil {
		return append(parms, handlers.QueryNumPerPage+"="+handlers.AllResults)
	}
	return append(parms,
		fmt.Sprintf("%s=%d", handlers.QueryPageNum, page.PageNum),
		fmt.Sprintf("%s=%d", handlers.QueryNumPerPage, page.NumPerPage),
	)
}

func appendSortParms(parms []string, sort *client.Sort) []string {
	if sort == nil {
		return parms
	}
	for _, f := range sort.SortBy {
		parms = append(parms, fmt.Sprintf("%s=%s", handlers.QuerySortBy, f))
	}
	return append(parms, fmt.Sprintf("%s=%v", handlers.QueryReverseSort, sort.Reverse))
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

//TODO(rlb):
// - reorder policies
// - re-node a HostEndpoint
