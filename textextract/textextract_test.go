// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package textextract

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fileganizer/testutil"
)

const filename = "testfile"

const multiLineContent = "Some Contents\non more than\none line"

func TestTextExtractCat(t *testing.T) {
	testutil.UseTempDir(t)
	command := []string{"cat", "FILENAME"} //nolint:goconst // In the test, it must be explicitly set to "FILENAME"

	require.NoError(t, os.WriteFile(filename, []byte(multiLineContent), 0600))

	output, err := TextExtract(context.Background(), filename, command)
	require.NoError(t, err)

	assert.Equal(t, multiLineContent, output,
		`TestTextExtract : file contents '%v' differs from expected contents '%v'`, output, multiLineContent)
}

func TestTextExtractCatWithArgs(t *testing.T) {
	testutil.UseTempDir(t)
	command := []string{"cat", "-n", "FILENAME"}

	require.NoError(t, os.WriteFile(filename, []byte(multiLineContent), 0600))

	output, err := TextExtract(context.Background(), filename, command)
	require.NoError(t, err)

	assert.Contains(t, output, "Some Contents")
}

func TestTextExtractCommandDoesNotExist(t *testing.T) {
	testutil.UseTempDir(t)
	command := []string{"thisCommandDoesNotExist", "FILENAME"}

	require.NoError(t, os.WriteFile(filename, []byte(multiLineContent), 0600))

	_, err := TextExtract(context.Background(), filename, command)

	assert.ErrorIsf(t, err, exec.ErrNotFound, `TestTextExtract : failed with error %v`, err)
}

func TestTextExtractFilenameDoesNotExist(t *testing.T) {
	command := []string{"cat", "FILENAME"}

	_, err := TextExtract(context.Background(), filename, command)
	require.Error(t, err)

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, "exit status 1", exitErr.Error())
}

func TestTextExtractEmptyCommand(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		_, err := TextExtract(context.Background(), "file.txt", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty command")
	})

	t.Run("empty slice", func(t *testing.T) {
		_, err := TextExtract(context.Background(), "file.txt", []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty command")
	})
}

func TestPlainTextTextExtract(t *testing.T) {
	testutil.UseTempDir(t)
	err := os.WriteFile("testplain.txt", []byte(multiLineContent), 0600)
	require.NoError(t, err)

	output, err := PlainTextTextExtract(context.Background(), "testplain.txt")
	require.NoError(t, err)
	assert.Equal(t, multiLineContent, output)
}

func TestPlainTextTextExtractFileNotFound(t *testing.T) {
	_, err := PlainTextTextExtract(context.Background(), "nonexistent.txt")
	assert.Error(t, err)
}
