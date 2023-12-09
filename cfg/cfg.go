// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cfg

import (
	"fileganizer/logger"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type VersionFlag bool

func printVersion(version string) (string, error) {
	output := fmt.Sprintf("%-15s: %s\n", "Version", version)

	// Get and print additionnal build info
	var lastCommit time.Time
	revision := "unknown"
	dirtyBuild := true

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return output, nil
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
	return output, nil
}

func (v VersionFlag) BeforeReset(version string) error {
	output, _ := printVersion(version)
	fmt.Printf("%s", output)
	os.Exit(0)
	return nil
}

type CLI struct {
	InputFile  string      `name:"file" short:"f" required:"" help:"File to scan"`
	ConfigFile string      `name:"config" short:"c" required:"" help:"Configuration file"`
	TextOutput bool        `name:"text-output" short:"t" default:false optional:"" help:"Show extracted text"`
	NoDryRun   bool        `name:"run" short:"r" default:false optional:"" help:"No Dry run with output of the command. Really run it !"`
	Version    VersionFlag `name:"version" short:"V"  help:"Show version info"`
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
	var cli CLI
	//l := logger.Get()
	kong.Parse(&cli, kong.Bind(version))
	var cfg Config
	cfg.InputFile = cli.InputFile
	cfg.TextOutput = cli.TextOutput
	cfg.NoDryRun = cli.NoDryRun

	err := cfg.readConfig(cli.ConfigFile)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (c *Config) readConfig(filename string) error {
	var data map[string]interface{}
	l := logger.Get()
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		l.Fatal("Could not read configuration file", zap.String("file", filename), zap.Error(err))
		return err
	}

	err = yaml.Unmarshal(yamlFile, &data)
	if err != nil {
		l.Fatal("Could not parse configuration file", zap.String("file", filename), zap.Error(err))
		return err
	}
	// Configuration file format
	//
	// ExtractTextCommand: ["pdftotext", "-nopgbrk", "-enc", "UTF-8", "FILENAME", "-"]
	//
	// env:
	//   - ENV_VAR_1
	//   - ENV_VAR_2
	//
	// commonTemplate: ""
	//
	// months:
	//   MONTHSFRENCH: ["janvier", "février", "mars", "avril", "mai", "juin", "juillet", "aout", "septembre", "octobre", "novembre", "décembre"]
	//
	// grokPatterns:
	//   YEAR: "(?:\d\d){1,2}"
	//   MONTHNUM2: "0[1-9]|1[0-2]"
	//   MONTHDAY: "(?:0[1-9])|(?:[12][0-9])|(?:3[01])|[1-9]"
	//   HOUR: "2[0123]|[01]?[0-9]"
	//   MINUTE: "[0-5][0-9]"
	//   SECOND: "(?:[0-5]?[0-9]|60)(?:[:.,][0-9]+)?"
	//   TIMEZONE: "Z%{HOUR}:%{MINUTE}"
	//   DATE: "%{YEAR:year}-%{MONTHNUM2:month}-%{MONTHDAY:day}"
	//   TIME: "%{HOUR:hour}:%{MINUTE:min}:%{SECOND:sec}"
	//
	// fileDescriptions:
	//   <type>:
	//     patterns:
	//       - "pattern1"
	//       - "pattern2"
	//     output: "output string with patterns"

	// parse ExtractTextCommand
	c.ExtractTextCommand = make([]string, 0)
	for _, e := range data["ExtractTextCommand"].([]interface{}) {
		c.ExtractTextCommand = append(c.ExtractTextCommand, e.(string))
	}

	// parse env
	c.EnvVars = make(map[string]string)
	for _, e := range data["env"].([]interface{}) {
		val, ok := os.LookupEnv(e.(string))
		if !ok {
			l.Fatal("Environment variable (from configuration file) is not set", zap.String("name", e.(string)))
			return err
		}
		c.EnvVars[e.(string)] = val
	}

	// parse commonTemplate
	c.CommonTemplate = data["commonTemplate"].(string)

	// parse months
	c.Months = make(map[string][]string)
	for k, v := range data["months"].(map[string]interface{}) {
		c.Months[k] = make([]string, 0, 12)
		for _, m := range v.([]interface{}) {
			c.Months[k] = append(c.Months[k], m.(string))
		}
	}

	// parse grokPatterns
	c.GrokPatterns = make(map[string]string)
	for k, v := range data["grokPatterns"].(map[string]interface{}) {
		c.GrokPatterns[k] = v.(string)
	}
	// append months patterns to GrokPatterns
	for k, months := range c.Months {
		c.GrokPatterns[k] = "(" + strings.Join(months, "|") + ")"
	}

	// parse fileDescriptions
	c.FileDescriptions = make([]FileDescription, 0)
	for k, v := range data["fileDescriptions"].(map[string]interface{}) {
		var d FileDescription
		d.Name = k
		d.Patterns = make([]string, 0)
		fd := v.(map[string]interface{})
		for _, p := range fd["patterns"].([]interface{}) {
			d.Patterns = append(d.Patterns, p.(string))
		}
		d.Output = fd["output"].(string)
		c.FileDescriptions = append(c.FileDescriptions, d)
	}
	return nil
}

func main() {
	cfg, _ := New("")
	fmt.Printf("%v\n", cfg)
}
