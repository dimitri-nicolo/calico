package waf_test

import (
	"os"
	"path/filepath"
	"testing"

	_ "embed"

	envoyauthz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/code"

	"github.com/projectcalico/calico/app-policy/internal/util/testutils"
	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/waf"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

//go:embed testdata/tigera.conf
var tigeraConfContents string

func TestCorazaWAFAuthzScenarios(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	for _, scenario := range corazaWAFScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runCorazaWAFAuthzScenario(t, &scenario)
		})
	}
}

var (
	corazaWAFScenarios = []corazaWAFScenario{
		{
			name:  "allow",
			store: nil,
			directives: []string{
				"Include @coraza.conf-recommended",
				"Include @crs-setup.conf.example",
				"Include @owasp_crs/*.conf",
				"SecRuleEngine On",
			},
			checkReq:         testutils.NewCheckRequestBuilder(),
			expectedResponse: waf.OK,
			expectedErr:      nil,
			expectedLogs:     nil,
		},
		{
			name:       "deny - SQL injection 1",
			store:      nil,
			directives: []string{},
			additionalConfigFiles: map[string]string{
				"tigera.conf": tigeraConfContents,
			},
			checkReq: testutils.NewCheckRequestBuilder(
				testutils.WithMethod("GET"),
				testutils.WithHost("my.loadbalancer.address"),
				testutils.WithPath("/cart?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"),
			),
			expectedResponse: waf.DENY,
			expectedErr:      nil,
			expectedLogs: []*v1.WAFLog{
				{},
			},
		},
		{
			name:  "deny - SQL injection 2, detection only",
			store: nil,
			directives: []string{
				"Include @coraza.conf-recommended",
				"Include @crs-setup.conf.example",
				"Include @owasp_crs/*.conf",
				"SecRuleEngine DetectionOnly",
			},
			checkReq: testutils.NewCheckRequestBuilder(
				testutils.WithMethod("POST"),
				testutils.WithHost("www.example.com"),
				testutils.WithPath("/vulnerable.php?id=1' waitfor delay '00:00:10'--"),
				testutils.WithScheme("https"),
			),
			expectedResponse: waf.OK,
			expectedErr:      nil,
			expectedLogs: []*v1.WAFLog{
				{},
			},
		},
	}
)

func runCorazaWAFAuthzScenario(t testing.TB, scenario *corazaWAFScenario) {
	psm := policystore.NewPolicyStoreManager()
	psm.OnInSync()

	tempDir := t.TempDir()
	files := []string{}
	for name, content := range scenario.additionalConfigFiles {
		t.Log("Writing additional config file", tempDir, name)
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %s", name, err)
		}
		// append the file to the directives
		files = append(files, filepath.Join(tempDir, name))
	}
	var observedLogs []*v1.WAFLog
	cb := func(v interface{}) {
		if v, ok := v.(*v1.WAFLog); ok {
			observedLogs = append(observedLogs, v)
		}
	}
	evp := waf.NewEventsPipeline(cb)
	waf, err := waf.New(
		files,
		scenario.directives,
		true,
		evp,
	)
	if err != nil {
		t.Fatalf("Failed to create WAF: %s", err)
	}

	resp, err := waf.Check(scenario.store, scenario.checkReq.Value())
	if err != scenario.expectedErr {
		t.Fatalf("Expected error %v, but got %v", scenario.expectedErr, err)
	}

	if resp.Status.Code != scenario.expectedResponse.Status.Code {
		t.Fatalf(
			"Expected response code %s, but got %s",
			code.Code(scenario.expectedResponse.Status.Code),
			code.Code(resp.Status.Code),
		)
	}

	evp.Flush()
	if len(observedLogs) != len(scenario.expectedLogs) {
		t.Fatalf("Expected %d logs, but got %d", len(scenario.expectedLogs), len(observedLogs))
	}
}

type corazaWAFScenario struct {
	name                  string
	directives            []string
	additionalConfigFiles map[string]string
	store                 *policystore.PolicyStore
	checkReq              *testutils.CheckRequestBuilder
	expectedResponse      *envoyauthz.CheckResponse
	expectedErr           error
	expectedLogs          []*v1.WAFLog
}
