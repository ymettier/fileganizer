// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"

	"fileganizer/logger"
)

var readBuildInfo = debug.ReadBuildInfo

func formatVersion(version string) string {
	output := fmt.Sprintf("%-15s: %s\n", "Version", version)

	var lastCommit time.Time
	var rawLastCommit string
	var parseVCSTimeErr error
	revision := "unknown"
	dirtyBuild := true

	info, ok := readBuildInfo()
	if !ok {
		return output
	}

	for _, kv := range info.Settings {
		if kv.Value == "" {
			continue
		}
		switch kv.Key {
		case "vcs.revision":
			revision = kv.Value
		case "vcs.time": //nolint:goconst
			rawLastCommit = kv.Value
			lastCommit, parseVCSTimeErr = time.Parse(time.RFC3339, kv.Value)
			if parseVCSTimeErr != nil {
				// formatVersion runs before the project logger is initialized,
				// so slog.Default() is the only available logger here.
				slog.Default().Warn("Failed to parse vcs.time", "value", rawLastCommit, "error", parseVCSTimeErr)
			}
		case "vcs.modified":
			dirtyBuild = kv.Value == "true"
		}
	}

	output += fmt.Sprintf("%-15s: %s\n", "Revision", revision)
	output += fmt.Sprintf("%-15s: %v\n", "Dirty Build", dirtyBuild)
	if parseVCSTimeErr != nil {
		output += fmt.Sprintf("%-15s: %s (raw)\n", "Last Commit", rawLastCommit)
	} else {
		output += fmt.Sprintf("%-15s: %s\n", "Last Commit", lastCommit)
	}
	output += fmt.Sprintf("%-15s: %s\n", "Go Version", info.GoVersion)
	return output
}

// ErrVersionRequested is returned by New when the user passes --version.
var ErrVersionRequested = errors.New("version requested")

// cliFlags holds the parsed command-line flag values.
type cliFlags struct {
	ConfigFile  string
	InputFile   string
	TextOutput  bool
	NoDryRun    bool
	ShowVersion bool
}

func parseFlags(args []string) (cliFlags, error) {
	fs := pflag.NewFlagSet("fileganizer", pflag.ContinueOnError)

	configFile := fs.StringP("config", "c", "", "Configuration file")
	inputFile := fs.StringP("file", "f", "", "File to scan")
	textOutput := fs.BoolP("text-output", "t", false, "Show extracted text")
	noDryRun := fs.BoolP("run", "r", false, "No Dry run with output of the command. Really run it !")
	showVersion := fs.BoolP("version", "V", false, "Show version info")

	if err := fs.Parse(args); err != nil {
		return cliFlags{}, fmt.Errorf("error parsing flags: %w", err)
	}

	if *showVersion {
		return cliFlags{ShowVersion: true}, nil
	}

	if *configFile == "" {
		return cliFlags{}, fmt.Errorf("--config/-c is required")
	}
	if *inputFile == "" {
		return cliFlags{}, fmt.Errorf("--file/-f is required")
	}

	return cliFlags{
		ConfigFile: *configFile,
		InputFile:  *inputFile,
		TextOutput: *textOutput,
		NoDryRun:   *noDryRun,
	}, nil
}

// FileDescription describes a document type to match, including the grok patterns
// to extract fields and the Go template to produce the output command.
type FileDescription struct {
	Name     string
	Patterns []string
	Output   string
}

// TextExtractorConfig defines how text is extracted for a specific MIME type.
// Supported types: "builtin" (internal extractor), "command" (runs an external command).
type TextExtractorConfig struct {
	Type    string
	Command []string
}

// Config holds all configuration values for the application, merging CLI flags,
// YAML config file, and environment variable overrides.
type Config struct {
	InputFile        string
	TextOutput       bool
	NoDryRun         bool
	GrokPatterns     map[string]string
	FileDescriptions []FileDescription
	EnvVars          map[string]string
	CommonTemplate   string
	Months           map[string][]string
	TextExtractors   map[string]TextExtractorConfig
}

