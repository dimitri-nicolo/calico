package logger_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/projectcalico/calico/app-policy/logger"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

func TestProcess(t *testing.T) {
	// Create a new LogHandler with a buffer as the writer
	buffer := &bytes.Buffer{}
	handler := logger.New(buffer)

	// Test processing a *v1.WAFLog value
	testValue := &v1.WAFLog{
		Msg: "Hello, World!",
		// Fill in other fields as necessary
	}
	handler.Process(testValue)

	// Check that the value was logged at the ErrorLevel
	assert.Contains(t, buffer.String(), testValue.Msg)
	assert.Contains(t, buffer.String(), "error")
	assert.Contains(t, buffer.String(), "@timestamp")

	// Test processing a different *v1.WAFLog value
	testValue2 := &v1.WAFLog{
		Msg: "Goodbye, World!",
		// Fill in other fields as necessary
	}
	handler.Process(testValue2)

	// Check that the value was logged at the ErrorLevel
	assert.Contains(t, buffer.String(), testValue2.Msg)
	assert.Contains(t, buffer.String(), "error")
	assert.Contains(t, buffer.String(), "@timestamp")
}

func TestFileWriter(t *testing.T) {
	// Test writing to a valid file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.log")

	writer, err := logger.FileWriter(tempFile)
	if err != nil {
		t.Fatalf("Error creating file writer: %s", err)
		return
	}
	assert.NotNil(t, writer)

	// Write some data to the file
	data := []byte("Hello, World!")
	_, err = writer.Write(data)
	assert.NoError(t, err)

	// Read the data from the file
	file, err := os.Open(tempFile)
	assert.NoError(t, err)
	readData, err := io.ReadAll(file)
	assert.NoError(t, err)
	assert.Equal(t, data, readData)

	// Test writing to an invalid file
	_, err = logger.FileWriter("/invalid/path")
	assert.Error(t, err)
}
