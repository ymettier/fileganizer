// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"strings"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fileganizer/testutil"
)

func TestVersion(t *testing.T) {
	wantedVersion := "1.2.3"
	output := formatVersion(wantedVersion)
	s := strings.Split(output, "\n")
	assert.Equal(t, "Version        : "+wantedVersion, s[0], "Printing version")
}

func TestNewVersionFlag(t *testing.T) {
	setArgs(t, "fileganizer", "-V")

	cfg, err := New("1.2.3")
	assert.ErrorIs(t, err, ErrVersionRequested)
	assert.Empty(t, cfg.InputFile)
}

func TestNewMissingRequiredFlags(t *testing.T) {
	t.Run("missing config flag", func(t *testing.T) {
		setArgs(t, "fileganizer", "-f", "input.txt")
		_, err := New("1.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--config/-c")
	})

	t.Run("missing file flag", func(t *testing.T) {
		setArgs(t, "fileganizer", "-c", "config.yaml")
		_, err := New("1.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--file/-f")
	})
}

func writeConfig(t *testing.T, content string) {
	t.Helper()
	err := os.WriteFile("test_config.yaml", []byte(content), 0600)
	require.NoError(t, err)
}

func setArgs(t *testing.T, args ...string) {
	t.Helper()
	oldArgs := os.Args
	os.Args = args
	t.Cleanup(func() { os.Args = oldArgs })
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	os.Setenv(key, value)
	t.Cleanup(func() { os.Unsetenv(key) })
}

func TestNewWithDefaults(t *testing.T) {
	testutil.UseTempDir(t)
	configContent := `
ExtractTextCommand: ["cat", "FILENAME"]
commonTemplate: "prefix"
months:
  MONTHSENGLISH: ["January"]
grokPatterns:
  NUMBER: '[0-9]+'
fileDescriptions:
  test:
    patterns:
      - "%{NUMBER:id}"
    output: "{{ .grok.id }}"
`
	writeConfig(t, configContent)
	setArgs(t, "fileganizer", "-c", "test_config.yaml", "-f", "input.txt")

	cfg, err := New("1.0")
	require.NoError(t, err)

	assert.Equal(t, "input.txt", cfg.InputFile)
	assert.False(t, cfg.TextOutput)
	assert.False(t, cfg.NoDryRun)
	assert.Equal(t, []string{"cat", "FILENAME"}, cfg.ExtractTextCommand)
	assert.Equal(t, "prefix", cfg.CommonTemplate)
	assert.Contains(t, cfg.Months, "MONTHSENGLISH")
	assert.Contains(t, cfg.GrokPatterns, "NUMBER")
	assert.Equal(t, "[0-9]+", cfg.GrokPatterns["NUMBER"])
	assert.Len(t, cfg.FileDescriptions, 1)
	assert.Equal(t, "test", cfg.FileDescriptions[0].Name)
	assert.Equal(t, []string{"%{NUMBER:id}"}, cfg.FileDescriptions[0].Patterns)
	assert.Equal(t, "{{ .grok.id }}", cfg.FileDescriptions[0].Output)
}

func TestNewTextOutputAndRunFlags(t *testing.T) {
	testutil.UseTempDir(t)
	configContent := `
ExtractTextCommand: ["cat", "FILENAME"]
fileDescriptions:
  test:
    patterns:
      - "%{NUMBER:id}"
    output: "{{ .grok.id }}"
`
	writeConfig(t, configContent)
	setArgs(t, "fileganizer", "-c", "test_config.yaml", "-f", "input.txt", "-t", "-r")

	cfg, err := New("1.0")
	require.NoError(t, err)

	assert.True(t, cfg.TextOutput)
	assert.True(t, cfg.NoDryRun)
}

func TestNewMissingExtractTextCommand(t *testing.T) {
	testutil.UseTempDir(t)
	configContent := `
commonTemplate: "prefix"
`
	writeConfig(t, configContent)
	setArgs(t, "fileganizer", "-c", "test_config.yaml", "-f", "input.txt")

	_, err := New("1.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ExtractTextCommand is required")
}

