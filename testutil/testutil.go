// Copyright 2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package testutil

import (
	"os"
	"testing"
)

// UseTempDir creates a temporary directory and changes the working directory to
// it. The original directory is restored via t.Cleanup.
func UseTempDir(t *testing.T) {
	useTempDir(t, os.Getwd, os.Chdir)
}

// useTempDir is the internal implementation that accepts mock functions for testing.
func useTempDir(t *testing.T, getwd func() (string, error), chdir func(string) error) {
	t.Helper()
	dir := t.TempDir()
	oldDir, err := getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := chdir(oldDir); err != nil {
			t.Logf("failed to restore working directory: %v", err)
		}
	})
}

// UseTempDirNoFatal is like useTempDir but returns errors instead of calling t.Fatal.
// Used for testing error paths.
func UseTempDirNoFatal(t *testing.T, getwd func() (string, error), chdir func(string) error) error {
	t.Helper()
	dir := t.TempDir()
	oldDir, err := getwd()
	if err != nil {
		return err
	}
	if err := chdir(dir); err != nil {
		return err
	}
	t.Cleanup(func() {
		if err := chdir(oldDir); err != nil {
			t.Logf("failed to restore working directory: %v", err)
		}
	})
	return nil
}
