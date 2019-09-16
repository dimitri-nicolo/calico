// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	k8s "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/tigera/compliance/pkg/server"
)

const (
	perform_list_report    = "perform_list_report"
	perform_get_report     = "perform_get_report"
	perform_get_reporttype = "perform_get_reporttype"
	perform_view_report    = "perform_view_report"
)

const (
	user_BasicUserAll     = "BasicUserAll"
	user_BasicUserLimited = "BasicUserLimited"
	user_BasicUserInvOnly = "BasicUserInvOnly"

	user_TokenUserAll     = "TokenUserAll"
	user_TokenUserLimited = "TokenUserLimited"
	user_TokenUserInvOnly = "TokenUserInvOnly"
)

const (
	on_report_type_inventory = "inventory"
	on_report_type_audit     = "audit"
)

const (
	on_report_AuditReport1     = "auditreport1"
	on_report_AuditReport2     = "auditreport2"
	on_report_InventoryReport1 = "inventoryreport1"
	on_report_InventoryReport2 = "inventoryreport2"
)

const (
	can    = true
	cannot = false
)

func authTestEntry(user string, authorize bool, operation, name string) TableEntry {
	if authorize {
		return Entry(fmt.Sprintf("User %s can %s on %s", user, operation, name), user, true, operation, name)
	} else {
		return Entry(fmt.Sprintf("User %s cannot %s on %s", user, operation, name), user, false, operation, name)
	}
}

func viewTestEntry(user string, authorize bool, operation, reportTypeName, reportName string) TableEntry {
	if authorize {
		return Entry(fmt.Sprintf("User %s can %s on %s/%s", user, operation, reportTypeName, reportName), user, true, operation, reportTypeName, reportName)
	} else {
		return Entry(fmt.Sprintf("User %s cannot %s on %s/%s", user, operation, reportTypeName, reportName), user, false, operation, reportTypeName, reportName)
	}
}

