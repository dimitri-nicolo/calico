// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"strings"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	clientv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3/fake"
	"github.com/tigera/compliance/pkg/report"
	"github.com/tigera/compliance/pkg/server"
)

// startTester starts and returns a server tester. This can be used to issue summary and report queries and to
// validate the response.
//
// The Calico and Report stores are mocked out, and the responses controlled via the control paraameters in the
// tester struct.
func startTester() *tester {
	// Create a new tester
	t := &tester{}

	// Choose an arbitrary port for the server to listen on.
	By("Choosing an arbitrary available local port for the queryserver")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	t.addr = listener.Addr().String()
	listener.Close()
	Expect(err).NotTo(HaveOccurred())

	By("Starting the compliance server")
	s := server.New(t, t, t.addr, "", "")
	s.Start()
	t.server = s
	t.client = &http.Client{Timeout: time.Second * 10}

	By("Waiting for a successful response from the version endpoint")
	get := func() error {
		_, err := t.client.Get("http://" + t.addr + "/compliance/version")
		return err
	}
	Eventually(get, "5s", "0.1s").Should(Succeed())

	return t
}

type tester struct {
	// Control parameters for the Calico Report List response.
	reportList    *v3.GlobalReportList
	reportListErr error

	// Control parameters for the Calico ReportType List response.
	reportTypeList    *v3.GlobalReportTypeList
	reportTypeListErr error

	// Control parameters for the archived ReportData list summaries response.
	summaries    []*report.ArchivedReportData
	summariesErr error

	// Control parameters for the archived ReportData get summary response.
	report    *report.ArchivedReportData
	reportErr error

	// Internal data for managing the server and client.
	addr   string
	server server.Server
	client *http.Client
}

type forecastFile struct {
	Format      string
	FileContent string
}

// stop will stop the test server.
func (t *tester) stop() {
	t.server.Stop()

	// Verify a call to Wait will actually finish.
	var finished bool
	go func() {
		t.server.Wait()
		finished = true
	}()
	Eventually(func() bool { return finished }, "5s", "0.1s").Should(BeTrue())
}

// list will issue a list of the report summaries via a client request, and will check the response against
// the provided expectation parameters.
func (t *tester) list(expStatus int, exp []server.Report) {
	listUrl := "http://" + t.addr + "/compliance/reports"
	r, err := t.client.Get(listUrl)
	Expect(err).NotTo(HaveOccurred())
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	Expect(err).NotTo(HaveOccurred())

	Expect(r.StatusCode).To(Equal(expStatus))
	if expStatus == http.StatusOK {
		output := &server.ReportList{}
		err = json.Unmarshal(bodyBytes, output)
		Expect(err).NotTo(HaveOccurred())
		Expect(output.Reports).To(HaveLen(len(exp)))
		Expect(output.Reports).To(Equal(exp))
	}
}

func (t *tester) downloadSingle(id string, expStatus int, forecast forecastFile) {
	downloadUrl := "http://" + t.addr + "/compliance/reports/" + id + "/download?format=" + forecast.Format
	r, err := t.client.Get(downloadUrl)
	Expect(err).NotTo(HaveOccurred())

	Expect(r.StatusCode).To(Equal(expStatus))

	//if we were not testing for an OK status we are done
	if expStatus != http.StatusOK {
		return
	}

	//check the file type that was downloaded
	condisp := r.Header.Get("Content-Disposition")
	Expect(condisp).Should(HavePrefix("attachment; filename="))
	Expect(condisp).Should(HaveSuffix(forecast.Format))

	// inspect the content
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	Expect(err).NotTo(HaveOccurred())

	Expect(strings.TrimSpace(string(bodyBytes))).To(Equal(forecast.FileContent))
}

func (t *tester) downloadMulti(id string, expStatus int, forecasts []forecastFile) {
	var fmts []string
	for _, v := range forecasts {
		fmts = append(fmts, v.Format)
	}
	formats := strings.Join(fmts, ",")

	downloadUrl := "http://" + t.addr + "/compliance/reports/" + id + "/download?format=" + formats
	r, err := t.client.Get(downloadUrl)
	Expect(err).NotTo(HaveOccurred())

	Expect(r.StatusCode).To(Equal(expStatus))

	//if we were not testing for an OK status we are done
	if expStatus != http.StatusOK {
		return
	}

	//check the file type that was downloaded
	condisp := r.Header.Get("Content-Disposition")
	Expect(condisp).Should(HavePrefix("attachment; filename="))
	Expect(condisp).Should(HaveSuffix(".zip"))

	//inspect the content
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	Expect(err).NotTo(HaveOccurred())

	//unzip the the file
	breader := bytes.NewReader(bodyBytes)
	zr, err := zip.NewReader(breader, int64(len(bodyBytes)))
	Expect(err).NotTo(HaveOccurred())

	//extract the files	into the files structure
	var files = make(map[string][]byte)
	for _, f := range zr.File {
		freader, err := f.Open()
		Expect(err).NotTo(HaveOccurred())
		var b bytes.Buffer
		io.Copy(&b, freader)
		files[f.Name] = b.Bytes()
	}

	//determine the base file name we should be looking for based on the zip file
	base := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(condisp, "attachment; filename="), ".zip"))

	//validate the file names and file content
	for _, fc := range forecasts {
		fn := fmt.Sprintf("%s-%s", base, fc.Format)
		Expect(files).To(HaveKeyWithValue(fn, []byte(fc.FileContent)))
	}

}

// RetrieveArchivedReport implements the ReportRetriever interface.
func (t *tester) RetrieveArchivedReport(id string) (*report.ArchivedReportData, error) {
	return t.report, t.reportErr
}

// RetrieveArchivedReportSummaries implements the ReportRetriever interface.
func (t *tester) RetrieveArchivedReportSummaries() ([]*report.ArchivedReportData, error) {
	return t.summaries, t.summariesErr
}

// GlobalReports implements the GlobalReportsGetter interface.
func (t *tester) GlobalReports() clientv3.GlobalReportInterface {
	return &gr{tester: t}
}

// GlobalReportTypes implements the GlobalReportTypesGetter interface.
func (t *tester) GlobalReportTypes() clientv3.GlobalReportTypeInterface {
	return &grt{tester: t}
}

// grt implements the GlobalReportTypeInterface
type grt struct {
	fake.FakeGlobalReportTypes
	tester *tester
}

// List overrides the default GlobalReportTypeInterface provided by FakeGlobalReportTypes to allow us
// to control the response to the List query.
func (g *grt) List(opts v1.ListOptions) (*v3.GlobalReportTypeList, error) {
	return g.tester.reportTypeList, g.tester.reportTypeListErr
}

// gr implements the GlobalReportInterface
type gr struct {
	fake.FakeGlobalReports
	tester *tester
}

// List overrides the default GlobalReportInterface provided by FakeGlobalReports to allow us
// to control the response to the List query.
func (g *gr) List(opts v1.ListOptions) (*v3.GlobalReportList, error) {
	return g.tester.reportList, g.tester.reportListErr
}
