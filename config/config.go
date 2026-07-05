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

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
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
	var cfg Config

	f := flag.NewFlagSet("fileganizer", flag.ContinueOnError)
	configFile := f.StringP("config", "c", "", "Configuration file")
	inputFile := f.StringP("file", "f", "", "File to scan")
	textOutput := f.BoolP("text-output", "t", false, "Show extracted text")
	noDryRun := f.BoolP("run", "r", false, "No Dry run with output of the command. Really run it !")
	showVersion := f.BoolP("version", "V", false, "Show version info")

	f.Parse(os.Args[1:])

	if *showVersion {
		fmt.Print(printVersion(version))
		os.Exit(0)
	}

	if *configFile == "" {
		return cfg, fmt.Errorf("--config/-c is required")
	}
	if *inputFile == "" {
		return cfg, fmt.Errorf("--file/-f is required")
	}

	cfg.InputFile = *inputFile
	cfg.TextOutput = *textOutput
	cfg.NoDryRun = *noDryRun

	err := cfg.readConfig(*configFile)
	if err != nil {
		return cfg, err
	}

	if len(cfg.ExtractTextCommand) == 0 {
		return cfg, fmt.Errorf("ExtractTextCommand is required in configuration file")
	}

	return cfg, nil
}

func (c *Config) readConfig(filename string) error {
	l := logger.Get()

	k := koanf.New(".")

	err := k.Load(file.Provider(filename), yaml.Parser())
	if err != nil {
		l.Error("Could not read or parse configuration file", "file", filename, "error", err)
		return err
	}

	// parse ExtractTextCommand
	c.ExtractTextCommand = k.Strings("ExtractTextCommand")

	// parse env
	c.EnvVars = make(map[string]string)
	if envList := k.Strings("env"); len(envList) > 0 {
		for _, e := range envList {
			val, ok := os.LookupEnv(e)
			if !ok {
				l.Error("Environment variable (from configuration file) is not set", "name", e)
				return fmt.Errorf("environment variable (from configuration file) is not set: %s", e)
			}
			c.EnvVars[e] = val
		}
	}

	// parse commonTemplate
	c.CommonTemplate = k.String("commonTemplate")

	// parse months
	if k.Exists("months") {
		var months map[string][]string
		if err := k.Unmarshal("months", &months); err != nil {
			l.Error("Could not parse months configuration", "error", err)
			return err
		}
		c.Months = months
	}
	if c.Months == nil {
		c.Months = make(map[string][]string)
	}

	// parse grokPatterns
	c.GrokPatterns = make(map[string]string)
	for key, val := range k.StringMap("grokPatterns") {
		c.GrokPatterns[key] = val
	}
	// append months patterns to GrokPatterns
	for key, months := range c.Months {
		c.GrokPatterns[key] = "(" + strings.Join(months, "|") + ")"
	}

	// parse fileDescriptions
	c.FileDescriptions = make([]FileDescription, 0)
	if fdRaw := k.Get("fileDescriptions"); fdRaw != nil {
		if fdMap, ok := fdRaw.(map[string]any); ok {
			for name, v := range fdMap {
				var d FileDescription
				d.Name = name
				if entry, ok := v.(map[string]any); ok {
					if patterns, ok := entry["patterns"].([]any); ok {
						d.Patterns = make([]string, 0, len(patterns))
						for _, p := range patterns {
							d.Patterns = append(d.Patterns, fmt.Sprintf("%v", p))
						}
					}
					if output, ok := entry["output"].(string); ok {
						d.Output = output
					}
				}
				c.FileDescriptions = append(c.FileDescriptions, d)
			}
		}
	}

	return nil
}
