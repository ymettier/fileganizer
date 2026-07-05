// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func captureOutput(f func() error) (string, error) {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := f()
	os.Stdout = orig
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out), err
}

func TestFileykjwmwqqjhght(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }() // os.Args is a "global variable", so keep the state from before the test, and restore it after.

	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhgh.yaml", "-f", "testdata/ykjwmwqqjhgh.txt"}

	output, err := captureOutput(func() error {
		main()
		return nil
	})
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice Summary\n  date: 2014-03-27\n  number: 001\n")
}

func TestFileykjwmwqqjhghtEnv(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }() // os.Args is a "global variable", so keep the state from before the test, and restore it after.

	os.Setenv("SOMEVAR", "magic")
	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhghEnv.yaml", "-f", "testdata/ykjwmwqqjhgh.txt"}

	output, err := captureOutput(func() error {
		main()
		return nil
	})
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice magic Summary\n  date: 2014-03-27\n  number: 001\n")
}

func TestFileNonMatchingPattern(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhghNoMatch.yaml", "-f", "testdata/ykjwmwqqjhgh.txt"}

	output, err := captureOutput(func() error {
		main()
		return nil
	})
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice Summary\n  date: 2014-03-27\n  number: 001\n")
	assert.NotContains(t, output, "should not appear")
}

func TestFileBrokenTemplate(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhghBrokenTpl.yaml", "-f", "testdata/ykjwmwqqjhgh.txt"}

	output, err := captureOutput(func() error {
		main()
		return nil
	})
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice Summary\n  date: 2014-03-27\n  number: 001\n")
}

func TestFileRunMode(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhghRun.yaml", "-f", "testdata/ykjwmwqqjhgh.txt", "-r"}

	output, err := captureOutput(func() error {
		main()
		return nil
	})
	assert.Nil(t, err)
	assert.Contains(t, output, "run mode works")
}

func TestFileFrenchMonths(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhghFrench.yaml", "-f", "testdata/ykjwmwqqjhghFrench.txt"}

	output, err := captureOutput(func() error {
		main()
		return nil
	})
	assert.Nil(t, err)
	assert.Contains(t, output, "08-27-2014")
}

func TestRunMissingConfigFile(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"./fileganizer", "-c", "testdata/nonexistent.yaml", "-f", "testdata/ykjwmwqqjhgh.txt"}

	err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent.yaml")
}

func TestRunMissingInputFile(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhgh.yaml", "-f", "testdata/nonexistent.txt"}

	err := run()
	assert.Error(t, err)
}

func TestRunTextOutputFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhgh.yaml", "-f", "testdata/ykjwmwqqjhgh.txt", "-t"}

	output, err := captureOutput(func() error {
		return run()
	})
	assert.NoError(t, err)
	assert.Contains(t, output, "Invoice")
}

func TestRunBrokenGrokPattern(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"./fileganizer", "-c", "testdata/config.ykjwmwqqjhghBrokenGrok.yaml", "-f", "testdata/ykjwmwqqjhgh.txt"}

	err := run()
	assert.Error(t, err)
}
