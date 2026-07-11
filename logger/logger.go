// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package logger

import (
	"context"
	"io"
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

// Logger wraps slog.Logger to provide a singleton logging instance with optional
// log rotation via lumberjack.
type Logger struct {
	*slog.Logger
}

// LogOptions controls the logger output format, level, destination, and rotation.
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
		Filename:   filename,
		MaxSize:    opts.MaxSize,
		MaxBackups: opts.MaxBackups,
		MaxAge:     opts.MaxAge,
		Compress:   opts.Compress,
	}
	return l, true
}

func resolveLogLevel(opts *LogOptions) slog.Level {
	levelStr := ""
	if opts != nil && opts.Level != "" {
		levelStr = opts.Level
	} else {
		levelStr = os.Getenv("FILEGANIZER_LOGGING_LEVEL")
	}
	if levelStr == "" {
		return slog.LevelInfo
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToUpper(levelStr))); err != nil {
		slog.Default().Error("invalid level, defaulting to INFO", "error", err)
		return slog.LevelInfo
	}
	return level
}

// newLogger creates a new logger based on opts or environment variables if opts is nil.
func newLogger(opts *LogOptions) *Logger {
	handlerOpts := &slog.HandlerOptions{
		Level: resolveLogLevel(opts),
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
