// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"fileganizer/logger"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"log/slog"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

func printVersion(version string) string {
	output := fmt.Sprintf("%-15s: %s\n", "Version", version)

	var lastCommit time.Time
	revision := "unknown"
	dirtyBuild := true

	info, ok := debug.ReadBuildInfo()
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
		case "vcs.time":
			lastCommit, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			dirtyBuild = kv.Value == "true"
		}
	}

	output += fmt.Sprintf("%-15s: %s\n", "Revision", revision)
	output += fmt.Sprintf("%-15s: %v\n", "Dirty Build", dirtyBuild)
	output += fmt.Sprintf("%-15s: %s\n", "Last Commit", lastCommit)
	output += fmt.Sprintf("%-15s: %s\n", "Go Version", info.GoVersion)
	return output
}

type CLIFlags struct {
	ConfigFile string
	InputFile  string
	TextOutput bool
	NoDryRun   bool
}

func parseFlags(version string) CLIFlags {
	fs := pflag.NewFlagSet("fileganizer", pflag.ContinueOnError)

	configFile := fs.StringP("config", "c", "", "Configuration file")
	inputFile := fs.StringP("file", "f", "", "File to scan")
	textOutput := fs.BoolP("text-output", "t", false, "Show extracted text")
	noDryRun := fs.BoolP("run", "r", false, "No Dry run with output of the command. Really run it !")
	showVersion := fs.BoolP("version", "V", false, "Show version info")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *showVersion {
		fmt.Print(printVersion(version))
		os.Exit(0)
	}

	if *configFile == "" {
		fmt.Fprintf(os.Stderr, "Error: --config/-c is required\n")
		os.Exit(1)
	}
	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: --file/-f is required\n")
		os.Exit(1)
	}

	return CLIFlags{
		ConfigFile: *configFile,
		InputFile:  *inputFile,
		TextOutput: *textOutput,
		NoDryRun:   *noDryRun,
	}
}

type FileDescription struct {
	Name     string
	Patterns []string
	Output   string
}

type Config struct {
	InputFile          string
	TextOutput         bool
	NoDryRun           bool
	GrokPatterns       map[string]string
	FileDescriptions   []FileDescription
	EnvVars            map[string]string
	CommonTemplate     string
	Months             map[string][]string
	ExtractTextCommand []string
}

func New(version string) (Config, error) {
	flags := parseFlags(version)
	var cfg Config
	cfg.InputFile = flags.InputFile
	cfg.TextOutput = flags.TextOutput
	cfg.NoDryRun = flags.NoDryRun

	err := cfg.readConfig(flags.ConfigFile)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func loggerConfig(k *koanf.Koanf) logger.LogOptions {
	logOpts := logger.LogOptions{
		Level:      "INFO",
		Filename:   "stdout",
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

func lookupConfigString(k *koanf.Koanf, camelKey string) (string, bool) {
	envKey := strings.ToLower(camelKey)
	if k.Exists(envKey) {
		return k.String(envKey), true
	}
	if k.Exists(camelKey) {
		return k.String(camelKey), true
	}
	return "", false
}

func lookupConfigStrings(k *koanf.Koanf, camelKey string) ([]string, bool) {
	envKey := strings.ToLower(camelKey)
	if k.Exists(envKey) {
		return k.Strings(envKey), true
	}
	if k.Exists(camelKey) {
		return k.Strings(camelKey), true
	}
	return nil, false
}

func lookupConfigMapKeys(k *koanf.Koanf, camelKey string) []string {
	envKey := strings.ToLower(camelKey)
	if k.Exists(envKey) {
		return k.MapKeys(envKey)
	}
	if k.Exists(camelKey) {
		return k.MapKeys(camelKey)
	}
	return nil
}

func (c *Config) readConfig(filename string) error {
	l := logger.Get()

	k := koanf.New(".")

	if err := k.Load(file.Provider(filename), yaml.Parser()); err != nil {
		return fmt.Errorf("failed to read configuration file %s: %w", filename, err)
	}

	if err := k.Load(env.Provider("FILEGANIZER_", ".", func(s string) string {
		s = strings.TrimPrefix(s, "FILEGANIZER_")
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "_", ".")
		return s
	}), nil); err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}

	logOpts := loggerConfig(k)
	logger.Reset(&logOpts)
	l = logger.Get()

	c.ExtractTextCommand, _ = lookupConfigStrings(k, "ExtractTextCommand")
	if len(c.ExtractTextCommand) == 0 {
		return fmt.Errorf("ExtractTextCommand is required in configuration file")
	}

	c.EnvVars = make(map[string]string)
	envList, _ := lookupConfigStrings(k, "env")
	for _, name := range envList {
		val, ok := os.LookupEnv(name)
		if !ok {
			l.Error("Environment variable (from configuration file) is not set", slog.String("name", name))
			return fmt.Errorf("environment variable (from configuration file) is not set: %s", name)
		}
		c.EnvVars[name] = val
	}

	if val, ok := lookupConfigString(k, "commonTemplate"); ok {
		c.CommonTemplate = val
	}

	c.Months = make(map[string][]string)
	for _, key := range lookupConfigMapKeys(k, "months") {
		prefix := "months." + key
		if vals, ok := lookupConfigStrings(k, prefix); ok {
			c.Months[key] = vals
		}
	}

	c.GrokPatterns = make(map[string]string)
	for _, key := range lookupConfigMapKeys(k, "grokPatterns") {
		prefix := "grokPatterns." + key
		if val, ok := lookupConfigString(k, prefix); ok {
			c.GrokPatterns[key] = val
		}
	}
	for key, months := range c.Months {
		c.GrokPatterns[key] = "(" + strings.Join(months, "|") + ")"
	}

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
		c.FileDescriptions = append(c.FileDescriptions, d)
	}

	l.Info("Configuration loaded",
		slog.String("file", filename),
		slog.Int("fileDescriptions", len(c.FileDescriptions)),
	)

	return nil
}
