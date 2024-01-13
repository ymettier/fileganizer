// Copyright 2023 The Fileganizer Authors. All rights reserved.
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
	assert.Equal(t, output, "Invoice Summary\n  date: 2014-03-27\n  number: 001\n")
}
