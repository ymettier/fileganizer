// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"fileganizer/config"
	"fileganizer/grok"
	"fileganizer/logger"
	"fileganizer/output"
	"fileganizer/pdftotext"
	"fileganizer/textextract"
)

// Version contains the build version string, set at compile time via version.txt.
var (
	Version = strings.TrimSpace(version)
	//go:embed version.txt
	version string
)

// sniffSize is the number of bytes read from the file for MIME type detection.
const sniffSize = 512

func run() (string, error) {
	// Create a context that is canceled on SIGINT or SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.New(Version, os.Args[1:])
	if err != nil {
		if errors.Is(err, config.ErrVersionRequested) {
			return "", nil
		}
		return "", err
	}

	txt, err := extractText(ctx, &cfg)
	if err != nil {
		return "", err
	}
	if cfg.TextOutput {
		return txt + "\n", nil
	}
	return processFileDescriptions(ctx, &cfg, txt)
}

// detectFileType reads the first bytes of a file and returns its MIME type.
// The MIME type is normalized to the base type without parameters (e.g. "text/plain").
func detectFileType(filename string) (string, error) {
	f, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, sniffSize)
	n, err := f.Read(buf)
	if err != nil {
		return "", err
	}

	mimeType := http.DetectContentType(buf[:n])
	// Strip parameters (e.g. "; charset=utf-8")
	if idx := strings.IndexByte(mimeType, ';'); idx >= 0 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	return mimeType, nil
}

func extractText(ctx context.Context, cfg *config.Config) (string, error) {
	mimeType, err := detectFileType(cfg.InputFile)
	if err != nil {
		return "", fmt.Errorf("failed to detect file type: %w", err)
	}

	l := logger.Get()
	l.Debug("Detected file type", "mime", mimeType, "file", cfg.InputFile)

	te, ok := cfg.TextExtractors[mimeType]
	if !ok {
		return "", fmt.Errorf("no textExtractor configured for MIME type %q", mimeType)
	}

	switch te.Type {
	case "command":
		return textextract.TextExtract(ctx, cfg.InputFile, te.Command)
	case "builtin":
		return builtinExtract(ctx, mimeType, cfg.InputFile)
	default:
		return "", fmt.Errorf("unsupported textExtractor type: %s", te.Type)
	}
}

func builtinExtract(ctx context.Context, mimeType, filename string) (string, error) {
	switch mimeType {
	case "application/pdf":
		return pdftotext.PDFTextExtract(ctx, filename)
	case "text/plain":
		return textextract.PlainTextTextExtract(ctx, filename)
	default:
		return "", fmt.Errorf("no builtin textExtractor for MIME type %q", mimeType)
	}
}

func processFileDescriptions(ctx context.Context, cfg *config.Config, txt string) (string, error) {
	g, err := grok.New(cfg.GrokPatterns)
	if err != nil {
		return "", err
	}
	o := output.New(cfg.CommonTemplate, cfg.Months)
	for _, fd := range cfg.FileDescriptions {
		r, err := g.ParseAll(fd.Patterns, txt)
		if err != nil {
			return "", err
		}
		if r == nil {
			continue
		}
		values := output.TemplateData{
			Env:      cfg.EnvVars,
			Grok:     r,
			Filename: cfg.InputFile,
		}
		outputResult, err := o.FromTemplate(fd.Output, values)
		if err != nil {
			logger.Get().Warn("Silently skipping template", "output", fd.Output, "error", err)
			continue
		}
		if cfg.NoDryRun {
			out, err := exec.CommandContext(ctx, "bash", "-c", outputResult).CombinedOutput() //nolint:gosec // intentional shell execution
			// Command output goes to stdout for user consumption.
			return string(out), err
		}
		return outputResult, nil
	}
	return "", nil
}

func main() {
	res, err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if res != "" {
		fmt.Printf("%s", res)
	}
}