// New parses CLI flags and the YAML configuration file, returning a fully
// populated Config. It returns ErrVersionRequested when --version is passed.
func New(version string, args []string) (Config, error) {
	flags, err := parseFlags(args)
	if err != nil {
		return Config{}, err
	}

	if flags.ShowVersion {
		fmt.Print(formatVersion(version))
		return Config{}, ErrVersionRequested
	}

	var cfg Config
	cfg.InputFile = flags.InputFile
	cfg.TextOutput = flags.TextOutput
	cfg.NoDryRun = flags.NoDryRun

	logOpts, err := cfg.readConfig(flags.ConfigFile)
	if err != nil {
		return cfg, err
	}

	logger.Reset(&logOpts)

	return cfg, nil
}

func loggerConfig(k *koanf.Koanf) logger.LogOptions {
	logOpts := logger.LogOptions{
		Level:      "INFO",
		Filename:   "stderr",
		MaxSize:    5,
		MaxBackups: 10,
		MaxAge:     14,
		Compress:   true,
		JSON:       false,
	}

	if v := k.String("logging.level"); v != "" {
		logOpts.Level = v
	}
	if v := k.String("logging.filename"); v != "" {
		logOpts.Filename = v
	}
	if k.Exists("logging.maxSize") {
		logOpts.MaxSize = k.Int("logging.maxSize")
	}
	if k.Exists("logging.maxBackups") {
		logOpts.MaxBackups = k.Int("logging.maxBackups")
	}
	if k.Exists("logging.maxAge") {
		logOpts.MaxAge = k.Int("logging.maxAge")
	}
	if k.Exists("logging.compress") {
		logOpts.Compress = k.Bool("logging.compress")
	}
	if k.Exists("logging.json") {
		logOpts.JSON = k.Bool("logging.json")
	}
	return logOpts
}

// findConfigKey returns the first matching key (lowercase or camelCase) that exists in k.
func findConfigKey(k *koanf.Koanf, camelKey string) (string, bool) {
	lowerKey := strings.ToLower(camelKey)
	if k.Exists(lowerKey) {
		return lowerKey, true
	}
	if k.Exists(camelKey) {
		return camelKey, true
	}
	return "", false
}

func lookupConfigString(k *koanf.Koanf, camelKey string) (string, bool) {
	key, ok := findConfigKey(k, camelKey)
	if !ok {
		return "", false
	}
	return k.String(key), true
}

func lookupConfigStrings(k *koanf.Koanf, camelKey string) ([]string, bool) {
	key, ok := findConfigKey(k, camelKey)
	if !ok {
		return nil, false
	}
	return k.Strings(key), true
}

func lookupConfigMapKeys(k *koanf.Koanf, camelKey string) []string {
	if key, ok := findConfigKey(k, camelKey); ok {
		return k.MapKeys(key)
	}
	return nil
}

func loadYAML(filename string) (*koanf.Koanf, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider(filename), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %w", filename, err)
	}

	// Build the key mapper from actual MIME types in the config so that
	// any MIME type (not just hardcoded ones) is restored correctly.
	mimeTypes := lookupConfigMapKeys(k, "textExtractor")
	mapper := fileganizerKeyMapper(mimeTypes)

	// env.Provider produces flat key=value pairs, so no parser is needed (nil is fine).
	loadConfigLayer(k, env.Provider("FILEGANIZER_", ".", mapper), "Failed to load environment variable overrides")

	return k, nil
}

// loadConfigLayer loads a koanf provider, logging a warning instead of failing
// when the provider errors, since env overrides are best-effort.
func loadConfigLayer(k *koanf.Koanf, loader koanf.Provider, msg string) {
	l := logger.Get()
	if err := k.Load(loader, nil); err != nil {
		l.Warn(msg, "err", err)
	}
}

// fileganizerKeyMapper returns a mapper function that converts a
// FILEGANIZER_-prefixed environment variable name to its corresponding koanf
// config key. It strips the prefix, lowercases, replaces underscores with dots,
// and restores MIME type slashes from the known types in the config.
func fileganizerKeyMapper(mimeTypes []string) func(string) string {
	// Build a lookup: lowercase env form -> koanf key.
	// e.g. "text/plain" -> envForm "text_plain" -> koanfKey "text/plain"
	replacements := make(map[string]string, len(mimeTypes))
	for _, mt := range mimeTypes {
		envForm := strings.ToLower(strings.ReplaceAll(mt, "/", "_"))
		replacements[envForm] = mt
	}

	return func(s string) string {
		s = strings.TrimPrefix(s, "FILEGANIZER_")
		s = strings.ToLower(s)

		// Try to find and replace any known MIME type part before
		// converting remaining underscores to dots.
		for envForm, koanfKey := range replacements {
			if idx := strings.Index(s, envForm); idx >= 0 {
				s = s[:idx] + koanfKey + s[idx+len(envForm):]
				break
			}
		}

		s = strings.ReplaceAll(s, "_", ".")
		return s
	}
}

