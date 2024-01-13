// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	wantedVersion := "1.2.3"
	output := printVersion(wantedVersion)
	s := strings.Split(output, "\n")
	assert.Equal(t, "Version        : "+wantedVersion, s[0], "Printing version")
}
