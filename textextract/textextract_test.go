// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package textextract

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

const filename = "testfile"

func createFile(filename, contents string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(contents)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	return nil
}

func TestTextExtractCat(t *testing.T) {
	fileContent := "Some Contents\non more than\none line"

	command := []string{"cat", "FILENAME"} //nolint:goconst // In the test, it must be explicitly set to "FILENAME"

	err := createFile(filename, fileContent)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed to create file '%s' with contents '%s'`, filename, fileContent)
	}
	defer os.Remove(filename)

	output, err := TextExtract(context.Background(), filename, command)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed with error %v`, err)
	}

	assert.Equal(t, fileContent, output, `TestTextExtract : file contents '%v' differs from expected contents '%v'`, output, fileContent)
}

func TestTextExtractCommandDoesNotExist(t *testing.T) {
	fileContent := "Some Contents\non more than\none line"

	command := []string{"thisCommandDoesNotExist", "FILENAME"}

	err := createFile(filename, fileContent)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed to create file '%s' with contents '%s'`, filename, fileContent)
	}
	defer os.Remove(filename)

	_, err = TextExtract(context.Background(), filename, command)

	assert.ErrorIsf(t, err, exec.ErrNotFound, `TestTextExtract : failed with error %v`, err)
}

func TestTextExtractFilenameDoesNotExist(t *testing.T) {
	command := []string{"cat", "FILENAME"}

	_, err := TextExtract(context.Background(), filename, command)
	if assert.Error(t, err) {
		werr, ok := err.(*exec.ExitError)
		if assert.Truef(t, ok, `TestTextExtract : expected exec.ExitError. Got %T : %v`, err, err) {
			assert.Equalf(t, "exit status 1", werr.Error(), `TestTextExtract : wrong error`)
		}
	}
}

func TestTextExtractEmptyCommand(t *testing.T) {
	_, err := TextExtract(context.Background(), "file.txt", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")

	_, err = TextExtract(context.Background(), "file.txt", []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
}
