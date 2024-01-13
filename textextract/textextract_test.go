// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package textextract

import (
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

	command := []string{"cat", "FILENAME"}

	err := createFile(filename, fileContent)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed to create file '%s' with contents '%s'`, filename, fileContent)
	}

	output, err := TextExtract(filename, command)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed with error %v`, err)
	}

	assert.Equal(t, fileContent, output, `TestTextExtract : file contents '%v' differs from expected contents '%v'`, output, fileContent)

	err = os.Remove(filename)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed to remove file '%s'`, filename)
	}
}

func TestTextExtractCommandDoesNotExist(t *testing.T) {
	fileContent := "Some Contents\non more than\none line"

	command := []string{"thisCommandDoesNotExist", "FILENAME"}

	err := createFile(filename, fileContent)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed to create file '%s' with contents '%s'`, filename, fileContent)
	}

	_, err = TextExtract(filename, command)

	assert.ErrorIsf(t, err, exec.ErrNotFound, `TestTextExtract : failed with error %v`, err)

	err = os.Remove(filename)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed to remove file '%s'`, filename)
	}
}

func TestTextExtractFilenameDoesNotExist(t *testing.T) {
	command := []string{"cat", "FILENAME"}

	_, err := TextExtract(filename, command)
	if assert.Error(t, err) {
		werr, ok := err.(*exec.ExitError)
		if assert.Truef(t, ok, `TestTextExtract : expected exec.ExitError. Got %T : %v`, err, err) {
			assert.Equalf(t, "exit status 1", werr.Error(), `TestTextExtract : wrong error`)
		}
	}
}
