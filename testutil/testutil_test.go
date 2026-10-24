// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package testutil

import (
	"errors"
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

func TestUseTempDir_GetwdError(t *testing.T) {
	err := UseTempDirNoFatal(t, func() (string, error) {
		return "", errors.New("getwd failed")
	}, func(string) error { return nil })
	if err == nil {
		t.Error("expected error from getwd")
	}
}

func TestUseTempDir_ChdirError(t *testing.T) {
	err := UseTempDirNoFatal(t, func() (string, error) {
		return "/tmp", nil
	}, func(string) error {
		return errors.New("chdir failed")
	})
	if err == nil {
		t.Error("expected error from chdir")
	}
}

func TestUseTempDir_CleanupChdirError(t *testing.T) {
	// Test that cleanup logs error when chdir fails
	// We use a custom chdir that fails only on the second call (cleanup)
	callCount := 0
	err := UseTempDirNoFatal(t, func() (string, error) {
		return "/tmp", nil
	}, func(string) error {
		callCount++
		if callCount == 2 {
			return errors.New("chdir failed")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// If we get here, the initial chdir succeeded and cleanup was registered
	// The cleanup will run after this test returns and log the error
}
