// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package logger

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOtherLogFile(t *testing.T) {
	// Note : we cannot test more than one logger file
	// because Get() will run only once (thanks to sync.Once).
	// So we test only with the file specified in the env var LOG_FILENAME.
	filename := "fileganizer_test.log"
	os.Setenv("LOG_FILENAME", filename)

	// Remove the logs file before the test
	if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
		os.Remove(filename)
	}

	l := Get()
	l.Warn("Message")
	if !assert.FileExists(t, filename) {
		return
	}

	jsonFile, err := os.Open(filename)
	if !assert.NoError(t, err) {
		return
	}

	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	var result map[string]any
	json.Unmarshal([]byte(byteValue), &result)
	// result :
	// {"level":"warn","timestamp":"2024-01-13T00:07:58.928+0100","msg":"Message","git_revision":"","go_version":"go1.21.5"}
	if !assert.Contains(t, result, "msg") {
		return
	}
	assert.Equal(t, result["msg"], "Message")
	os.Remove(filename)
}
