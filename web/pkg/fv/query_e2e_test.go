package fv

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calicoctl/calicoctl/resourcemgr"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/testutils"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
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

			By("Creating a query interface")
			qi := client.NewQueryInterface(c)
			var configured map[model.ResourceKey]resourcemgr.ResourceObject

			for _, tqd := range tqds {
				By(fmt.Sprintf("Creating the resources for test: %s", tqd.description))
				configured = createResources(c, tqd.resources, configured)

				By(fmt.Sprintf("Running query for test: %s", tqd.description))
				queryFn := func() interface{} {
					// Return the result if we have it, otherwise the error, this allows us to use Eventually to
					// check both values and errors.
					r, err := qi.RunQuery(context.Background(), tqd.query)
					if err != nil {
						return err
					}
					return r
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

//TODO(rlb):
// - reorder policies
// - re-node a HostEndpoint
