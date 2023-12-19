// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	wantedVersion := "1.2.3"
	output := printVersion(wantedVersion)
	s := strings.Split(output, "\n")
	if s[0] != "Version        : "+wantedVersion {
		t.Fatalf(`printVersion(%s) = %q, beginning with %q, wanted %s, nil`, wantedVersion, output, s[0], wantedVersion)
	}
}