func (c *Config) parseTextExtractor(k *koanf.Koanf) error {
	c.TextExtractors = make(map[string]TextExtractorConfig)

	mimeTypes := lookupConfigMapKeys(k, "textExtractor")
	if len(mimeTypes) == 0 {
		return fmt.Errorf("textExtractor is required in configuration file")
	}

	for _, mimeType := range mimeTypes {
		prefix := "textExtractor." + mimeType + "."
		te := TextExtractorConfig{}

		if v, ok := lookupConfigString(k, prefix+"type"); ok {
			te.Type = v
		}

		switch te.Type {
		case "builtin":
			// No additional config needed for builtin extractor
		case "command":
			cmd, ok := lookupConfigStrings(k, prefix+"command")
			if !ok || len(cmd) == 0 {
				return fmt.Errorf("textExtractor.%s.command is required when type is 'command'", mimeType)
			}
			te.Command = cmd
		case "":
			return fmt.Errorf("textExtractor.%s.type is required", mimeType)
		default:
			return fmt.Errorf("textExtractor.%s.type must be 'builtin' or 'command', got %q", mimeType, te.Type)
		}

		c.TextExtractors[mimeType] = te
	}

	return nil
}

func (c *Config) parseEnvVars(k *koanf.Koanf) error {
	c.EnvVars = make(map[string]string)
	envList, _ := lookupConfigStrings(k, "env")
	for _, name := range envList {
		val, ok := os.LookupEnv(name)
		if !ok {
			return fmt.Errorf("environment variable (from configuration file) is not set: %s", name)
		}
		c.EnvVars[name] = val
	}
	return nil
}

func (c *Config) parseMonths(k *koanf.Koanf) {
	c.Months = make(map[string][]string)
	for _, key := range lookupConfigMapKeys(k, "months") {
		prefix := "months." + key
		if vals, ok := lookupConfigStrings(k, prefix); ok {
			c.Months[key] = vals
		}
	}
}

func (c *Config) parseGrokPatterns(k *koanf.Koanf) error {
	c.GrokPatterns = make(map[string]string)
	for _, key := range lookupConfigMapKeys(k, "grokPatterns") {
		prefix := "grokPatterns." + key
		if val, ok := lookupConfigString(k, prefix); ok {
			c.GrokPatterns[key] = val
		}
	}
	for key, months := range c.Months {
		if _, exists := c.GrokPatterns[key]; exists {
			return fmt.Errorf("month key %q conflicts with existing grok pattern", key)
		}
		c.GrokPatterns[key] = "(" + strings.Join(months, "|") + ")"
	}
	return nil
}

func (c *Config) parseFileDescriptions(k *koanf.Koanf) {
	c.FileDescriptions = make([]FileDescription, 0)
	for _, id := range lookupConfigMapKeys(k, "fileDescriptions") {
		prefix := "fileDescriptions." + id + "."
		d := FileDescription{
			Name: id,
		}
		if patterns, ok := lookupConfigStrings(k, prefix+"patterns"); ok {
			d.Patterns = patterns
		}
		if output, ok := lookupConfigString(k, prefix+"output"); ok {
			d.Output = output
		}
		if d.Output == "" {
			d.Output = "{{.Filename}}"
		}
		c.FileDescriptions = append(c.FileDescriptions, d)
	}
}

func (c *Config) readConfig(filename string) (logger.LogOptions, error) {
	k, err := loadYAML(filename)
	if err != nil {
		return logger.LogOptions{}, err
	}

	logOpts := loggerConfig(k)

	if err := c.parseTextExtractor(k); err != nil {
		return logOpts, err
	}
	if err := c.parseEnvVars(k); err != nil {
		return logOpts, err
	}

	if val, ok := lookupConfigString(k, "commonTemplate"); ok {
		c.CommonTemplate = val
	}

	c.parseMonths(k)
	if err := c.parseGrokPatterns(k); err != nil {
		return logOpts, err
	}
	c.parseFileDescriptions(k)

	return logOpts, nil
}
