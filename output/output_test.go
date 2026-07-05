// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var months = map[string][]string{
	"French": {"Janvier", "Février", "Mars", "Avril", "Mai", "Juin", "Juillet", "Aout", "Septembre", "Octobre", "Novembre", "Décembre"},
}

var vars = map[string]any{
	"year":       "1970",
	"identifier": "123",
}

var templates = map[string]string{
	"year 1970 identifier 123": `year {{ .year }} identifier {{ .identifier }}`,
	"year 1970 lowercase":      `year {{ .year }} {{ ToLower "LoWerCaSe" }}`,
	"year 1970 UPPERCASE":      `year {{ .year }} {{ ToUpper "UppErCaSe" }}`,
	"year 1970 month 03":       `year {{ .year }} month {{ MonthIndex "Mars" }}`,
}

func TestFromTemplate(t *testing.T) {
	for wants, tpl := range templates {
		o := New(tpl, months)

		r, err := o.FromTemplate("", vars)
		assert.NoErrorf(t, err, "Fails on template '%s'", tpl)
		assert.Equalf(t, wants+"\n", r, "Fails on template '%s'", tpl)
	}
}

func TestFromTemplateWithCommonTemplate(t *testing.T) {
	o := New("year {{ .year }}", months)

	r, err := o.FromTemplate("{{ .identifier }}", vars)
	assert.NoError(t, err)
	assert.Equal(t, "year 1970\n123", r)
}

func TestFromTemplateWithBrokenTemplate(t *testing.T) {
	o := New("year {{ .year }}", months)

	_, err := o.FromTemplate("{{", vars)
	assert.Error(t, err)
}

func TestFromTemplate_ExecuteError(t *testing.T) {
	o := New("", nil)

	// ToUpper expects a string; passing an int causes a type error at execution
	_, err := o.FromTemplate("{{ ToUpper 42 }}", nil)
	assert.Error(t, err)
}

func TestMonthIndex_NoMatch(t *testing.T) {
	o := New("", nil)

	result := o.MonthIndex("NotAMonth")
	assert.Equal(t, "NotAMonth", result)
}

func TestMonthIndex_EmptyMonths(t *testing.T) {
	o := New("", map[string][]string{})

	result := o.MonthIndex("January")
	assert.Equal(t, "January", result)
}
