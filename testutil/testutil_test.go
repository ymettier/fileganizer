// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package testutil

import (
	"os"
	"testing"
)

func TestUseTempDir(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	UseTempDir(t)

	newDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if newDir == originalDir {
		t.Error("UseTempDir did not change the working directory")
	}
}

func TestUseTempDir_RestoresOriginal(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sub", func(t *testing.T) {
		UseTempDir(t)
	})

	restoredDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if restoredDir != originalDir {
		t.Errorf("working directory was not restored: got %q, want %q", restoredDir, originalDir)
	}
}

func TestUseTempDir_CleanupLog(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sub", func(t *testing.T) {
		UseTempDir(t)
		dir, _ := os.Getwd()
		os.RemoveAll(dir)
	})

	restoredDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if restoredDir != originalDir {
		t.Errorf("working directory was not restored: got %q, want %q", restoredDir, originalDir)
	}
}
