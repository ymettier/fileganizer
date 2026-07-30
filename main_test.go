// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fileganizer/config"
	"fileganizer/testutil"
)

func withArgs(t *testing.T, args ...string) {
	t.Helper()
	oldArgs := os.Args
	os.Args = append([]string{"./fileganizer"}, args...)
	t.Cleanup(func() { os.Args = oldArgs })
}

func TestFileInvoice(t *testing.T) {
	withArgs(t, "-c", "testdata/config.invoice.yaml", "-f", "testdata/invoice.txt")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice Summary\n  date: 2014-03-27\n  number: 001\n")
}

func TestFileInvoiceEnv(t *testing.T) {
	os.Setenv("SOMEVAR", "magic")
	defer os.Unsetenv("SOMEVAR")
	withArgs(t, "-c", "testdata/config.invoice-env.yaml", "-f", "testdata/invoice.txt")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice magic Summary\n  date: 2014-03-27\n  number: 001\n")
}

func TestBuiltinExtractUnsupportedMIME(t *testing.T) {
	withArgs(t, "-c", "testdata/config.broken.mime.yaml", "-f", "testdata/minimal.wav")

	_, err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no builtin textExtractor for MIME type")
}

func TestDetectFileType_ReadError(t *testing.T) {
	testutil.UseTempDir(t)

	// Using a directory causes f.Read to fail
	err := os.MkdirAll("unreadable", 0o755)
	require.NoError(t, err)

	_, err = detectFileType("unreadable")
	assert.Error(t, err)
}

func TestPDFBuiltinExtractor(t *testing.T) {
	withArgs(t, "-c", "testdata/config.pdfBuiltin.yaml", "-f", "pdftotext/testdata/forged-invoice.pdf")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice INV-2024-001")
}

func TestPDFBuiltinExtractorEnv(t *testing.T) {
	os.Setenv("COMPANY", "ACME Corp")
	defer os.Unsetenv("COMPANY")
	withArgs(t, "-c", "testdata/config.pdfBuiltinEnv.yaml", "-f", "pdftotext/testdata/forged-invoice.pdf")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice INV-2024-001 from ACME Corp")
}

func TestFileNonMatchingPattern(t *testing.T) {
	withArgs(t, "-c", "testdata/config.invoice-nomatch.yaml", "-f", "testdata/invoice.txt")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice Summary\n  date: 2014-03-27\n  number: 001\n")
	assert.NotContains(t, output, "should not appear")
}

func TestFileBrokenTemplate(t *testing.T) {
	withArgs(t, "-c", "testdata/config.broken.template.yaml", "-f", "testdata/invoice.txt")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice Summary\n  date: 2014-03-27\n  number: 001\n")
}

func TestFileRunMode(t *testing.T) {
	withArgs(t, "-c", "testdata/config.invoice-run.yaml", "-f", "testdata/invoice.txt", "-r")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "run mode works")
}

func TestFileFrenchMonths(t *testing.T) {
	withArgs(t, "-c", "testdata/config.invoice-french.yaml", "-f", "testdata/invoice-french.txt")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "08-27-2014")
}

func TestRunMissingConfigFile(t *testing.T) {
	withArgs(t, "-c", "testdata/nonexistent.yaml", "-f", "testdata/invoice.txt")

	_, err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent.yaml")
}

func TestRunMissingInputFile(t *testing.T) {
	withArgs(t, "-c", "testdata/config.invoice.yaml", "-f", "testdata/nonexistent.txt")

	_, err := run()
	assert.Error(t, err)
}

func TestRunTextOutputFlag(t *testing.T) {
	withArgs(t, "-c", "testdata/config.invoice.yaml", "-f", "testdata/invoice.txt", "-t")

	output, err := run()
	assert.NoError(t, err)
	assert.Contains(t, output, "Invoice")
}

func TestRunBrokenGrokPattern(t *testing.T) {
	withArgs(t, "-c", "testdata/config.broken.grok.yaml", "-f", "testdata/invoice.txt")

	_, err := run()
	assert.Error(t, err)
}

func TestFileRunModeFails(t *testing.T) {
	withArgs(t, "-c", "testdata/config.broken.run.yaml", "-f", "testdata/invoice.txt", "-r")

	_, err := run()
	assert.Error(t, err)
}

func TestRunBrokenGrokPatternDefinition(t *testing.T) {
	withArgs(t, "-c", "testdata/config.broken.regex.yaml", "-f", "testdata/invoice.txt")

	_, err := run()
	assert.Error(t, err)
}

func TestRunVersionFlag(t *testing.T) {
	withArgs(t, "-V")

	_, err := run()
	assert.NoError(t, err)
}

func TestBSBStatements(t *testing.T) {
	tests := []struct {
		name       string
		pdf        string
		wantOutput string
	}{
		{"bsb001", "pdftotext/testdata/bsb-001-statement.pdf", "20250630__straits_capital__xin_yi_tan\n"},
		{"bsb002", "pdftotext/testdata/bsb-002-statement.pdf", "20250630__liberty_national_bank__robert_wilson\n"},
		{"bsb003", "pdftotext/testdata/bsb-003-statement.pdf", "20251031__continental_trust__sanne_mulders\n"},
		{"bsb004", "pdftotext/testdata/bsb-004-statement.pdf", "20250731__silk_road_banking__mei_ling_tsang\n"},
		{"bsb005", "pdftotext/testdata/bsb-005-statement.pdf", "20250430__harbour_bank__genevieve_cote\n"}, //nolint:misspell
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withArgs(t, "-c", "testdata/config.bsb.yaml", "-f", tc.pdf)

			output, err := run()
			assert.Nil(t, err)
			assert.Equal(t, tc.wantOutput, output)
		})
	}
}

func TestExtractTextMimeNotInConfig(t *testing.T) {
	withArgs(t, "-c", "testdata/config.broken.mime.yaml", "-f", "testdata/invoice.txt")

	_, err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no textExtractor configured for MIME type")
}

func TestProcessFileDescriptionsNoMatch(t *testing.T) {
	withArgs(t, "-c", "testdata/config.nomatch.yaml", "-f", "testdata/invoice.txt")

	output, err := run()
	assert.Nil(t, err)
	assert.Empty(t, output)
}

func TestExtractTextUnsupportedType(t *testing.T) {
	cfg := config.Config{
		InputFile: "testdata/invoice.txt",
		TextExtractors: map[string]config.TextExtractorConfig{
			"text/plain": {Type: "bogus"},
		},
	}
	_, err := extractText(context.Background(), &cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported textExtractor type")
}

func TestFileInvoiceWithCatCommand(t *testing.T) {
	withArgs(t, "-c", "testdata/config.invoice-cat.yaml", "-f", "testdata/invoice.txt")

	output, err := run()
	assert.Nil(t, err)
	assert.Contains(t, output, "Invoice Summary\n  date: 2014-03-27\n  number: 001\n")
}
