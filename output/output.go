// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"fileganizer/logger"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// Output holds template configuration and renders Go templates with parsed data.
type Output struct {
	CommonTemplate string
	Months         map[string][]string
}

// New creates an Output with an optional common template prefix and month mappings.
func New(tpl string, months map[string][]string) Output {
	var o Output
	o.CommonTemplate = tpl
	o.Months = months
	return o
}

// MonthIndex returns the zero-padded month number (01-12) for the given month
// name by looking it up in the configured month lists. Returns the input as-is
// if not found.
func (o Output) MonthIndex(month string) string {
	for _, months := range o.Months {
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
	fulltemplate := tmpl
	if o.CommonTemplate != "" {
		fulltemplate = strings.Join(append([]string{o.CommonTemplate}, tmpl), "\n")
	}

	mytemplate, err := template.New("main").Funcs(funcMap).Parse(fulltemplate)
	if err != nil {
		l.Error("Failed to parse template", "error", err)
		return "", err
	}

	var doc bytes.Buffer
	if err := mytemplate.Execute(&doc, vars); err != nil {
		l.Error("Failed to execute template", "error", err)
		return "", err
	}

	return doc.String(), nil
}
