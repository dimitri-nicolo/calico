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

	BinaryFileMIME = "application/octet-stream"

	NoSuchFileDirectorErrMsg = "no such file or directory"

	ModelFileExtension = ".model"
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
			return "", err
		}
	}

	filePath := filepath.Join(directory, filename)
	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// write this byte array to the file
	_, err = f.Write(fileBytes)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func (s *FileModelStorageHandler) Load(r *http.Request) (string, *api_error.APIError) {
	modelPath := filepath.Join(s.FileStoragePath, r.URL.Path+ModelFileExtension)

	dat, err := os.ReadFile(modelPath)
	if err != nil {
		errStatusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), NoSuchFileDirectorErrMsg) {
			errStatusCode = http.StatusNotFound
		}
		return "", &api_error.APIError{
			StatusCode: errStatusCode,
			Err:        err,
		}
	}

	base64ModelStr := base64.StdEncoding.EncodeToString(dat)

	return base64ModelStr, nil
}
