// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"fileganizer/config"
	"fileganizer/grok"
	"fileganizer/logger"
	"fileganizer/output"
	"fileganizer/textextract"
)

// Version contains the build version string, set at compile time via version.txt.
var (
	Version string = strings.TrimSpace(version)
	//go:embed version.txt
	version string
)

func run() error {
	ctx := context.Background()

	cfg, err := config.New(Version)
	if err != nil {
		if errors.Is(err, config.ErrVersionRequested) {
			return nil
		}
		return err
	}

	txt, err := textextract.TextExtract(ctx, cfg.InputFile, cfg.ExtractTextCommand)
	if err != nil {
		return err
	}
	if cfg.TextOutput {
		fmt.Printf("%v\n", txt)
		return nil
	}
	g, err := grok.New(cfg.GrokPatterns)
	if err != nil {
		return err
	}
	o := output.New(cfg.CommonTemplate, cfg.Months)
	for _, fd := range cfg.FileDescriptions {
		r, err := g.ParseAll(fd.Patterns, txt)
		if err != nil {
			return err
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
			logger.Get().Debug("Silently skipping template", "output", fd.Output, "error", err)
			continue
		}
		if cfg.NoDryRun {
			out, err := exec.CommandContext(ctx, "bash", "-c", outputResult).CombinedOutput()
			fmt.Printf("%s", string(out))
			if err != nil {
				return err
			}
		} else {
			fmt.Printf("%s", outputResult)
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
