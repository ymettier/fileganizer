// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package grok

import (
	"fileganizer/logger"
	"fmt"

	"github.com/logrusorgru/grokky"
	"go.uber.org/zap"
)

type Grok struct {
	Host grokky.Host
}

func New(patterns map[string]string) Grok {
	var g Grok
	g.Host = grokky.New()
	for k, p := range patterns {
		g.Host.Must(k, p)
	}
	return g
}

func (g *Grok) ParseAll(grokPatterns []string, text string) (map[string]string, error) {
	var result = make(map[string]string)
	l := logger.Get()
	for _, p := range grokPatterns {
		r, err := g.Parse(p, text)
		if err != nil {
			return nil, err
		}
		if len(r) == 0 {
			l.Debug("No pattern matched", zap.String("pattern", p))
			return nil, nil
		} else {
			for k, v := range r {
				result[k] = v
			}
		}
	}
	return result, nil
}

func (g *Grok) Parse(grokPattern string, text string) (map[string]string, error) {
	l := logger.Get()
	l.Debug("Testing pattern", zap.String("pattern", grokPattern), zap.String("text", text))
	p, err := g.Host.Compile(grokPattern)
	if err != nil {
		l.Fatal("grok compile failed", zap.Error(err))
		return nil, err
	}
	result := p.Parse(text)

	return result, nil
}

func main() {
	fmt.Println("vim-go")
}