func TestNewMissingConfigFile(t *testing.T) {
	testutil.UseTempDir(t)
	setArgs(t, "fileganizer", "-c", "nonexistent.yaml", "-f", "input.txt")

	_, err := New("1.0")
	assert.Error(t, err)
}

func TestNewWithEnvVars(t *testing.T) {
	testutil.UseTempDir(t)
	configContent := `
ExtractTextCommand: ["cat", "FILENAME"]
env:
  - MY_VAR
fileDescriptions:
  test:
    patterns:
      - "%{NUMBER:id}"
    output: "{{ .grok.id }}"
`
	writeConfig(t, configContent)
	setArgs(t, "fileganizer", "-c", "test_config.yaml", "-f", "input.txt")
	setEnv(t, "MY_VAR", "myvalue")

	cfg, err := New("1.0")
	require.NoError(t, err)
	assert.Equal(t, "myvalue", cfg.EnvVars["MY_VAR"])
}

func TestNewWithMissingEnvVar(t *testing.T) {
	testutil.UseTempDir(t)
	configContent := `
ExtractTextCommand: ["cat", "FILENAME"]
env:
  - MISSING_VAR
fileDescriptions:
  test:
    patterns:
      - "%{NUMBER:id}"
    output: "{{ .grok.id }}"
`
	writeConfig(t, configContent)
	setArgs(t, "fileganizer", "-c", "test_config.yaml", "-f", "input.txt")

	_, err := New("1.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MISSING_VAR")
}

func TestNewWithEnvVarOverrides(t *testing.T) {
	testutil.UseTempDir(t)
	configContent := `
ExtractTextCommand: ["cat", "FILENAME"]
commonTemplate: "original"
fileDescriptions:
  test:
    patterns:
      - "%{NUMBER:id}"
    output: "{{ .grok.id }}"
`
	writeConfig(t, configContent)
	setArgs(t, "fileganizer", "-c", "test_config.yaml", "-f", "input.txt")
	setEnv(t, "FILEGANIZER_COMMONTEMPLATE", "overridden")

	cfg, err := New("1.0")
	require.NoError(t, err)
	assert.Equal(t, "overridden", cfg.CommonTemplate)
}

func TestNewWithGrokPatternsAndMonths(t *testing.T) {
	testutil.UseTempDir(t)
	configContent := `
ExtractTextCommand: ["cat", "FILENAME"]
months:
  MONTHSFRENCH: ["Janvier", "Février", "Mars"]
grokPatterns:
  NUMBER: '[0-9]+'
  YEAR: "(?:\\d\\d){1,2}"
fileDescriptions:
  invoice:
    patterns:
      - "%{NUMBER:id}"
    output: "{{ .grok.id }}"
`
	writeConfig(t, configContent)
	setArgs(t, "fileganizer", "-c", "test_config.yaml", "-f", "input.txt")

	cfg, err := New("1.0")
	require.NoError(t, err)

	assert.Contains(t, cfg.GrokPatterns, "NUMBER")
	assert.Contains(t, cfg.GrokPatterns, "YEAR")
	assert.Contains(t, cfg.GrokPatterns, "MONTHSFRENCH")
	assert.Equal(t, "(Janvier|Février|Mars)", cfg.GrokPatterns["MONTHSFRENCH"])
}

func TestLookupConfigString(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Set("camelKey", "camelValue"))
	require.NoError(t, k.Set("lowercasekey", "lowerValue"))

	val, ok := lookupConfigString(k, "camelKey")
	assert.True(t, ok)
	assert.Equal(t, "camelValue", val)

	val, ok = lookupConfigString(k, "lowercasekey")
	assert.True(t, ok)
	assert.Equal(t, "lowerValue", val)

	_, ok = lookupConfigString(k, "nonexistent")
	assert.False(t, ok)
}

