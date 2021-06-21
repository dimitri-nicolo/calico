// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handlers_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/projectcalico/libcalico-go/lib/errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tigera/packetcapture-api/pkg/cache"
	"github.com/tigera/packetcapture-api/pkg/capture"
	"github.com/tigera/packetcapture-api/pkg/handlers"
	"github.com/tigera/packetcapture-api/pkg/middleware"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	calicolib "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

const loremLipsum = "Lorem Lipsum"

var _ = Describe("Download", func() {
	var req *http.Request

	var files = []string{"a", "b"}
	var otherFiles = []string{"c", "d"}
	var packetCaptureOneNode = &v3.PacketCapture{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "ns",
		},
		Status: calicolib.PacketCaptureStatus{
			Files: []calicolib.PacketCaptureFile{
				{
					Node:      "node",
					Directory: "dir",
					FileNames: files,
				},
			},
		},
	}
	var packetCaptureMultipleNodes = &v3.PacketCapture{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "ns",
		},
		Status: calicolib.PacketCaptureStatus{
			Files: []calicolib.PacketCaptureFile{
				{
					Node:      "nodeOne",
					Directory: "dir",
					FileNames: files,
				},
				{
					Node:      "nodeTwo",
					Directory: "dir",
					FileNames: otherFiles,
				},
			},
		},
	}

	var point = capture.EntryPoint{PodName: "entryPod", PodNamespace: "entryNs", CaptureDirectory: "dir",
		CaptureNamespace: "ns", CaptureName: "name"}
	var pointNodeOne = capture.EntryPoint{PodName: "entryPodOne", PodNamespace: "entryNs", CaptureDirectory: "dir",
		CaptureNamespace: "ns", CaptureName: "name"}
	var pointNodeTwo = capture.EntryPoint{PodName: "entryPodTwo", PodNamespace: "entryNs", CaptureDirectory: "dir",
		CaptureNamespace: "ns", CaptureName: "name"}

	BeforeEach(func() {
		// Create a new request
		var err error
		req, err = http.NewRequest("GET", "/download/ns/name?files.zip", nil)
		Expect(err).NotTo(HaveOccurred())

		// Setup the variables on the context to be used for authN/authZ
		req = req.WithContext(middleware.WithClusterID(req.Context(), "cluster"))
		req = req.WithContext(middleware.WithNamespace(req.Context(), "ns"))
		req = req.WithContext(middleware.WithCaptureName(req.Context(), "name"))
	})

	It("Downloads files from a single node", func() {
		// Create a temp directory to store all the files needed for the test
		var tempDir, err = ioutil.TempDir("/tmp", "test")
		Expect(err).NotTo(HaveOccurred())

		// Create dummy files and add them to a tar archive
		var tarFile = createTarArchive(tempDir, files)
		defer os.RemoveAll(tempDir)

		tarFileReader, err := os.Open(tarFile.Name())
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockLocator = &capture.MockLocator{}
		var mockFileRetrieval = &capture.MockFileRetrieval{}
		mockLocator.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockLocator.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", point).Return(tarFileReader, nil, nil)
		var download = handlers.NewDownload(mockCache, mockLocator, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Download)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Header().Get("Content-Type")).To(Equal("application/zip"))
		Expect(recorder.Header().Get("Content-Disposition")).To(Equal("attachment; filename=files.zip"))
		Expect(recorder.Header().Get("Content-Length")).NotTo(Equal(""))

		archive, err := ioutil.TempFile(tempDir, "result.*.zip")
		Expect(err).NotTo(HaveOccurred())

		// Write the body to file
		_, err = io.Copy(archive, recorder.Body)
		Expect(err).NotTo(HaveOccurred())
		validateArchive(archive, files)
	})

	It("Downloads files from a multiple node", func() {
		// Create a temp directory to store all the files needed for the test
		var tempDir, err = ioutil.TempDir("/tmp", "test")
		Expect(err).NotTo(HaveOccurred())

		// Create dummy files and add them to a tar archive
		var tarFileNodeOne = createTarArchive(tempDir, files)
		var tarFileNodeTwo = createTarArchive(tempDir, otherFiles)
		defer os.RemoveAll(tempDir)

		tarFileReaderOne, err := os.Open(tarFileNodeOne.Name())
		Expect(err).NotTo(HaveOccurred())
		tarFileReaderTwo, err := os.Open(tarFileNodeTwo.Name())
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockLocator = &capture.MockLocator{}
		var mockFileRetrieval = &capture.MockFileRetrieval{}
		mockLocator.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureMultipleNodes, nil)
		mockLocator.On("GetEntryPod", "cluster", "nodeOne").Return("entryNs", "entryPodOne", nil)
		mockLocator.On("GetEntryPod", "cluster", "nodeTwo").Return("entryNs", "entryPodTwo", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", pointNodeOne).Return(tarFileReaderOne, nil, nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", pointNodeTwo).Return(tarFileReaderTwo, nil, nil)
		var download = handlers.NewDownload(mockCache, mockLocator, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Download)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Header().Get("Content-Type")).To(Equal("application/zip"))
		Expect(recorder.Header().Get("Content-Disposition")).To(Equal("attachment; filename=files.zip"))
		Expect(recorder.Header().Get("Content-Length")).NotTo(Equal(""))

		archive, err := ioutil.TempFile(tempDir, "result.*.zip")
		Expect(err).NotTo(HaveOccurred())

		// Write the body to file
		_, err = io.Copy(archive, recorder.Body)
		Expect(err).NotTo(HaveOccurred())
		var allFiles []string
		allFiles = append(allFiles, files...)
		allFiles = append(allFiles, otherFiles...)
		validateArchive(archive, allFiles)
	})

	DescribeTable("Failure to get packet capture",
		func(expectedStatus int, expectedError error) {
			// Bootstrap the download
			var mockCache = &cache.MockClientCache{}
			var mockLocator = &capture.MockLocator{}
			var mockFileRetrieval = &capture.MockFileRetrieval{}
			mockLocator.On("GetPacketCapture", "cluster", "name", "ns").Return(nil, expectedError)
			var download = handlers.NewDownload(mockCache, mockLocator, mockFileRetrieval)

			// Bootstrap the http recorder
			recorder := httptest.NewRecorder()
			handler := http.HandlerFunc(download.Download)
			handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(expectedStatus))
			Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal(expectedError.Error()))
		},
		Entry("Missing resource", http.StatusNotFound, errors.ErrorResourceDoesNotExist{}),
		Entry("Failure to get resource", http.StatusInternalServerError, fmt.Errorf("any error")),
	)

	It("TarError returns an error via io.Reader", func() {
		var errorWriter bytes.Buffer
		var _, err = errorWriter.WriteString("any error")
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockLocator = &capture.MockLocator{}
		var mockFileRetrieval = &capture.MockFileRetrieval{}
		mockLocator.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockLocator.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", point).Return(nil, &errorWriter, nil)
		var download = handlers.NewDownload(mockCache, mockLocator, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Download)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal("any error"))
	})

	It("Fails to locate an entry pod", func() {
		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockLocator = &capture.MockLocator{}
		var mockFileRetrieval = &capture.MockFileRetrieval{}
		mockLocator.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockLocator.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", fmt.Errorf("any error"))
		var download = handlers.NewDownload(mockCache, mockLocator, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Download)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal("any error"))
	})
})

