// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package grok

import (
	"github.com/logrusorgru/grokky"

	"fileganizer/logger"
)

// Grok wraps a grokky host to compile and match grok patterns against text.
type Grok struct {
	host grokky.Host
}

// New creates a Grok instance and registers the given named patterns.
func New(patterns map[string]string) (Grok, error) {
	var g Grok
	g.host = grokky.New()
	for k, p := range patterns {
		if err := g.host.Add(k, p); err != nil {
			return g, err
		}
	}
	return g, nil
}

// ParseAll applies each grok pattern in order and merges all named captures
// into a single result map. All patterns must match on the text; the first
// non-matching pattern aborts the entire set and returns nil.
func (g *Grok) ParseAll(grokPatterns []string, text string) (map[string]string, error) {
	var result = make(map[string]string)
	for _, p := range grokPatterns {
		r, err := g.Parse(p, text)
		if err != nil {
			return nil, err
		}
		if len(r) == 0 {
			logger.Get().Debug("No pattern matched", "pattern", p)
			return nil, nil
		}
		for k, v := range r {
			result[k] = v
		}
	}
	return result, nil
}

// Parse compiles a single grok pattern and extracts named captures from text.
func (g *Grok) Parse(grokPattern, text string) (map[string]string, error) {
	l := logger.Get()
	l.Debug("Testing pattern", "pattern", grokPattern, "text", text)
	p, err := g.host.Compile(grokPattern)
	if err != nil {
		return nil, err
	}
	result := p.Parse(text)

	return result, nil
}
