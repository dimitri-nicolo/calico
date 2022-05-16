package storage

import (
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
)

const (
	ModelFileDataKey = "model"
	AcceptHeaderKey  = "Accept"

	NoSuchFileDirectoryErrMsg = "no such file or directory"
	ModelFileExtension        = ".model"

	fileSizeUnknown = -1
	emptyContent    = ""
)

// FileModelStorageHandler serves the storage and retrieval of models as a
// File object
type FileModelStorageHandler struct {
	FileStoragePath string
}

func (s *FileModelStorageHandler) Save(r *http.Request) error {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	base64ModelBytes, err := base64.StdEncoding.DecodeString(string(bytes))
	if err != nil {
		return err
	}

	modelPath := r.URL.Path

	filePath, err := s.createOnPath(modelPath, base64ModelBytes)
	if err != nil {
		return err
	}

	log.Infof("Saved file: %s", filePath)
	return nil
}

func (s *FileModelStorageHandler) createOnPath(path string, fileBytes []byte) (string, error) {
	pathSlice := strings.Split(path, "/")

	filename := pathSlice[len(pathSlice)-1] + ModelFileExtension
	pathSlice = pathSlice[:len(pathSlice)-1]
	directory := filepath.Join(s.FileStoragePath, strings.Join(pathSlice, "/"))

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.MkdirAll(directory, os.ModePerm)

		if err != nil {
			return emptyContent, err
		}
	}

	filePath := filepath.Join(directory, filename)
	f, err := os.Create(filePath)
	if err != nil {
		return emptyContent, err
	}
	defer f.Close()

	// write this byte array to the file
	_, err = f.Write(fileBytes)
	if err != nil {
		return emptyContent, err
	}

	return filePath, nil
}

func (s *FileModelStorageHandler) Load(r *http.Request) (int64, string, *api_error.APIError) {

	modelPath := filepath.Join(s.FileStoragePath, r.URL.Path+ModelFileExtension)

	dat, err := os.ReadFile(modelPath)
	if err != nil {
		errStatusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), NoSuchFileDirectoryErrMsg) {
			errStatusCode = http.StatusNotFound
		}
		return fileSizeUnknown, emptyContent, &api_error.APIError{
			StatusCode: errStatusCode,
			Err:        err,
		}
	}

	base64ModelStr := base64.StdEncoding.EncodeToString(dat)

	size, apiErr := s.Stat(r)
	if apiErr != nil {
		log.Warnf("Unable to get file info for: %s", modelPath)
		return fileSizeUnknown, base64ModelStr, nil
	}

	return size, base64ModelStr, nil
}

func (s *FileModelStorageHandler) Stat(r *http.Request) (int64, *api_error.APIError) {
	modelPath := s.getModelFilePath(r)

	fi, err := os.Stat(modelPath)
	if err != nil {
		errStatusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), NoSuchFileDirectoryErrMsg) {
			errStatusCode = http.StatusNotFound
		}

		return fileSizeUnknown, &api_error.APIError{
			StatusCode: errStatusCode,
			Err:        err,
		}
	}

	size := fi.Size()

	return size, nil
}

func (s *FileModelStorageHandler) getModelFilePath(r *http.Request) string {
	return filepath.Join(s.FileStoragePath, r.URL.Path+ModelFileExtension)
}
