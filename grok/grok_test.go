// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package grok

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const contents = "Some text\n" +
	"with some patterns below\n" +
	"Identifier : 123\n" +
	"Other identifier : x123\n" +
	"Old date 1970 is the beginning of time\n"

var grokPatterns = map[string]string{
	"NUMBER":              "[0-9]+",
	"STRING":              "\\w+",
	"JUSTMATCH":           ".*",
	"SPACESANDEMPTYLINES": "[\\s\\n]+",
	"YEAR":                "(?:\\d\\d){1,2}",
	"MONTHNUM2":           "0[1-9]|1[0-2]",
	"MONTHDAY":            "(?:0[1-9])|(?:[12][0-9])|(?:3[01])|[1-9]",
}

var patternsMatching = []string{
	"Identifier : %{NUMBER:identifier}",
	"date %{YEAR:year} is",
}

var patternsNotMatching = []string{
	"Nothing : %{NUMBER:identifier}",
	"identifier : %{NUMBER:identifier}",
	"identifier : %{YEAR:year}",
}

func TestNew(t *testing.T) {
	g, err := New(grokPatterns)
	assert.NoError(t, err)
	assert.Equal(t, len(grokPatterns), len(g.host))
}

func TestParseOK(t *testing.T) {
	g, err := New(grokPatterns)
	require.NoError(t, err)

	for _, p := range patternsMatching {
		r, err := g.Parse(p, contents)
		assert.NoErrorf(t, err, "Fails on pattern '%s' with error %v", p, err)
		assert.Lenf(t, r, 1, "Fails on pattern '%s'", p)
	}
}

func TestParseNothing(t *testing.T) {
	g, err := New(grokPatterns)
	require.NoError(t, err)
	for _, p := range patternsNotMatching {
		r, err := g.Parse(p, contents)
		assert.NoErrorf(t, err, "Fails on pattern '%s' with error %v", p, err)
		assert.Lenf(t, r, 0, "Fails on pattern '%s'", p)
	}
}

func TestParseAllOK(t *testing.T) {
	g, err := New(grokPatterns)
	require.NoError(t, err)

	r, err := g.ParseAll(patternsMatching, contents)
	assert.NoError(t, err)
	assert.Contains(t, r, "identifier")
	assert.Equal(t, r["identifier"], "123")
	assert.Contains(t, r, "year")
	assert.Equal(t, r["year"], "1970")
}

func TestParseAllNothing(t *testing.T) {
	g, err := New(grokPatterns)
	require.NoError(t, err)

	r, err := g.ParseAll(patternsNotMatching, contents)
	assert.NoError(t, err)
	assert.Len(t, r, 0)
}

func TestNew_InvalidPatternRegistration(t *testing.T) {
	badPatterns := map[string]string{"": ""}
	_, err := New(badPatterns)
	assert.Error(t, err)
}

func TestParse_InvalidCompile(t *testing.T) {
	g, err := New(grokPatterns)
	require.NoError(t, err)

	_, err = g.Parse("%{NONEXISTENT:bad}", contents)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NONEXISTENT")
}

func TestParseAll_ErrorMidChain(t *testing.T) {
	g, err := New(grokPatterns)
	require.NoError(t, err)

	// First pattern matches, second fails
	patterns := []string{
		"Identifier : %{NUMBER:identifier}",
		"%{NONEXISTENT:bad}",
	}
	_, err = g.ParseAll(patterns, contents)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NONEXISTENT")
}
