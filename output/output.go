// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"fileganizer/logger"
)

// Output holds template configuration and renders Go templates with parsed data.
type Output struct {
	commonTemplate string
	months         map[string][]string
}

// New creates an Output with an optional common template prefix and month mappings.
func New(tpl string, months map[string][]string) Output {
	var o Output
	o.commonTemplate = tpl
	o.months = months
	return o
}

// MonthIndex returns the zero-padded month number (01-12) for the given month
// name by looking it up in the configured month lists. Returns the input as-is
// if not found.
func (o Output) MonthIndex(month string) string {
	for _, months := range o.months {
		for i, data := range months {
			if data == month {
				return fmt.Sprintf("%02d", i+1)
			}
		}
	}
	return month
}

// FromTemplate renders the output template (prefixed with CommonTemplate if set)
// using the provided variables and returns the result as a string.
func (o Output) FromTemplate(tmpl string, vars map[string]any) (string, error) {
	l := logger.Get()
	funcMap := template.FuncMap{
		"ToUpper":            strings.ToUpper,
		"ToLower":            strings.ToLower,
		"MonthIndex":         func(m string) string { return o.MonthIndex(m) }, //nolint:gocritic
		"NowYYYY":            func() string { return time.Now().Format("2006") },
		"NowYYYYMMDD":        func() string { return time.Now().Format("20060102") },
		"NowYYYYMMDD_HHMMSS": func() string { return time.Now().Format("20060102_030405") },
	}
	fullTpl := tmpl
	if o.commonTemplate != "" {
		fullTpl = strings.Join(append([]string{o.commonTemplate}, tmpl), "\n")
	}

	parsed, err := template.New("main").Funcs(funcMap).Parse(fullTpl)
	if err != nil {
		l.Error("Failed to parse template", "error", err)
		return "", err
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, vars); err != nil {
		l.Error("Failed to execute template", "error", err)
		return "", err
	}

	return buf.String(), nil
}
