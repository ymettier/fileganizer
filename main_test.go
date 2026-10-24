// Copyright 2024-2026 The Fileganizer Authors. All rights reserved.
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

const invoiceOutput = "Invoice Summary\n  date: 2014-03-27\n  number: 001\n"

func TestInvoiceAndTextExtraction(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T)
		args       []string
		wantErr    bool
		wantSubstr string
	}{
		{
			name:       "file invoice",
			args:       []string{"-c", "testdata/config.invoice.yaml", "-f", "testdata/invoice.txt"},
			wantSubstr: invoiceOutput,
		},
		{
			name: "file invoice env",
			setup: func(t *testing.T) {
				t.Setenv("SOMEVAR", "magic")
			},
			args:       []string{"-c", "testdata/config.invoice-env.yaml", "-f", "testdata/invoice.txt"},
			wantSubstr: "Invoice magic Summary\n  date: 2014-03-27\n  number: 001\n",
		},
		{
			name:       "file non matching pattern",
			args:       []string{"-c", "testdata/config.invoice-nomatch.yaml", "-f", "testdata/invoice.txt"},
			wantSubstr: invoiceOutput,
		},
		{
			name:       "file broken template",
			args:       []string{"-c", "testdata/config.broken.template.yaml", "-f", "testdata/invoice.txt"},
			wantSubstr: invoiceOutput,
		},
		{
			name:       "file run mode",
			args:       []string{"-c", "testdata/config.invoice-run.yaml", "-f", "testdata/invoice.txt", "-r"},
			wantSubstr: "run mode works",
		},
		{
			name:       "file french months",
			args:       []string{"-c", "testdata/config.invoice-french.yaml", "-f", "testdata/invoice-french.txt"},
			wantSubstr: "08-27-2014",
		},
		{
			name:    "run missing config file",
			args:    []string{"-c", "testdata/nonexistent.yaml", "-f", "testdata/invoice.txt"},
			wantErr: true,
		},
		{
			name:    "run missing input file",
			args:    []string{"-c", "testdata/config.invoice.yaml", "-f", "testdata/nonexistent.txt"},
			wantErr: true,
		},
		{
			name:       "run text output flag",
			args:       []string{"-c", "testdata/config.invoice.yaml", "-f", "testdata/invoice.txt", "-t"},
			wantSubstr: "Invoice",
		},
		{
			name:    "run broken grok pattern",
			args:    []string{"-c", "testdata/config.broken.grok.yaml", "-f", "testdata/invoice.txt"},
			wantErr: true,
		},
		{
			name:    "file run mode fails",
			args:    []string{"-c", "testdata/config.broken.run.yaml", "-f", "testdata/invoice.txt", "-r"},
			wantErr: true,
		},
		{
			name:    "run broken grok pattern definition",
			args:    []string{"-c", "testdata/config.broken.regex.yaml", "-f", "testdata/invoice.txt"},
			wantErr: true,
		},
		{
			name:    "run version flag",
			args:    []string{"-V"},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			withArgs(t, tc.args...)
			output, err := run()
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.Nil(t, err)
			if tc.wantSubstr != "" {
				assert.Contains(t, output, tc.wantSubstr)
			}
		})
	}
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
	tests := []struct {
		name       string
		setup      func(*testing.T)
		config     string
		input      string
		wantSubstr string
	}{
		{
			name:       "pdf builtin extractor",
			config:     "testdata/config.pdfBuiltin.yaml",
			input:      "pdftotext/testdata/forged-invoice.pdf",
			wantSubstr: "Invoice INV-2024-001",
		},
		{
			name: "pdf builtin extractor env",
			setup: func(t *testing.T) {
				t.Setenv("COMPANY", "ACME Corp")
			},
			config:     "testdata/config.pdfBuiltinEnv.yaml",
			input:      "pdftotext/testdata/forged-invoice.pdf",
			wantSubstr: "Invoice INV-2024-001 from ACME Corp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			withArgs(t, "-c", tc.config, "-f", tc.input)
			output, err := run()
			assert.Nil(t, err)
			assert.Contains(t, output, tc.wantSubstr)
		})
	}
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

func TestExtractAndProcessEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T)
		runTest    func(t *testing.T)
		wantErr    bool
		wantSubstr string
		errSubstr  string
		emptyOut   bool
	}{
		{
			name: "mime not in config",
			setup: func(t *testing.T) {
				withArgs(t, "-c", "testdata/config.broken.mime.yaml", "-f", "testdata/invoice.txt")
			},
			runTest: func(t *testing.T) {
				_, err := run()
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "no textExtractor configured for MIME type")
			},
		},
		{
			name: "process file descriptions no match",
			setup: func(t *testing.T) {
				withArgs(t, "-c", "testdata/config.nomatch.yaml", "-f", "testdata/invoice.txt")
			},
			runTest: func(t *testing.T) {
				output, err := run()
				assert.Nil(t, err)
				assert.Empty(t, output)
			},
		},
		{
			name: "extract text unsupported type",
			runTest: func(t *testing.T) {
				cfg := config.Config{
					InputFile: "testdata/invoice.txt",
					TextExtractors: map[string]config.TextExtractorConfig{
						"text/plain": {Type: "bogus"},
					},
				}
				_, err := extractText(context.Background(), &cfg)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported textExtractor type")
			},
		},
		{
			name: "file invoice with cat command",
			setup: func(t *testing.T) {
				withArgs(t, "-c", "testdata/config.invoice-cat.yaml", "-f", "testdata/invoice.txt")
			},
			runTest: func(t *testing.T) {
				output, err := run()
				assert.Nil(t, err)
				assert.Contains(t, output, invoiceOutput)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			tc.runTest(t)
		})
	}
}
