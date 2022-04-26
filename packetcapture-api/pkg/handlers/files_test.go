// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handlers_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/packetcapture-api/pkg/cache"
	"github.com/projectcalico/calico/packetcapture-api/pkg/capture"
	"github.com/projectcalico/calico/packetcapture-api/pkg/handlers"
	"github.com/projectcalico/calico/packetcapture-api/pkg/middleware"
)

const loremLipsum = "Lorem Lipsum"

var _ = Describe("FilesDownload", func() {
	var req *http.Request

	BeforeEach(func() {
		// Create a new request
		var err error
		req, err = http.NewRequest("GET", "/download/ns/name/files.zip", nil)
		Expect(err).NotTo(HaveOccurred())

		// Setup the variables on the context to be used for authN/authZ
		req = req.WithContext(middleware.WithClusterID(req.Context(), "cluster"))
		req = req.WithContext(middleware.WithNamespace(req.Context(), "ns"))
		req = req.WithContext(middleware.WithCaptureName(req.Context(), "name"))
	})

	It("Can archive a pcap file with only its header", func() {
		// Create a temp directory to store all the files needed for the test
		var tempDir, err = ioutil.TempDir("/tmp", "test")
		Expect(err).NotTo(HaveOccurred())

		// Create dummy files and add them to a tar archive
		var tarFile = createTarArchive(tempDir, files, pcapHeader())
		defer os.RemoveAll(tempDir)

		tarFileReader, err := os.Open(tarFile.Name())
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", point).Return(tarFileReader, nil, nil)
		var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

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
		validateArchive(archive, files, pcapHeader())
	})

	It("Downloads files from a single node", func() {
		// Create a temp directory to store all the files needed for the test
		var tempDir, err = ioutil.TempDir("/tmp", "test")
		Expect(err).NotTo(HaveOccurred())

		// Create dummy files and add them to a tar archive
		var tarFile = createTarArchive(tempDir, files, []byte(loremLipsum))
		defer os.RemoveAll(tempDir)

		tarFileReader, err := os.Open(tarFile.Name())
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", point).Return(tarFileReader, nil, nil)
		var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

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
		validateArchive(archive, files, []byte(loremLipsum))
	})

	It("Downloads files from 0 files", func() {
		// Create a temp directory to store all the files needed for the test
		var tempDir, err = ioutil.TempDir("/tmp", "test")
		Expect(err).NotTo(HaveOccurred())

		// Create dummy files and add them to a tar archive
		var tarFile = createTarArchive(tempDir, noFiles, []byte(loremLipsum))
		defer os.RemoveAll(tempDir)

		tarFileReader, err := os.Open(tarFile.Name())
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureNoFiles, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", point).Return(tarFileReader, nil, nil)
		var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Download)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusNoContent))
		Expect(recorder.Body.String()).To(Equal(""))
		Expect(recorder.Header().Get("Content-Type")).To(Equal(""))
		Expect(recorder.Header().Get("Content-Disposition")).To(Equal(""))
		Expect(recorder.Header().Get("Content-Length")).To(Equal(""))
	})

	It("Downloads files from a multiple node", func() {
		// Create a temp directory to store all the files needed for the test
		var tempDir, err = ioutil.TempDir("/tmp", "test")
		Expect(err).NotTo(HaveOccurred())

		// Create dummy files and add them to a tar archive
		var tarFileNodeOne = createTarArchive(tempDir, files, []byte(loremLipsum))
		var tarFileNodeTwo = createTarArchive(tempDir, otherFiles, []byte(loremLipsum))
		defer os.RemoveAll(tempDir)

		tarFileReaderOne, err := os.Open(tarFileNodeOne.Name())
		Expect(err).NotTo(HaveOccurred())
		tarFileReaderTwo, err := os.Open(tarFileNodeTwo.Name())
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureMultipleNodes, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "nodeOne").Return("entryNs", "entryPodOne", nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "nodeTwo").Return("entryNs", "entryPodTwo", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", pointNodeOne).Return(tarFileReaderOne, nil, nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", pointNodeTwo).Return(tarFileReaderTwo, nil, nil)
		var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

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
		validateArchive(archive, allFiles, []byte(loremLipsum))
	})

	DescribeTable("Failure to get packet capture",
		func(expectedStatus int, expectedError error) {
			// Bootstrap the download
			var mockCache = &cache.MockClientCache{}
			var mockK8sCommands = &capture.MockK8sCommands{}
			var mockFileRetrieval = &capture.MockFileCommands{}
			mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(nil, expectedError)
			var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

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

	DescribeTable("PacketCapture has no files attached",
		func(packetCapture *v3.PacketCapture) {
			// Bootstrap the download
			var mockCache = &cache.MockClientCache{}
			var mockK8sCommands = &capture.MockK8sCommands{}
			var mockFileRetrieval = &capture.MockFileCommands{}
			mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCapture, nil)
			var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

			// Bootstrap the http recorder
			recorder := httptest.NewRecorder()
			handler := http.HandlerFunc(download.Download)
			handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusNoContent))
		},
		Entry("Empty status", packetCaptureEmptyStatus),
		Entry("Missing status", packetCaptureNoStatus),
		Entry("No files generated for packet capture", packetCaptureNoFiles),
	)

	It("TarError returns an error via io.Reader", func() {
		var errorWriter bytes.Buffer
		var _, err = errorWriter.WriteString("any error")
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", point).Return(nil, &errorWriter, nil)
		var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Download)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal("any error"))
	})

	It("Ignore tar output removing leading /' from member names", func() {
		// Create a temp directory to store all the files needed for the test
		var tempDir, err = ioutil.TempDir("/tmp", "test")
		Expect(err).NotTo(HaveOccurred())

		// Create dummy files and add them to a tar archive
		var tarFile = createTarArchive(tempDir, files, []byte(loremLipsum))
		defer os.RemoveAll(tempDir)

		tarFileReader, err := os.Open(tarFile.Name())
		Expect(err).NotTo(HaveOccurred())

		var errorWriter bytes.Buffer
		_, err = errorWriter.WriteString("tar: removing leading '/' from member names")
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockk8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockk8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockk8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", point).Return(tarFileReader, &errorWriter, nil)
		var download = handlers.NewFiles(mockCache, mockk8sCommands, mockFileRetrieval)

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
		validateArchive(archive, files, []byte(loremLipsum))
	})

	It("Ignore tar output No such file or directory", func() {
		// Create a temp directory to store all the files needed for the test
		var tempDir, err = ioutil.TempDir("/tmp", "test")
		Expect(err).NotTo(HaveOccurred())

		// Create dummy files and add them to a tar archive
		var tarFile = createTarArchive(tempDir, noFiles, []byte(loremLipsum))
		defer os.RemoveAll(tempDir)

		tarFileReader, err := os.Open(tarFile.Name())
		Expect(err).NotTo(HaveOccurred())

		var errorWriter bytes.Buffer
		_, err = errorWriter.WriteString(
			"tar: /var/log/calico/pcap/tigera-manager/test-delete: No such file or directory" +
				"\ntar: error exit delayed from previous errors")
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("OpenTarReader", "cluster", point).Return(tarFileReader, &errorWriter, nil)
		var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Download)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusNoContent))
		Expect(recorder.Body.String()).To(Equal(""))
		Expect(recorder.Header().Get("Content-Type")).To(Equal(""))
		Expect(recorder.Header().Get("Content-Disposition")).To(Equal(""))
		Expect(recorder.Header().Get("Content-Length")).To(Equal(""))
	})

	It("Fails to locate an entry pod", func() {
		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", fmt.Errorf("any error"))
		var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Download)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal("any error"))
	})
})