func TestLookupConfigStrings(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Set("items", []string{"a", "b"}))

	vals, ok := lookupConfigStrings(k, "items")
	assert.True(t, ok)
	assert.Equal(t, []string{"a", "b"}, vals)

	_, ok = lookupConfigStrings(k, "nonexistent")
	assert.False(t, ok)
}

func TestLookupConfigMapKeys(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Set("section.a", "1"))
	require.NoError(t, k.Set("section.b", "2"))
	require.NoError(t, k.Set("section.c", "3"))

	keys := lookupConfigMapKeys(k, "section")
	assert.ElementsMatch(t, []string{"a", "b", "c"}, keys)

	empty := lookupConfigMapKeys(k, "nonexistent")
	assert.Empty(t, empty)
}

func TestParseFlags_Help(t *testing.T) {
	_, err := parseFlags([]string{"fileganizer", "--help"})
	require.Error(t, err)
	assert.ErrorIs(t, err, pflag.ErrHelp)
}

func TestParseFlags_InvalidFlag(t *testing.T) {
	_, err := parseFlags([]string{"fileganizer", "--bogus"})
	require.Error(t, err)
}

func TestLoggerConfigDefaults(t *testing.T) {
	k := koanf.New(".")
	opts := loggerConfig(k)

	assert.Equal(t, "INFO", opts.Level)
	assert.Equal(t, "stderr", opts.Filename)
	assert.Equal(t, 5, opts.MaxSize)
	assert.Equal(t, 10, opts.MaxBackups)
	assert.Equal(t, 14, opts.MaxAge)
	assert.True(t, opts.Compress)
	assert.False(t, opts.JSON)
}

func TestLoggerConfigOverrides(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Set("logging.level", "DEBUG"))
	require.NoError(t, k.Set("logging.filename", "/var/log/test.log"))
	require.NoError(t, k.Set("logging.maxSize", 50))
	require.NoError(t, k.Set("logging.maxBackups", 20))
	require.NoError(t, k.Set("logging.maxAge", 30))
	require.NoError(t, k.Set("logging.compress", false))
	require.NoError(t, k.Set("logging.json", true))

	opts := loggerConfig(k)

	assert.Equal(t, "DEBUG", opts.Level)
	assert.Equal(t, "/var/log/test.log", opts.Filename)
	assert.Equal(t, 50, opts.MaxSize)
	assert.Equal(t, 20, opts.MaxBackups)
	assert.Equal(t, 30, opts.MaxAge)
	assert.False(t, opts.Compress)
	assert.True(t, opts.JSON)
}

func TestLoggerConfigPartialOverrides(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Set("logging.level", "ERROR"))
	require.NoError(t, k.Set("logging.filename", "stdout"))

	opts := loggerConfig(k)

	assert.Equal(t, "ERROR", opts.Level)
	assert.Equal(t, "stdout", opts.Filename)
	assert.Equal(t, 5, opts.MaxSize)
	assert.Equal(t, 10, opts.MaxBackups)
	assert.Equal(t, 14, opts.MaxAge)
	assert.True(t, opts.Compress)
	assert.False(t, opts.JSON)
}

func TestParseGrokPatterns_MonthKeyCollision(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Set("grokPatterns.COLLIDE", "[0-9]+"))
	require.NoError(t, k.Set("months.COLLIDE", []string{"jan", "feb"}))

	c := &Config{}
	c.parseMonths(k)

	err := c.parseGrokPatterns(k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts with existing grok pattern")
}

func TestParseGrokPatterns_MonthKeyCollisionReversed(t *testing.T) {
	k := koanf.New(".")
	require.NoError(t, k.Set("months.COLLIDE", []string{"jan", "feb"}))
	require.NoError(t, k.Set("grokPatterns.COLLIDE", ".*"))

	c := &Config{}
	c.parseMonths(k)

	err := c.parseGrokPatterns(k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts with existing grok pattern")
}
