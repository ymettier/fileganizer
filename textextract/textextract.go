// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package textextract

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"fileganizer/logger"
)

const templateFileName = "FILENAME"

// TextExtract runs an external command to extract text from a file. The special
// token "FILENAME" in the command arguments is replaced with the actual filename.
func TextExtract(ctx context.Context, filename string, command []string) (string, error) {
	l := logger.Get()
	l.Debug("TextExtract", "command", command)
	if len(command) == 0 {
		return "", errors.New("empty command")
	}
	args := make([]string, 0, len(command)-1)
	for _, v := range command[1:] {
		if v == templateFileName {
			args = append(args, filename)
		} else {
			args = append(args, v)
		}
	}
	source, err := exec.CommandContext(ctx, command[0], args...).Output() //nolint:gosec
	if err != nil {
		return "", err
	}
	return string(source), nil
}

// PlainTextTextExtract reads a plain text file and returns its contents as a string.
func PlainTextTextExtract(_ context.Context, filename string) (string, error) {
	l := logger.Get()
	l.Debug("PlainTextTextExtract", "file", filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