var _ = Describe("Authenticate against K8s apiserver", func() {
	var k8sClient k8s.Interface
	var k8sConfig restclient.Config

	var authHeaders = map[string][]string{
		user_BasicUserAll:     {fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("BasicUserAll:basicpw")))},
		user_BasicUserLimited: {fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("BasicUserLimited:basicpwl")))},
		user_BasicUserInvOnly: {fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("BasicUserInvOnly:basicpwio")))},
		user_TokenUserAll:     {"bearer a012345"},
		user_TokenUserLimited: {"bearer b2468AB"},
		user_TokenUserInvOnly: {"bearer c345678"},
	}

	BeforeEach(func() {
		k8sConfig = restclient.Config{}

		k8sConfig.Host = os.Getenv("KUBERNETES_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_SERVICE_PORT")
		if k8sConfig.Host == ":" {
			k8sConfig.Host = "https://localhost:6443"
		}

		k8sConfig.Insecure = true
		if k8sConfig.RateLimiter == nil && k8sConfig.QPS > 0 {
			k8sConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(k8sConfig.QPS, k8sConfig.Burst)
		}

		k8sClient = k8s.NewForConfigOrDie(&k8sConfig)
		Expect(k8sClient).NotTo(BeNil())

	})
	AfterEach(func() {
	})

	// This test validates that the rbac is configured correctly for the
	// composite test. If this test fails then rbac is not configured correctly
	// and the results of the composite tests are invalid.
	// See the test/rbac folder for the users (in *.csv) and roles and bindings for them.
	DescribeTable("Test rbac configuration",
		func(user string, can bool, operation string, name string) {

			req := &http.Request{Header: http.Header{"Authorization": authHeaders[user]}}

			k8sauth := newMock(user, can, operation, name)
			factory := server.NewStandardRbacHelperFactory(k8sauth)
			auth := factory.NewReportRbacHelper(req)

			var stat bool
			var err error

			switch operation {
			case perform_list_report:
				stat, err = auth.CanListReports()
			case perform_get_report:
				stat, err = auth.CanGetReport(name)
			case perform_get_reporttype:
				stat, err = auth.CanGetReportType(name)
			default:
				panic(fmt.Sprintf("Invalid operation in test: %s", operation))
			}

			if can {
				Expect(err).To(BeNil())
				Expect(stat).To(Equal(can), "Should be allowed")
			} else {
				Expect(err).To(Not(BeNil()))
				Expect(stat).To(Equal(cannot), "Should be denied")
			}
		},

		//basic-auth based tests ---------------------------------------------------------

		//BasicAll user has access to all reports and report types
		authTestEntry(user_BasicUserAll, can, perform_get_reporttype, on_report_type_audit),
		authTestEntry(user_BasicUserAll, can, perform_get_reporttype, on_report_type_inventory),
		authTestEntry(user_BasicUserAll, can, perform_list_report, on_report_AuditReport1),
		authTestEntry(user_BasicUserAll, can, perform_list_report, on_report_AuditReport2),
		authTestEntry(user_BasicUserAll, can, perform_get_report, on_report_AuditReport1),
		authTestEntry(user_BasicUserAll, can, perform_get_report, on_report_AuditReport2),
		authTestEntry(user_BasicUserAll, can, perform_list_report, on_report_InventoryReport1),
		authTestEntry(user_BasicUserAll, can, perform_list_report, on_report_InventoryReport2),
		authTestEntry(user_BasicUserAll, can, perform_get_report, on_report_InventoryReport1),
		authTestEntry(user_BasicUserAll, can, perform_get_report, on_report_InventoryReport2),

		//BasicUserLimited user has limited access
		//can access both types
		//can get and list inventory1report  cannot get or list inventory2report
		//can list audit1report and get audit2report
		authTestEntry(user_BasicUserLimited, can, perform_get_reporttype, on_report_type_audit),
		authTestEntry(user_BasicUserLimited, can, perform_get_reporttype, on_report_type_inventory),
		authTestEntry(user_BasicUserLimited, can, perform_list_report, on_report_AuditReport1),
		authTestEntry(user_BasicUserLimited, cannot, perform_list_report, on_report_AuditReport2),
		authTestEntry(user_BasicUserLimited, cannot, perform_get_report, on_report_AuditReport1),
		authTestEntry(user_BasicUserLimited, can, perform_get_report, on_report_AuditReport2),
		authTestEntry(user_BasicUserLimited, can, perform_list_report, on_report_InventoryReport1),
		authTestEntry(user_BasicUserLimited, cannot, perform_list_report, on_report_InventoryReport2),
		authTestEntry(user_BasicUserLimited, can, perform_get_report, on_report_InventoryReport1),
		authTestEntry(user_BasicUserLimited, cannot, perform_get_report, on_report_InventoryReport2),

		//BasicUserNoAudit user only has access to inventory type
		//however here does have access to list and get audit1report
		//this is for testing composite access rules
		authTestEntry(user_BasicUserInvOnly, cannot, perform_get_reporttype, on_report_type_audit),
		authTestEntry(user_BasicUserInvOnly, can, perform_get_reporttype, on_report_type_inventory),
		authTestEntry(user_BasicUserInvOnly, can, perform_list_report, on_report_AuditReport1),
		authTestEntry(user_BasicUserInvOnly, cannot, perform_list_report, on_report_AuditReport2),
		authTestEntry(user_BasicUserInvOnly, can, perform_get_report, on_report_AuditReport1),
		authTestEntry(user_BasicUserInvOnly, cannot, perform_get_report, on_report_AuditReport2),
		authTestEntry(user_BasicUserInvOnly, can, perform_list_report, on_report_InventoryReport1),
		authTestEntry(user_BasicUserInvOnly, can, perform_list_report, on_report_InventoryReport2),
		authTestEntry(user_BasicUserInvOnly, can, perform_get_report, on_report_InventoryReport1),
		authTestEntry(user_BasicUserInvOnly, cannot, perform_get_report, on_report_InventoryReport2),

		//Token based tests ---------------------------------------------------------

		//TokenAll user has access to all reports and report types
		authTestEntry(user_TokenUserAll, can, perform_get_reporttype, on_report_type_audit),
		authTestEntry(user_TokenUserAll, can, perform_get_reporttype, on_report_type_inventory),
		authTestEntry(user_TokenUserAll, can, perform_list_report, on_report_AuditReport1),
		authTestEntry(user_TokenUserAll, can, perform_list_report, on_report_AuditReport2),
		authTestEntry(user_TokenUserAll, can, perform_get_report, on_report_AuditReport1),
		authTestEntry(user_TokenUserAll, can, perform_get_report, on_report_AuditReport2),
		authTestEntry(user_TokenUserAll, can, perform_list_report, on_report_InventoryReport1),
		authTestEntry(user_TokenUserAll, can, perform_list_report, on_report_InventoryReport2),
		authTestEntry(user_TokenUserAll, can, perform_get_report, on_report_InventoryReport1),
		authTestEntry(user_TokenUserAll, can, perform_get_report, on_report_InventoryReport2),

		//TokenUserLimited user has limited access
		//can access both types
		//can get and list inventory1report  cannot get or list inventory2report
		//can list audit1report and get audit2report
		authTestEntry(user_TokenUserLimited, can, perform_get_reporttype, on_report_type_audit),
		authTestEntry(user_TokenUserLimited, can, perform_get_reporttype, on_report_type_inventory),
		authTestEntry(user_TokenUserLimited, can, perform_list_report, on_report_AuditReport1),
		authTestEntry(user_TokenUserLimited, cannot, perform_list_report, on_report_AuditReport2),
		authTestEntry(user_TokenUserLimited, cannot, perform_get_report, on_report_AuditReport1),
		authTestEntry(user_TokenUserLimited, can, perform_get_report, on_report_AuditReport2),
		authTestEntry(user_TokenUserLimited, can, perform_list_report, on_report_InventoryReport1),
		authTestEntry(user_TokenUserLimited, cannot, perform_list_report, on_report_InventoryReport2),
		authTestEntry(user_TokenUserLimited, can, perform_get_report, on_report_InventoryReport1),
		authTestEntry(user_TokenUserLimited, cannot, perform_get_report, on_report_InventoryReport2),

		//TokenUserNoAudit user only has access to inventory type
		//however here it does have access to list and get audit1report
		//this however will be denied becase the user does not have access to the report type
		//this is for testing composite access rules
		authTestEntry(user_TokenUserInvOnly, cannot, perform_get_report, on_report_type_audit),
		authTestEntry(user_TokenUserInvOnly, can, perform_get_reporttype, on_report_type_inventory),
		authTestEntry(user_TokenUserInvOnly, can, perform_list_report, on_report_AuditReport1),
		authTestEntry(user_TokenUserInvOnly, cannot, perform_list_report, on_report_AuditReport2),
		authTestEntry(user_TokenUserInvOnly, can, perform_get_report, on_report_AuditReport1),
		authTestEntry(user_TokenUserInvOnly, cannot, perform_get_report, on_report_AuditReport2),
		authTestEntry(user_TokenUserInvOnly, can, perform_list_report, on_report_InventoryReport1),
		authTestEntry(user_TokenUserInvOnly, can, perform_list_report, on_report_InventoryReport2),
		authTestEntry(user_TokenUserInvOnly, can, perform_get_report, on_report_InventoryReport1),
		authTestEntry(user_TokenUserInvOnly, cannot, perform_get_report, on_report_InventoryReport2),
	)

	DescribeTable("Test composite authorization",
		func(user string, can bool, operation string, reportTypeName, reportName string) {

			req := &http.Request{Header: http.Header{"Authorization": authHeaders[user]}}

			k8sauth := newMock(user, can, operation, reportName)
			factory := server.NewStandardRbacHelperFactory(k8sauth)
			auth := factory.NewReportRbacHelper(req)

			var stat bool
			var err error

			switch operation {
			case perform_view_report:
				stat, err = auth.CanViewReport(reportTypeName, reportName)
			default:
				panic(fmt.Sprintf("Invalid operation in test: %s", operation))
			}

			if can {
				Expect(err).To(BeNil())
				Expect(stat).To(Equal(can), "Should be allowed")
			} else {
				Expect(err).To(Not(BeNil()))
				Expect(stat).To(Equal(cannot), "Should be denied")
			}
		},

		//Basic-auth based tests ---------------------------------------------------------

		viewTestEntry(user_BasicUserAll, can, perform_view_report, on_report_type_inventory, on_report_InventoryReport1),
		viewTestEntry(user_BasicUserAll, can, perform_view_report, on_report_type_inventory, on_report_InventoryReport2),
		viewTestEntry(user_BasicUserAll, can, perform_view_report, on_report_type_audit, on_report_AuditReport1),
		viewTestEntry(user_BasicUserAll, can, perform_view_report, on_report_type_audit, on_report_AuditReport2),

		viewTestEntry(user_BasicUserLimited, can, perform_view_report, on_report_type_inventory, on_report_InventoryReport1),
		viewTestEntry(user_BasicUserLimited, cannot, perform_view_report, on_report_type_inventory, on_report_InventoryReport2),
		viewTestEntry(user_BasicUserLimited, cannot, perform_view_report, on_report_type_audit, on_report_AuditReport1),
		viewTestEntry(user_BasicUserLimited, can, perform_view_report, on_report_type_audit, on_report_AuditReport2),

		viewTestEntry(user_BasicUserInvOnly, can, perform_view_report, on_report_type_inventory, on_report_InventoryReport1),
		viewTestEntry(user_BasicUserInvOnly, cannot, perform_view_report, on_report_type_inventory, on_report_InventoryReport2),
		viewTestEntry(user_BasicUserInvOnly, cannot, perform_view_report, on_report_type_audit, on_report_AuditReport1),
		viewTestEntry(user_BasicUserInvOnly, cannot, perform_view_report, on_report_type_audit, on_report_AuditReport2),

		//Token based tests ---------------------------------------------------------

		viewTestEntry(user_TokenUserAll, can, perform_view_report, on_report_type_inventory, on_report_InventoryReport1),
		viewTestEntry(user_TokenUserAll, can, perform_view_report, on_report_type_inventory, on_report_InventoryReport2),
		viewTestEntry(user_TokenUserAll, can, perform_view_report, on_report_type_audit, on_report_AuditReport1),
		viewTestEntry(user_TokenUserAll, can, perform_view_report, on_report_type_audit, on_report_AuditReport2),

		viewTestEntry(user_TokenUserLimited, can, perform_view_report, on_report_type_inventory, on_report_InventoryReport1),
		viewTestEntry(user_TokenUserLimited, cannot, perform_view_report, on_report_type_inventory, on_report_InventoryReport2),
		viewTestEntry(user_TokenUserLimited, cannot, perform_view_report, on_report_type_audit, on_report_AuditReport1),
		viewTestEntry(user_TokenUserLimited, can, perform_view_report, on_report_type_audit, on_report_AuditReport2),

		viewTestEntry(user_TokenUserInvOnly, can, perform_view_report, on_report_type_inventory, on_report_InventoryReport1),
		viewTestEntry(user_TokenUserInvOnly, cannot, perform_view_report, on_report_type_inventory, on_report_InventoryReport2),
		viewTestEntry(user_TokenUserInvOnly, cannot, perform_view_report, on_report_type_audit, on_report_AuditReport1),
		viewTestEntry(user_TokenUserInvOnly, cannot, perform_view_report, on_report_type_audit, on_report_AuditReport2),
	)

})

type mock struct {
	shouldIt bool
}

func newMock(user string, can bool, operation string, name string) *mock {
	return &mock{
		shouldIt: can,
	}
}

// KubernetesAuthnAuthz to satisfy K8sAuthInterface
func (m *mock) KubernetesAuthnAuthz(handler http.Handler) http.Handler {
	return handler
}

// Authorized to satisfy K8sAuthInterface
func (m *mock) Authorize(req *http.Request) (status int, err error) {
	if m.shouldIt {
		return 0, nil
	} else {
		return http.StatusUnauthorized, fmt.Errorf("Error performing AccessReview")
	}
}