func pcapHeader() []byte {
	const magicMicroseconds = 0xA1B2C3D4
	const versionMajor = 2
	const versionMinor = 4
	const snapshotLength = 1024
	const linkTypeEthernet = 1

	var pcapHeader = make([]byte, 24)
	binary.LittleEndian.PutUint32(pcapHeader[0:4], magicMicroseconds)
	binary.LittleEndian.PutUint16(pcapHeader[4:6], versionMajor)
	binary.LittleEndian.PutUint16(pcapHeader[6:8], versionMinor)
	binary.LittleEndian.PutUint32(pcapHeader[16:20], snapshotLength)
	binary.LittleEndian.PutUint32(pcapHeader[20:24], uint32(linkTypeEthernet))

	return pcapHeader
}

var _ = Describe("FilesDelete", func() {
	var req *http.Request

	BeforeEach(func() {
		// Create a new request
		var err error
		req, err = http.NewRequest("DELETE", "/files/ns/name", nil)
		Expect(err).NotTo(HaveOccurred())

		// Setup the variables on the context to be used for authN/authZ
		req = req.WithContext(middleware.WithClusterID(req.Context(), "cluster"))
		req = req.WithContext(middleware.WithNamespace(req.Context(), "ns"))
		req = req.WithContext(middleware.WithCaptureName(req.Context(), "name"))
	})

	It("Deletes files from a single node", func() {
		// Bootstrap the files
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(finishedPacketCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockK8sCommands.On("UpdatePacketCaptureStatusWithNoFiles", "cluster", "name", "ns",
			map[string]struct{}{"node": {}}).Return(nil)
		mockFileRetrieval.On("Delete", "cluster", point).Return(nil, nil)
		var files = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(files.Delete)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
	})

	It("Delete files from a multiple node", func() {
		// Bootstrap the download
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(finishedPacketCaptureMultipleNodes, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "nodeOne").Return("entryNs", "entryPodOne", nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "nodeTwo").Return("entryNs", "entryPodTwo", nil)
		mockK8sCommands.On("UpdatePacketCaptureStatusWithNoFiles", "cluster", "name", "ns",
			map[string]struct{}{"nodeOne": {}, "nodeTwo": {}}).Return(nil)
		mockFileRetrieval.On("Delete", "cluster", pointNodeOne).Return(nil, nil)
		mockFileRetrieval.On("Delete", "cluster", pointNodeTwo).Return(nil, nil)
		var download = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(download.Delete)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
	})

	DescribeTable("Failure to get packet capture",
		func(expectedStatus int, expectedError error) {
			// Bootstrap the files
			var mockCache = &cache.MockClientCache{}
			var mockK8sCommands = &capture.MockK8sCommands{}
			var mockFileRetrieval = &capture.MockFileCommands{}
			mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(nil, expectedError)
			var files = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

			// Bootstrap the http recorder
			recorder := httptest.NewRecorder()
			handler := http.HandlerFunc(files.Delete)
			handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(expectedStatus))
			Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal(expectedError.Error()))
		},
		Entry("Missing resource", http.StatusNotFound, errors.ErrorResourceDoesNotExist{}),
		Entry("Failure to get resource", http.StatusInternalServerError, fmt.Errorf("any error")),
	)

	DescribeTable("Fail to delete files for packetCapture with non-finished states",
		func(packetCapture *v3.PacketCapture, expectedStatus int, expectedErrMsg string) {
			// Bootstrap the files
			var mockCache = &cache.MockClientCache{}
			var mockK8sCommands = &capture.MockK8sCommands{}
			var mockFileRetrieval = &capture.MockFileCommands{}
			mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(packetCapture, nil)
			var files = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

			// Bootstrap the http recorder
			recorder := httptest.NewRecorder()
			handler := http.HandlerFunc(files.Delete)
			handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(expectedStatus))
			Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal(expectedErrMsg))
		},
		Entry("All nodes in different state", differentStatesPacketCaptureMultipleNodes, http.StatusForbidden, "capture state is not Finished"),
		Entry("Missing finished state", packetCaptureMultipleNodes, http.StatusForbidden, "capture state cannot be determined"),
		Entry("One finished state", oneFinishedPacketCaptureMultipleNodes, http.StatusForbidden, "capture state is not Finished"),
	)

	It("Delete returns an error via io.Reader", func() {
		var errorWriter bytes.Buffer
		var _, err = errorWriter.WriteString("any error")
		Expect(err).NotTo(HaveOccurred())

		// Bootstrap the files
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(finishedPacketCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockFileRetrieval.On("Delete", "cluster", point).Return(&errorWriter, nil)
		var files = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(files.Delete)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal("any error"))
	})

	It("Fails to locate an entry pod", func() {
		// Bootstrap the files
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(finishedPacketCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", fmt.Errorf("any error"))
		var files = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(files.Delete)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal("any error"))
	})

	It("Fails to update status", func() {
		// Bootstrap the files
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(finishedPacketCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockK8sCommands.On("UpdatePacketCaptureStatusWithNoFiles", "cluster", "name", "ns",
			map[string]struct{}{"node": {}}).Return(cerrors.ErrorResourceUpdateConflict{Identifier: "any"})
		mockFileRetrieval.On("Delete", "cluster", point).Return(nil, nil)
		var files = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(files.Delete)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(strings.Trim(recorder.Body.String(), "\n")).To(Equal("update conflict: any"))
		mockK8sCommands.AssertNumberOfCalls(GinkgoT(), "UpdatePacketCaptureStatusWithNoFiles", 4)
	})

	It("Retries to update status", func() {
		// Bootstrap the files
		var mockCache = &cache.MockClientCache{}
		var mockK8sCommands = &capture.MockK8sCommands{}
		var mockFileRetrieval = &capture.MockFileCommands{}
		mockK8sCommands.On("GetPacketCapture", "cluster", "name", "ns").Return(finishedPacketCaptureOneNode, nil)
		mockK8sCommands.On("GetEntryPod", "cluster", "node").Return("entryNs", "entryPod", nil)
		mockK8sCommands.On("UpdatePacketCaptureStatusWithNoFiles", "cluster", "name", "ns",
			map[string]struct{}{"node": {}}).Return(cerrors.ErrorResourceUpdateConflict{Identifier: "any"}).Twice()
		mockK8sCommands.On("UpdatePacketCaptureStatusWithNoFiles", "cluster", "name", "ns",
			map[string]struct{}{"node": {}}).Return(nil)
		mockFileRetrieval.On("Delete", "cluster", point).Return(nil, nil)
		var files = handlers.NewFiles(mockCache, mockK8sCommands, mockFileRetrieval)

		// Bootstrap the http recorder
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(files.Delete)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		mockK8sCommands.AssertNumberOfCalls(GinkgoT(), "UpdatePacketCaptureStatusWithNoFiles", 3)
	})
})

func validateArchive(archive *os.File, files []string, expectedData []byte) {
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

		Expect(content.String()).To(Equal(string(expectedData)))
		file.Close()
	}
}

func createTarArchive(dir string, files []string, data []byte) *os.File {
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
		_, err = file.Write(data)
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
