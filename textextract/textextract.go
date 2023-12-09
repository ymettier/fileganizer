// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package textextract

import (
	"fileganizer/logger"

	"os/exec"

	"go.uber.org/zap"
)

func TextExtract(filename string, command []string) (string, error) {
	l := logger.Get()
	l.Debug("ExtractTextCommand", zap.Strings("command", command))
	args := make([]string, 0)
	for i, v := range command {
		if i == 0 {
			continue
		}
		if v == "FILENAME" {
			args = append(args, filename)
		} else {
			args = append(args, v)
		}
	}
	l.Debug("ExtractTextCommand", zap.String("command", command[0]))
	source, err := exec.Command(command[0], args...).Output()
	if err != nil {
		l.Fatal("ExtractTextCommand failed", zap.Error(err))
		return "", err
	}
	return string(source), nil
}
