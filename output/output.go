// Copyright 2023 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"fileganizer/logger"
	"fmt"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
)

type Output struct {
	CommonTemplate string
	Months         map[string][]string
}

func New(tpl string, months map[string][]string) Output {
	var o Output
	o.CommonTemplate = tpl
	o.Months = months
	return o
}

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

func (o Output) FromTemplate(tmpl string, vars map[string]interface{}) (string, error) {
	l := logger.Get()
	funcMap := template.FuncMap{
		"ToUpper":            strings.ToUpper,
		"ToLower":            strings.ToLower,
		"MonthIndex":         func(m string) string { return o.MonthIndex(m) },
		"NowYYYY":            func() string { return time.Now().Format("2006") },
		"NowYYYYMMDD":        func() string { return time.Now().Format("20060102") },
		"NowYYYYMMDD_HHMMSS": func() string { return time.Now().Format("20060102_030405") },
	}
	fulltemplate := tmpl
	if len(o.CommonTemplate) > 0 {
		fulltemplate = strings.Join([]string{o.CommonTemplate, tmpl}, "\n")
	}

	mytemplate := template.Must(template.New("main").Funcs(funcMap).Parse(fulltemplate))

	var doc bytes.Buffer
	err := mytemplate.Execute(&doc, vars)
	if err != nil {
		l.Fatal("Failed to execute template", zap.Error(err))
		return "", err
	}

	return doc.String(), nil
}

func main() {
	fmt.Println("vim-go")
}
