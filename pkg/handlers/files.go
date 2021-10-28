// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handlers

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/tigera/packetcapture-api/pkg/cache"
	"github.com/tigera/packetcapture-api/pkg/capture"

	"github.com/projectcalico/libcalico-go/lib/errors"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/packetcapture-api/pkg/middleware"
)

const ignoringTarError = "tar: removing leading '/' from member names"

// Files defines the logic and http handler needed to retrieve/delete the files generated by a packet capture.
// For each node that contains packet capture files, it will launch a remote pod/exec command and bundle
// the files as a zip archive.
// For each node that contains packet capture files, it will launch a remote pod/exec command to delete all files.
type Files struct {
	capture.Locator
	capture.FileCommands
	cache.ClientCache
	ignoreTarDeleteRegex *regexp.Regexp
}

// NewFiles creates a new Files structure
func NewFiles(cache cache.ClientCache, locator capture.Locator, retrieval capture.FileCommands) *Files {
	ignoreTarDeleteRegex, _ := regexp.Compile("tar: (.)+: No such file or directory")

	return &Files{
		Locator:              locator,
		FileCommands:         retrieval,
		ClientCache:          cache,
		ignoreTarDeleteRegex: ignoreTarDeleteRegex,
	}
}

// Download is a http handler that returns the files generated by a packet capture as a zip archive
func (f *Files) Download(w http.ResponseWriter, r *http.Request) {
	log.Infof("Received the following request %s", r.RequestURI)

	if r.Method != http.MethodGet {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
	}

	var namespace = middleware.NamespaceFromContext(r.Context())
	var captureName = middleware.CaptureNameFromContext(r.Context())
	var clusterID = middleware.ClusterIDFromContext(r.Context())

	packetCapture, err := f.Locator.GetPacketCapture(clusterID, captureName, namespace)
	if err != nil {
		log.WithError(err).Errorf("Failed to get packet capture %s/%s", namespace, captureName)
		switch err.(type) {
		case errors.ErrorResourceDoesNotExist:
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if f.hasNoFiles(packetCapture) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var zipWriter = zip.NewWriter(w)
	var totalContentLength uint64 = 0
	var filesRead = 0
	for _, file := range packetCapture.Status.Files {
		log.Debugf("Copying files %v", file.FileNames)
		ns, pod, err := f.Locator.GetEntryPod(clusterID, file.Node)
		if err != nil {
			log.WithError(err).Errorf("Failed locate entry pods for %s/%s", namespace, captureName)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var entryPoint = capture.EntryPoint{PodNamespace: ns, PodName: pod, CaptureDirectory: file.Directory, CaptureName: captureName, CaptureNamespace: namespace}
		log.Debugf("Entry pods is %v", entryPoint)

		reader, errorReader, err := f.FileCommands.OpenTarReader(clusterID, entryPoint)
		if err != nil {
			log.WithError(err).Errorf("Failed create a remote command to retrieve the files from %v", entryPoint)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Start reading and compress files in a zip archive
		if reader != nil {
			var tarReader = tar.NewReader(reader)
			for {
				header, err := tarReader.Next()
				if err != nil {
					if err != io.EOF {
						log.WithError(err).Errorf("Failed to read stream from %s", file.Node)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					break
				}

				if header.FileInfo().IsDir() {
					continue
				}

				filesRead++
				zipHeader, err := zip.FileInfoHeader(header.FileInfo())
				if err != nil {
					log.WithError(err).Errorf("Failed write tar header to archive file %s from %s", header.FileInfo().Name(), file.Node)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				writer, err := zipWriter.CreateHeader(zipHeader)
				if err != nil {
					log.WithError(err).Errorf("Failed write tar header to archive file %s from %s", header.FileInfo().Name(), file.Node)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				var written int64
				written, err = io.Copy(writer, tarReader)
				if err != nil {
					log.WithError(err).Errorf("Failed add to archive file %s from %s", header.FileInfo().Name(), file.Node)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				zipWriter.Flush()
				log.Debugf("Compressed size is %v, while file size is %v", zipHeader.CompressedSize64, header.FileInfo().Size())
				log.Debugf("Written data is %v for %s", written, header.FileInfo().Name())
				totalContentLength = totalContentLength + zipHeader.CompressedSize64
			}
		}

		// Read the error from the stream
		err = readErrorFromStream(errorReader)
		if err != nil && !strings.Contains(err.Error(), ignoringTarError) && !f.ignoreTarDeleteRegex.MatchString(err.Error()) {
			log.WithError(err).Error("Remote command failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if filesRead == 0 {
		// Return 204 No Content
		w.WriteHeader(http.StatusNoContent)
		return
	} else {
		// Write headers for the request
		cd := mime.FormatMediaType("attachment", map[string]string{"filename": middleware.ZipFiles})
		w.Header().Set("Content-Disposition", cd)
		w.Header().Set("Content-Length", fmt.Sprint(totalContentLength))

		// Close the zip writer
		zipWriter.Flush()
		zipWriter.Close()
	}
}

func (f *Files) hasNoFiles(packetCapture *v3.PacketCapture) bool {
	if len(packetCapture.Status.Files) == 0 {
		return true
	}

	var reportedFiles = 0
	for _, filesPerNode := range packetCapture.Status.Files {
		reportedFiles = reportedFiles + len(filesPerNode.FileNames)
	}

	return reportedFiles == 0
}

func readErrorFromStream(errorReader io.Reader) error {
	if errorReader == nil {
		return nil
	}
	var errorBuffer bytes.Buffer
	var _, err = io.Copy(&errorBuffer, errorReader)
	if err != nil {
		return err
	}

	if len(errorBuffer.String()) != 0 {
		return fmt.Errorf("%s", errorBuffer.String())
	}
	return nil
}

// Delete is a http handler that deletes the files generated by a packet capture
func (f *Files) Delete(w http.ResponseWriter, r *http.Request) {
	log.Infof("Received the following request %s", r.RequestURI)

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
	}

	var namespace = middleware.NamespaceFromContext(r.Context())
	var captureName = middleware.CaptureNameFromContext(r.Context())
	var clusterID = middleware.ClusterIDFromContext(r.Context())

	packetCapture, err := f.Locator.GetPacketCapture(clusterID, captureName, namespace)
	if err != nil {
		log.WithError(err).Errorf("Failed to get packet capture %s/%s", namespace, captureName)
		switch err.(type) {
		case errors.ErrorResourceDoesNotExist:
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	for _, file := range packetCapture.Status.Files {
		if file.State == nil {
			var err = fmt.Errorf("capture state cannot be determined")
			log.WithError(err).Errorf("Failed delete files for %s/%s", namespace, captureName)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if file.State != nil && *file.State != v3.PacketCaptureStateFinished {
			var err = fmt.Errorf("capture state is not Finished")
			log.WithError(err).Errorf("Failed delete files for %s/%s", namespace, captureName)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	for _, file := range packetCapture.Status.Files {
		log.Debugf("Delete files %v", file)
		ns, pod, err := f.Locator.GetEntryPod(clusterID, file.Node)
		if err != nil {
			log.WithError(err).Errorf("Failed locate entry pods for %s/%s", namespace, captureName)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var entryPoint = capture.EntryPoint{PodNamespace: ns, PodName: pod, CaptureDirectory: file.Directory, CaptureName: captureName, CaptureNamespace: namespace}
		log.Debugf("Entry pods is %v", entryPoint)

		errorReader, err := f.FileCommands.Delete(clusterID, entryPoint)
		if err != nil {
			log.WithError(err).Errorf("Failed create a remote command to retrieve the files from %v", entryPoint)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Read the error from the stream
		err = readErrorFromStream(errorReader)
		if err != nil {
			log.WithError(err).Error("Remote command failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
