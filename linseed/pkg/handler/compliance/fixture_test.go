package compliance

import (
	"time"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/list"
)

const dummyURL = "anyURL"

var (
	noReport []v1.ReportData

	// Create a basic report.
	v3report = apiv3.ReportData{
		ReportName:     "test-report",
		ReportTypeName: "my-report-type",
		StartTime:      metav1.Time{Time: time.Unix(1, 0)},
		EndTime:        metav1.Time{Time: time.Unix(2, 0)},
		GenerationTime: metav1.Time{Time: time.Unix(3, 0)},
	}
	report          = v1.ReportData{ReportData: &v3report}
	multipleReports = []v1.ReportData{
		report, report,
	}

	//Create a compliance benchmark
	noBenchmarks []v1.Benchmarks
	benchmarks   = v1.Benchmarks{
		Version:           "v1",
		KubernetesVersion: "v1.0",
		Type:              v1.TypeKubernetes,
		NodeName:          "lodestone",
		Timestamp:         metav1.Time{Time: time.Unix(1, 0)},
		Error:             "",
		Tests: []v1.BenchmarkTest{
			{
				Section:     "a.1",
				SectionDesc: "testing the test",
				TestNumber:  "1",
				TestDesc:    "making sure that we're right",
				TestInfo:    "information is fluid",
				Status:      "Just swell",
				Scored:      true,
			},
		},
	}
	multipleBenchmark = []v1.Benchmarks{benchmarks, benchmarks}

	//Create a compliance Snapshot
	noSnapshot []v1.Snapshot
	snapshots  = v1.Snapshot{
		ResourceList: list.TimestampedResourceList{
			ResourceList: &apiv3.NetworkPolicyList{
				TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
				ListMeta: metav1.ListMeta{},
				Items: []apiv3.NetworkPolicy{
					{
						TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "projectcalico.org/v3"},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "np1",
							Namespace: "default",
						},
					},
				},
			},
			RequestStartedTimestamp:   metav1.Time{Time: time.Unix(1, 0)},
			RequestCompletedTimestamp: metav1.Time{Time: time.Unix(2, 0)},
		},
	}
	multipleSnapshot = []v1.Snapshot{snapshots, snapshots}
)

var bulkResponseSuccess = &v1.BulkResponse{
	Total:     2,
	Succeeded: 2,
	Failed:    0,
}

var bulkResponsePartialSuccess = &v1.BulkResponse{
	Total:     2,
	Succeeded: 1,
	Failed:    1,
	Errors: []v1.BulkError{
		{
			Resource: "res",
			Type:     "index error",
			Reason:   "I couldn't do it",
		},
	},
}
