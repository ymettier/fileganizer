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

func TestGetOtherTxtLogFile(t *testing.T) {
	filename := "txt_test.log"
	os.Setenv("LOG_TXT_FILENAME", filename)
	defer os.Unsetenv("LOG_TXT_FILENAME")

	// Remove the logs file before the test
	if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
		os.Remove(filename)
	}

	l := newLogger()
	l.Warn("Message")
	if !assert.FileExists(t, filename) {
		return
	}

	defer os.Remove(filename)

	txtFile, err := os.Open(filename)
	if !assert.NoError(t, err) {
		return
	}

	defer txtFile.Close()
	byteValue, _ := io.ReadAll(txtFile)

	// result :
	// 2024-01-13T00:07:58.928+0100	WARN	Message
	if !assert.Contains(t, string(byteValue), "WARN") {
		return
	}
	if !assert.Contains(t, string(byteValue), "Message") {
		return
	}
}

func TestGetOtherJsonLogFile(t *testing.T) {
	filename := "json_test.log"
	os.Setenv("LOG_JSON_FILENAME", filename)
	defer os.Unsetenv("LOG_JSON_FILENAME")

	// Remove the logs file before the test
	if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
		os.Remove(filename)
	}

	l := newLogger()
	l.Warn("Message")
	if !assert.FileExists(t, filename) {
		return
	}

	defer os.Remove(filename)

	jsonFile, err := os.Open(filename)
	if !assert.NoError(t, err) {
		return
	}

	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)

	var result map[string]any
	if err = json.Unmarshal(byteValue, &result); !assert.NoError(t, err) {
		return
	}
	// result :
	// {"level":"warn","timestamp":"2024-01-13T00:07:58.928+0100","msg":"Message","git_revision":"","go_version":"go1.21.5"}
	if !assert.Contains(t, result, "msg") {
		return
	}
	assert.Equal(t, result["msg"], "Message")
}
