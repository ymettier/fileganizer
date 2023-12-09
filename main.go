// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fileganizer/cfg"
	"fileganizer/grok"
	"fileganizer/logger"
	"fileganizer/output"
	"fileganizer/textextract"
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "embed"

	"go.uber.org/zap"
)

var (
	Version string = strings.TrimSpace(version)
	//go:embed version.txt
	version string
)

func main() {
	l := logger.Get()

	cfg, err := cfg.New(Version)
	if err != nil {
		os.Exit(1)
	}

	txt, err := textextract.TextExtract(cfg.InputFile, cfg.ExtractTextCommand)
	if err != nil {
		os.Exit(1)
	}
	if cfg.TextOutput {
		fmt.Printf("%v\n", txt)
		os.Exit(0)
	}
	//	fmt.Printf("Config %v\n", cfg)
	//	fmt.Printf("Patterns %v\n", cfg.FileDescriptions)
	g := grok.New(cfg.GrokPatterns)
	o := output.New(cfg.CommonTemplate, cfg.Months)
	for _, fd := range cfg.FileDescriptions {
		r, err := g.ParseAll(fd.Patterns, txt)
		if err != nil {
			os.Exit(1)
		}
		if r == nil {
			continue
		}
		values := map[string]interface{}{
			"env":      cfg.EnvVars,
			"grok":     r,
			"filename": cfg.InputFile,
		}
		outputResult, err := o.FromTemplate(fd.Output, values)
		if err != nil {
			continue
		}
		if cfg.NoDryRun {
			run, err := exec.Command("bash", "-c", outputResult).Output()
			fmt.Printf("%s", string(run[:]))
			if err != nil {
				l.Fatal("Run output command", zap.String("command output", string(run[:])), zap.Error(err))
				os.Exit(1)
			}
		} else {
			fmt.Printf("%s", outputResult)

		}
	}
}
