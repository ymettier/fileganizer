// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package grok

import (
	"fileganizer/logger"

	"github.com/logrusorgru/grokky"
)

// Grok wraps a grokky host to compile and match grok patterns against text.
type Grok struct {
	Host grokky.Host
}

// New creates a Grok instance and registers the given named patterns.
func New(patterns map[string]string) Grok {
	var g Grok
	g.Host = grokky.New()
	for k, p := range patterns {
		g.Host.Must(k, p)
	}
	return g
}

// ParseAll applies each grok pattern in order and merges all named captures
// into a single result map. Returns nil if no pattern matched.
func (g *Grok) ParseAll(grokPatterns []string, text string) (map[string]string, error) {
	var result = make(map[string]string)
	l := logger.Get()
	for _, p := range grokPatterns {
		r, err := g.Parse(p, text)
		if err != nil {
			return nil, err
		}
		if len(r) == 0 {
			l.Debug("No pattern matched", "pattern", p)
			return nil, nil
		} else {
			for k, v := range r {
				result[k] = v
			}
		}
	}
	return result, nil
}

// Parse compiles a single grok pattern and extracts named captures from text.
func (g *Grok) Parse(grokPattern, text string) (map[string]string, error) {
	l := logger.Get()
	l.Debug("Testing pattern", "pattern", grokPattern, "text", text)
	p, err := g.Host.Compile(grokPattern)
	if err != nil {
		l.Error("grok compile failed", "error", err)
		return nil, err
	}
	result := p.Parse(text)

	return result, nil
}
