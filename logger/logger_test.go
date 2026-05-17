// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package logger

import (
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