func validateArchive(archive *os.File, files []string) {
	defer GinkgoRecover()

	zipReader, err := zip.OpenReader(archive.Name())
	Expect(err).NotTo(HaveOccurred())
	Expect(len(zipReader.File)).To(Equal(len(files)))

	for _, f := range zipReader.File {
		file, err := f.Open()
		Expect(err).NotTo(HaveOccurred())
		var content bytes.Buffer
		_, err = io.Copy(&content, file)
		Expect(err).NotTo(HaveOccurred())

		Expect(content.String()).To(Equal(loremLipsum))
		file.Close()
	}
}

func createTarArchive(dir string, files []string) *os.File {
	defer GinkgoRecover()

	// Create the file for the tar archive
	var tarFile, err = ioutil.TempFile(dir, "archive.*.tar")
	Expect(err).NotTo(HaveOccurred())

	// Archive the file to the tar archive
	var tarWriter = tar.NewWriter(tarFile)

	for _, file := range files {
		// Create a temporary file with some random data in it
		file, err := ioutil.TempFile(dir, fmt.Sprintf("%s.*.txt", file))
		Expect(err).NotTo(HaveOccurred())
		_, err = file.Write([]byte(loremLipsum))
		Expect(err).NotTo(HaveOccurred())
		file.Close()

		// Open a reader for the file
		fileReader, err := os.Open(file.Name())
		Expect(err).NotTo(HaveOccurred())

		// Write the file header to the archive
		info, err := fileReader.Stat()
		Expect(err).NotTo(HaveOccurred())
		header, err := tar.FileInfoHeader(info, info.Name())
		Expect(err).NotTo(HaveOccurred())
		header.Name = fileReader.Name()
		err = tarWriter.WriteHeader(header)
		Expect(err).NotTo(HaveOccurred())

		// Write the content to the tar archive
		_, err = io.Copy(tarWriter, fileReader)
		Expect(err).NotTo(HaveOccurred())

		fileReader.Close()
	}

	tarWriter.Flush()
	tarWriter.Close()

	return tarFile
}
