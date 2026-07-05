// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

type ctxKey struct{}

var (
	mu     sync.RWMutex
	logger *Logger
)

type Logger struct {
	*slog.Logger
}

type LogOptions struct {
	JSON       bool
	Level      string
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// getWriter returns an io.Writer and whether lumberjack rotation is being used.
func getWriter(opts *LogOptions) (io.Writer, bool) {
	filename := ""
	if opts != nil && opts.Filename != "" {
		filename = opts.Filename
	}

	if filename == "" {
		return os.Stderr, false
	}

	switch filename {
	case "stdout":
		return os.Stdout, false
	case "stderr": //nolint:goconst
		return os.Stderr, false
	}

	l := &lumberjack.Logger{
		Filename: filename,
	}
	if opts != nil {
		l.MaxSize = opts.MaxSize
		l.MaxBackups = opts.MaxBackups
		l.MaxAge = opts.MaxAge
		l.Compress = opts.Compress
	} else {
		l.MaxSize = 5
		l.MaxBackups = 10
		l.MaxAge = 14
		l.Compress = true
	}
	return l, true
}

// newLogger creates a new logger based on opts or environment variables if opts is nil.
func newLogger(opts *LogOptions) *Logger {
	// Define log level
	level := slog.LevelInfo
	var levelStr string
	if opts != nil && opts.Level != "" {
		levelStr = opts.Level
	} else {
		levelStr = os.Getenv("LOG_LEVEL")
	}

	if levelStr != "" {
		var l slog.Level
		if err := l.UnmarshalText([]byte(strings.ToUpper(levelStr))); err != nil {
			log.Println(fmt.Errorf("invalid level, defaulting to INFO: %w", err))
		} else {
			level = l
		}
	}

	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	w, usingLumberjack := getWriter(opts)

	// Create new logger
	var handler slog.Handler
	if opts != nil && opts.JSON {
		handler = slog.NewJSONHandler(w, handlerOpts)
	} else {
		handler = slog.NewTextHandler(w, handlerOpts)
	}
	l := &Logger{slog.New(handler)}

	if opts != nil {
		attrs := []any{
			slog.String("level", opts.Level),
			slog.String("filename", opts.Filename),
			slog.Bool("json", opts.JSON),
		}
		if usingLumberjack {
			attrs = append(attrs,
				slog.Int("maxSize", opts.MaxSize),
				slog.Int("maxBackups", opts.MaxBackups),
				slog.Int("maxAge", opts.MaxAge),
				slog.Bool("compress", opts.Compress),
			)
		}
		l.Info("Logger configuration", attrs...)
	}

	return l
}

// Get initializes a Logger instance if it has not been initialized
// already and returns the same instance for subsequent calls.
func Get() *Logger {
	mu.RLock()
	l := logger
	mu.RUnlock()

	if l != nil {
		return l
	}

	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		logger = newLogger(nil)
	}
	return logger
}

// Reset re-initializes the global logger with the provided options.
func Reset(opts *LogOptions) {
	mu.Lock()
	defer mu.Unlock()
	logger = newLogger(opts)
}

// FromCtx returns the Logger associated with the ctx. If no logger
// is associated, the default logger is returned.
func FromCtx(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	}
	return Get()
}

// WithCtx returns a copy of ctx with the Logger attached.
func WithCtx(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}
