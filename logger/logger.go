// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type ctxKey struct{}

var once sync.Once

var logger *Logger

type Logger struct {
	*slog.Logger
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}

func getGitRevision() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, v := range buildInfo.Settings {
		if v.Key == "vcs.revision" {
			return v.Value
		}
	}
	return ""
}

type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			_ = handler.Handle(ctx, r)
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// Loggers are defined with these environment variables:
// - LOG_TXT_FILENAME
// - LOG_JSON_FILENAME
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

	// Define logFd for outputs
	logWriters := make(map[string]io.Writer)
	for _, t := range []string{"TXT", "JSON"} {
		filename := os.Getenv("LOG_" + t + "_FILENAME")

		if filename != "" {
			switch filename {
			case "stdout":
				logWriters[t] = os.Stdout
			case "stderr":
				logWriters[t] = os.Stderr
			default:
				logWriters[t] = &lumberjack.Logger{
					Filename:   filename,
					MaxSize:    5,
					MaxBackups: 10,
					MaxAge:     14,
					Compress:   true,
				}
			}
		}
	}
	if len(logWriters) == 0 {
		logWriters["TXT"] = os.Stderr
	}

	var handlers []slog.Handler

	// Define and append TXT logger core
	if w, ok := logWriters["TXT"]; ok {
		handlers = append(handlers, slog.NewTextHandler(w, opts))
	}

	// Define JSON logger core
	if w, ok := logWriters["JSON"]; ok {
		jsonOpts := &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.String("timestamp", a.Value.Time().Format("2006-01-02T15:04:05.000Z0700"))
				}
				return a
			},
		}
		gitRevision := getGitRevision()
		h := slog.NewJSONHandler(w, jsonOpts).WithAttrs([]slog.Attr{
			slog.String("git_revision", gitRevision),
			slog.String("go_version", runtime.Version()),
		})
		handlers = append(handlers, h)
	}

	// Create new logger
	return &Logger{slog.New(&multiHandler{handlers: handlers})}
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
