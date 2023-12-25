// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package textextract

import (
	"errors"
	"os"
	"os/exec"
	"testing"
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

	if output != fileContent {
		t.Fatalf(`TestTextExtract : file contents '%v' differs from expected contents '%v'`, output, fileContent)
	}
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
	if err != nil {
		if !errors.Is(err, exec.ErrNotFound) {
			t.Fatalf(`TestTextExtract : failed with error %v`, err)
		}
	}

	err = os.Remove(filename)
	if err != nil {
		t.Fatalf(`TestTextExtract : failed to remove file '%s'`, filename)
	}
}

func TestTextExtractFilenameDoesNotExist(t *testing.T) {
	command := []string{"cat", "FILENAME"}

	_, err := TextExtract(filename, command)
	if err != nil {
		want := "exit status 1"
		if werr, ok := err.(*exec.ExitError); ok {
			if s := werr.Error(); s != want {
				t.Fatalf(`TestTextExtract : running with file that does not exists, got '%q', wants '%q'`, s, want)
			}
		} else {
			t.Fatalf("expected *exec.ExitError from command with file that does not exist; got %T: %v", err, err)
		}
	}
}
