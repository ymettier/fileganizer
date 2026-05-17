// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"syscall"
)

type ctxKey struct{}

var once sync.Once

var logger *Logger

type Logger struct {
	*slog.Logger
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.Error(msg, args...)
	os.Exit(1)
}

// Loggers are defined with these environment variables:
// - LOG_TXT_FILENAME
//
// The value can be either "stdout", "stderr" or a filename.
//
// LOG_LEVEL environment variable sets the log level.
func newLogger() *Logger {
	// Define log level
	level := slog.LevelInfo
	levelEnv := os.Getenv("LOG_LEVEL")
	if levelEnv != "" {
		var l slog.Level
		if err := l.UnmarshalText([]byte(strings.ToUpper(levelEnv))); err != nil {
			slog.Warn("invalid level, defaulting to INFO", "error", err)
		} else {
			level = l
		}
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Define log writer
	var w io.Writer = os.Stderr
	filename := os.Getenv("LOG_TXT_FILENAME")

	if filename != "" {
		switch filename {
		case "stdout":
			w = os.Stdout
		case "stderr":
			w = os.Stderr
		default:
			f, err := os.OpenFile(filename, //nolint:gosec // The user specifies the filename in the configuration file.
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IRGRP|syscall.S_IROTH,
			)
			if err != nil {
				slog.Warn("failed to open log file, defaulting to stderr", "filename", filename, "error", err)
			} else {
				w = f
			}
		}
	}

	// Create new logger
	return &Logger{slog.New(slog.NewTextHandler(w, opts))}
}

// Get initializes a Logger instance if it has not been initialized
// already and returns the same instance for subsequent calls.
func Get() *Logger {
	once.Do(func() {
		logger = newLogger()
	})

	return logger
}

// FromCtx returns the Logger associated with the ctx. If no logger
// is associated, the default logger is returned, unless it is nil
// in which case a disabled logger is returned.
func FromCtx(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	} else if l := logger; l != nil {
		return l
	}

	return &Logger{slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// WithCtx returns a copy of ctx with the Logger attached.
func WithCtx(ctx context.Context, l *Logger) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		if lp == l {
			// Do not store same logger.
			return ctx
		}
	}

	return context.WithValue(ctx, ctxKey{}, l)
}
