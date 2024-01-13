// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package grok

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	g := New(grokPatterns)
	assert.Equal(t, len(grokPatterns), len(g.Host))
}

func TestParseOK(t *testing.T) {
	g := New(grokPatterns)

	for _, p := range patternsMatching {
		r, err := g.Parse(p, contents)
		assert.NoErrorf(t, err, "Fails on pattern '%s' with error %v", p, err)
		assert.Lenf(t, r, 1, "Fails on pattern '%s'", p)
	}
}

func TestParseNothing(t *testing.T) {
	g := New(grokPatterns)
	for _, p := range patternsNotMatching {
		r, err := g.Parse(p, contents)
		assert.NoErrorf(t, err, "Fails on pattern '%s' with error %v", p, err)
		assert.Lenf(t, r, 0, "Fails on pattern '%s'", p)
	}
}

func TestParseAllOK(t *testing.T) {
	g := New(grokPatterns)

	r, err := g.ParseAll(patternsMatching, contents)
	assert.NoError(t, err)
	assert.Contains(t, r, "identifier")
	assert.Equal(t, r["identifier"], "123")
	assert.Contains(t, r, "year")
	assert.Equal(t, r["year"], "1970")
}

func TestParseAllNothing(t *testing.T) {
	g := New(grokPatterns)

	r, err := g.ParseAll(patternsNotMatching, contents)
	assert.NoError(t, err)
	assert.Len(t, r, 0)
}
