// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package testutil

import (
	"os"
	"testing"
)

// UseTempDir creates a temporary directory and changes the working directory to
// it. The original directory is restored via t.Cleanup.
func UseTempDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Logf("failed to restore working directory: %v", err)
		}
	})
}
