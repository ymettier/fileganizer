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

func TestGrokOperations(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) (Grok, error)
		run       func(t *testing.T, g Grok)
	}{
		{
			name: "new",
			setup: func(t *testing.T) (Grok, error) {
				return New(grokPatterns)
			},
			run: func(t *testing.T, g Grok) {
				assert.Equal(t, len(grokPatterns), len(g.host))
			},
		},
		{
			name: "parse all ok",
			setup: func(t *testing.T) (Grok, error) {
				return New(grokPatterns)
			},
			run: func(t *testing.T, g Grok) {
				r, err := g.ParseAll(patternsMatching, contents)
				assert.NoError(t, err)
				assert.Contains(t, r, "identifier")
				assert.Equal(t, r["identifier"], "123")
				assert.Contains(t, r, "year")
				assert.Equal(t, r["year"], "1970")
			},
		},
		{
			name: "parse all nothing",
			setup: func(t *testing.T) (Grok, error) {
				return New(grokPatterns)
			},
			run: func(t *testing.T, g Grok) {
				r, err := g.ParseAll(patternsNotMatching, contents)
				assert.NoError(t, err)
				assert.Len(t, r, 0)
			},
		},
		{
			name: "parse invalid compile",
			setup: func(t *testing.T) (Grok, error) {
				return New(grokPatterns)
			},
			run: func(t *testing.T, g Grok) {
				_, err := g.Parse("%{NONEXISTENT:bad}", contents)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "NONEXISTENT")
			},
		},
		{
			name: "parse all error mid chain",
			setup: func(t *testing.T) (Grok, error) {
				return New(grokPatterns)
			},
			run: func(t *testing.T, g Grok) {
				patterns := []string{
					"Identifier : %{NUMBER:identifier}",
					"%{NONEXISTENT:bad}",
				}
				_, err := g.ParseAll(patterns, contents)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "NONEXISTENT")
			},
		},
		{
			name: "new invalid pattern registration",
			setup: func(t *testing.T) (Grok, error) {
				badPatterns := map[string]string{"": ""}
				return New(badPatterns)
			},
			run: func(t *testing.T, g Grok) {
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, err := tc.setup(t)
			if tc.name == "new invalid pattern registration" {
				assert.Error(t, err)
				tc.run(t, g)
				return
			}
			require.NoError(t, err)
			tc.run(t, g)
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantLen int
		wantErr bool
	}{
		{"identifier number ok", "Identifier : %{NUMBER:identifier}", 1, false},
		{"other identifier ok", "Other identifier : %{STRING:identifier}", 1, false},
		{"date year ok", "date %{YEAR:year} is", 1, false},
		{"nothing number", "Nothing : %{NUMBER:identifier}", 0, false},
		{"identifier string case sensitive", "identifier : %{NUMBER:identifier}", 0, false},
		{"identifier year", "identifier : %{YEAR:year}", 0, false},
	}

	g, err := New(grokPatterns)
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := g.Parse(tc.pattern, contents)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, r, tc.wantLen)
		})
	}
}
