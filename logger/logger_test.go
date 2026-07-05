// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package logger

import (
	"context"
	"encoding/json"
	"fileganizer/testutil"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetGlobal() {
	mu.Lock()
	defer mu.Unlock()
	logger = nil
}

func TestGetOtherTxtLogFile(t *testing.T) {
	testutil.UseTempDir(t)
	filename := "txt_test.log"

	l := newLogger(&LogOptions{Filename: filename})
	l.Warn("Message")
	if !assert.FileExists(t, filename) {
		return
	}

	txtFile, err := os.Open(filename)
	if !assert.NoError(t, err) {
		return
	}

	defer txtFile.Close()
	byteValue, _ := io.ReadAll(txtFile)

	assert.Contains(t, string(byteValue), "level=WARN")
	assert.Contains(t, string(byteValue), "msg=Message")
}

func TestGetOtherJsonLogFile(t *testing.T) {
	testutil.UseTempDir(t)
	filename := "json_test.log"

	l := newLogger(&LogOptions{Filename: filename, JSON: true})
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

	// Find the test message line (second line, after the config log)
	lines := strings.Split(string(byteValue), "\n")
	var messageLine string
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry["msg"] == "Message" {
			messageLine = line
			break
		}
	}
	assert.NotEmpty(t, messageLine, "should find the test message line")

	var result map[string]any
	err = json.Unmarshal([]byte(messageLine), &result)
	assert.NoError(t, err)
	assert.Equal(t, "WARN", result["level"])
	assert.Equal(t, "Message", result["msg"])
}

func TestLogLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("LOG_LEVEL")

	l := newLogger(nil)
	assert.True(t, l.Enabled(context.Background(), -4)) // slog.LevelDebug is -4
}

func TestGetWriter_File(t *testing.T) {
	testutil.UseTempDir(t)
	filename := "lumberjack_test.log"

	w, ok := getWriter(&LogOptions{Filename: filename})
	require.NotNil(t, w)
	assert.True(t, ok)

	_, err := w.Write([]byte("test"))
	assert.NoError(t, err)
}

func TestGetWriter_FileDefaults(t *testing.T) {
	testutil.UseTempDir(t)
	filename := "lumberjack_defaults.log"

	w, ok := getWriter(&LogOptions{Filename: filename, MaxSize: 0, MaxBackups: 0, MaxAge: 0, Compress: false})
	assert.NotNil(t, w)
	assert.True(t, ok)
}

func TestGet(t *testing.T) {
	resetGlobal()

	l1 := Get()
	require.NotNil(t, l1)

	l2 := Get()
	assert.Same(t, l1, l2)
}

func TestReset(t *testing.T) {
	resetGlobal()

	l1 := Get()
	require.NotNil(t, l1)

	Reset(&LogOptions{Filename: "stderr", JSON: true}) //nolint:goconst
	l2 := Get()
	assert.NotSame(t, l1, l2)
}

func TestWithCtx_FromCtx(t *testing.T) {
	l := newLogger(nil)
	ctx := WithCtx(context.Background(), l)

	extracted := FromCtx(ctx)
	assert.Same(t, l, extracted)
}

func TestFromCtx_NoLogger(t *testing.T) {
	resetGlobal()
	l := FromCtx(context.Background())
	assert.NotNil(t, l) // returns the default logger
}

func TestNewLogger_InvalidLevelEnv(t *testing.T) {
	os.Setenv("LOG_LEVEL", "BOGUS")
	defer os.Unsetenv("LOG_LEVEL")

	l := newLogger(nil)
	assert.NotNil(t, l)

	// should have defaulted to INFO
	assert.True(t, l.Enabled(context.Background(), 0))   // slog.LevelInfo
	assert.False(t, l.Enabled(context.Background(), -8)) // slog.LevelDebug
}

func TestNewLogger_NilOptsNoEnv(t *testing.T) {
	l := newLogger(nil)
	assert.NotNil(t, l)
}

func TestNewLogger_LumberjackConfigLog(t *testing.T) {
	testutil.UseTempDir(t)
	filename := "lumberjack_cfg.log"

	l := newLogger(&LogOptions{
		Filename:   filename,
		Level:      "INFO",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     7,
		Compress:   true,
	})
	require.NotNil(t, l)
	assert.FileExists(t, filename)
}

func TestGetWriter_Stdout(t *testing.T) {
	w, ok := getWriter(&LogOptions{Filename: "stdout"})
	assert.NotNil(t, w)
	assert.False(t, ok)
}

func TestGetWriter_Stderr(t *testing.T) {
	w, ok := getWriter(&LogOptions{Filename: "stderr"})
	assert.NotNil(t, w)
	assert.False(t, ok)
}

func TestGetWriter_NilOpts(t *testing.T) {
	w, ok := getWriter(nil)
	assert.NotNil(t, w)
	assert.False(t, ok)
}

func TestGetWriter_EmptyFilename(t *testing.T) {
	w, ok := getWriter(&LogOptions{Filename: ""})
	assert.NotNil(t, w)
	assert.False(t, ok)
}

func TestGet_RaceCondition(t *testing.T) {
	resetGlobal()

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l := Get()
			assert.NotNil(t, l)
		}()
	}
	wg.Wait()
}
