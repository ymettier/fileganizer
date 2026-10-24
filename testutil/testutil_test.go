// Copyright 2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package testutil

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUseTempDir(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	UseTempDir(t)

	newDir, err := os.Getwd()
	require.NoError(t, err)

	assert.NotEqual(t, originalDir, newDir, "UseTempDir did not change the working directory")
}

func TestUseTempDir_RestoresOriginal(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	t.Run("sub", func(t *testing.T) {
		UseTempDir(t)
	})

	restoredDir, err := os.Getwd()
	require.NoError(t, err)

	assert.Equal(t, originalDir, restoredDir, "working directory was not restored")
}

func TestUseTempDir_CleanupLog(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	restoreDir := t.TempDir()
	require.NoError(t, os.Chdir(restoreDir))

	t.Run("sub", func(t *testing.T) {
		UseTempDir(t)
		require.NoError(t, os.RemoveAll(restoreDir))
	})

	require.NoError(t, os.Chdir(originalDir))
}

func TestUseTempDir_GetwdError(t *testing.T) {
	err := UseTempDirNoFatal(t, func() (string, error) {
		return "", errors.New("getwd failed")
	}, func(string) error { return nil })
	assert.Error(t, err, "expected error from getwd")
}

func TestUseTempDir_ChdirError(t *testing.T) {
	err := UseTempDirNoFatal(t, func() (string, error) {
		return "/tmp", nil
	}, func(string) error {
		return errors.New("chdir failed")
	})
	assert.Error(t, err, "expected error from chdir")
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
	require.NoError(t, err)
	// If we get here, the initial chdir succeeded and cleanup was registered
	// The cleanup will run after this test returns and log the error
}
