// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ctxKey struct{}

var once sync.Once

var logger *zap.Logger

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

// Loggers are defined with these environment variables:
// - LOG_TXT_FILENAME
// - LOG_JSON_FILENAME
//
// The value can be either "stdout", "stderr" or a filename.
//
// LOG_LEVEL environment variable sets the log level.
func newLogger() *zap.Logger {
	// Define log level
	level := zap.InfoLevel
	levelEnv := os.Getenv("LOG_LEVEL")
	if levelEnv != "" {
		levelFromEnv, err := zapcore.ParseLevel(levelEnv)
		if err != nil {
			log.Println(
				fmt.Errorf("invalid level, defaulting to INFO: %w", err),
			)
		}
		level = levelFromEnv
	}

	logLevel := zap.NewAtomicLevelAt(level)

	// Define logFd for outputs
	logFd := make(map[string]zapcore.WriteSyncer)
	for _, t := range []string{"TXT", "JSON"} {
		filename := os.Getenv("LOG_" + t + "_FILENAME")

		if filename != "" {
			switch filename {
			case "stdout":
				logFd[t] = zapcore.AddSync(os.Stdout)
			case "stderr":
				logFd[t] = zapcore.AddSync(os.Stderr)
			default:
				logFd[t] = zapcore.AddSync(&lumberjack.Logger{
					Filename:   filename,
					MaxSize:    5,
					MaxBackups: 10,
					MaxAge:     14,
					Compress:   true,
				})
			}
		}
	}
	if len(logFd) == 0 {
		logFd["TXT"] = zapcore.AddSync(os.Stderr)
	}

	// Define main core
	core := zapcore.NewTee()

	// Define and append TXT logger core
	if _, ok := logFd["TXT"]; ok {
		loggerTxtCfg := zap.NewDevelopmentEncoderConfig()
		loggerTxtCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		loggerTxtEncoder := zapcore.NewConsoleEncoder(loggerTxtCfg)
		loggerCore := zapcore.NewCore(loggerTxtEncoder, logFd["TXT"], logLevel)
		core = zapcore.NewTee(core, loggerCore)
	}

	// Define JSON logger core
	if _, ok := logFd["JSON"]; ok {
		loggerJSONCfg := zap.NewProductionEncoderConfig()
		loggerJSONCfg.TimeKey = "timestamp"
		loggerJSONCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		loggerJSONEncoder := zapcore.NewJSONEncoder(loggerJSONCfg)

		gitRevision := getGitRevision()

		loggerCore := zapcore.NewCore(loggerJSONEncoder, logFd["JSON"], logLevel).
			With(
				[]zapcore.Field{
					zap.String("git_revision", gitRevision),
					zap.String("go_version", runtime.Version()),
				},
			)
		core = zapcore.NewTee(core, loggerCore)
	}

	// Create new zap
	return zap.New(core)
}

// Get initializes a zap.Logger instance if it has not been initialized
// already and returns the same instance for subsequent calls.
func Get() *zap.Logger {
	once.Do(func() {
		logger = newLogger()
	})

	return logger
}

// FromCtx returns the Logger associated with the ctx. If no logger
// is associated, the default logger is returned, unless it is nil
// in which case a disabled logger is returned.
func FromCtx(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	} else if l := logger; l != nil {
		return l
	}

	return zap.NewNop()
}

// WithCtx returns a copy of ctx with the Logger attached.
func WithCtx(ctx context.Context, l *zap.Logger) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		if lp == l {
			// Do not store same logger.
			return ctx
		}
	}

	return context.WithValue(ctx, ctxKey{}, l)
}
