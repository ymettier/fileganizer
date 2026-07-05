// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package textextract

import (
	"context"
	"fileganizer/logger"

	"os/exec"
)

const templateFileName = "FILENAME"

func TextExtract(ctx context.Context, filename string, command []string) (string, error) {
	l := logger.Get()
	l.Debug("ExtractTextCommand", "command", command)
	args := make([]string, 0)
	for i, v := range command {
		if i == 0 {
			continue
		}
		if v == templateFileName {
			args = append(args, filename)
		} else {
			args = append(args, v)
		}
	}
	l.Debug("ExtractTextCommand", "command", command[0])
	source, err := exec.CommandContext(ctx, command[0], args...).Output() //nolint:gosec
	if err != nil {
		l.Error("ExtractTextCommand failed", "error", err)
		return "", err
	}
	return string(source), nil
}
