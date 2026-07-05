// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fileganizer/config"
	"fileganizer/grok"
	"fileganizer/logger"
	"fileganizer/output"
	"fileganizer/textextract"
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "embed"
)

// Version contains the build version string, set at compile time via version.txt.
var (
	Version string = strings.TrimSpace(version)
	//go:embed version.txt
	version string
)

func main() {
	l := logger.Get()

	cfg, err := config.New(Version)
	if err != nil {
		os.Exit(1)
	}

	txt, err := textextract.TextExtract(context.Background(), cfg.InputFile, cfg.ExtractTextCommand)
	if err != nil {
		os.Exit(1)
	}
	if cfg.TextOutput {
		fmt.Printf("%v\n", txt)
		os.Exit(0)
	}
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
		values := map[string]any{
			"env":      cfg.EnvVars,
			"grok":     r,
			"filename": cfg.InputFile,
		}
		outputResult, err := o.FromTemplate(fd.Output, values)
		if err != nil {
			continue
		}
		if cfg.NoDryRun {
			run, err := exec.CommandContext(context.Background(), "bash", "-c", outputResult).Output()
			fmt.Printf("%s", string(run))
			if err != nil {
				l.Error("Run output command", "command output", string(run), "error", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("%s", outputResult)
		}
	}
}
