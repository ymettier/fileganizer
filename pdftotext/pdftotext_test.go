// Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pdftotext

import (
	"context"
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fileganizer/testutil"
)

const (
	testdataDir         = "testdata"
	testDescendantFonts = "DescendantFonts"
)

func TestPDFTextExtract(t *testing.T) {
	t.Run("bsb-001 (Singapore bank statement)", func(t *testing.T) {
		output, err := PDFTextExtract(context.Background(), testdataDir+"/bsb-001-statement.pdf")
		require.NoError(t, err)

		assert.Contains(t, output, "Straits Capital")
		assert.Contains(t, output, "Xin Yi Tan")
		assert.Contains(t, output, "15,336.33")
		assert.Contains(t, output, "SC Savings Account")
		assert.Contains(t, output, "30/06/2025")
	})

	t.Run("bsb-002 (US credit card statement)", func(t *testing.T) {
		output, err := PDFTextExtract(context.Background(), testdataDir+"/bsb-002-statement.pdf")
		require.NoError(t, err)

		assert.Contains(t, output, "Liberty National Bank")
		assert.Contains(t, output, "Robert Wilson")
		assert.Contains(t, output, "3,565.64")
		assert.Contains(t, output, "LNB")
		assert.Contains(t, output, "07/24/2025")
	})

	t.Run("bsb-003 (Netherlands bank statement)", func(t *testing.T) {
		output, err := PDFTextExtract(context.Background(), testdataDir+"/bsb-003-statement.pdf")
		require.NoError(t, err)

		assert.Contains(t, output, "Continental Trust")
		assert.Contains(t, output, "Sanne Mulders")
		assert.Contains(t, output, "Rekeningafschrift")
		assert.Contains(t, output, "31.10.2025")
	})

	t.Run("bsb-004 (Hong Kong bank statement)", func(t *testing.T) {
		output, err := PDFTextExtract(context.Background(), testdataDir+"/bsb-004-statement.pdf")
		require.NoError(t, err)

		assert.Contains(t, output, "Silk Road Banking")
		assert.Contains(t, output, "Mei Ling Tsang")
		assert.Contains(t, output, "37,502.81")
		assert.Contains(t, output, "31/07/2025")
	})

	t.Run("bsb-005 (Canadian French bank statement)", func(t *testing.T) {
		output, err := PDFTextExtract(context.Background(), testdataDir+"/bsb-005-statement.pdf")
		require.NoError(t, err)

		assert.Contains(t, output, "Harbour Bank") //nolint:misspell // Canadian spelling
		assert.Contains(t, output, "Genevieve Cote")
		assert.Contains(t, output, "10 426,76")
		assert.Contains(t, output, "30 avril 2025")
	})
}

func TestPDFTextExtractFileNotFound(t *testing.T) {
	_, err := PDFTextExtract(context.Background(), "nonexistent.pdf")
	assert.Error(t, err)
}

func TestPDFTextExtractBadFile(t *testing.T) {
	testutil.UseTempDir(t)
	err := os.WriteFile("bad_test.pdf", []byte("not a pdf"), 0600)
	require.NoError(t, err)

	_, err = PDFTextExtract(context.Background(), "bad_test.pdf")
	assert.Error(t, err)
}

func TestTextFromContentStream(t *testing.T) {
	t.Run("literal Tj", func(t *testing.T) {
		content := []byte("BT\n/F1 12 Tf\n100 700 Td\n(Hello World) Tj\nET\n")
		text := textFromContentStream(content, nil)
		assert.Contains(t, text, "Hello World")
	})

	t.Run("hex Tj", func(t *testing.T) {
		content := []byte("<48656C6C6F> Tj")
		text := textFromContentStream(content, nil)
		assert.Contains(t, text, "Hello")
	})

	t.Run("TJ array", func(t *testing.T) {
		content := []byte("[(Hello)(World)]TJ")
		text := textFromContentStream(content, nil)
		assert.Contains(t, text, "Hello")
		assert.Contains(t, text, "World")
	})

	t.Run("empty", func(t *testing.T) {
		text := textFromContentStream(nil, nil)
		assert.Equal(t, "", text)
	})

	t.Run("single quote literal", func(t *testing.T) {
		content := []byte("(Hello World)'")
		text := textFromContentStream(content, nil)
		assert.Contains(t, text, "Hello World")
	})

	t.Run("single quote hex", func(t *testing.T) {
		content := []byte("<48656C6C6F>'")
		text := textFromContentStream(content, nil)
		assert.Contains(t, text, "Hello")
	})
}

func TestToUnicodeMap(t *testing.T) {
	cmap := `
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
2 beginbfchar
<0001> <0049>
<0002> <004E>
endbfchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`
	m := toUnicodeMap([]byte(cmap))
	assert.Equal(t, rune('I'), m[0x0001])
	assert.Equal(t, rune('N'), m[0x0002])
}

func TestToUnicodeMap_Bfrange(t *testing.T) {
	cmap := `
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfrange
<0001> <0003> [<0049> <004E> <0056>]
endbfrange
endcmap
end
end
`
	m := toUnicodeMap([]byte(cmap))
	assert.Equal(t, rune('I'), m[0x0001])
	assert.Equal(t, rune('N'), m[0x0002])
	assert.Equal(t, rune('V'), m[0x0003])
}

func TestDecodeText(t *testing.T) {
	cmap := map[uint16]rune{
		0x0001: 'I',
		0x0002: 'N',
		0x0003: 'V',
	}

	result := decodeText([]byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03}, cmap)
	assert.Equal(t, "INV", result)

	result = decodeText([]byte{0x48, 0x65, 0x6C, 0x6C, 0x6F}, nil)
	assert.Equal(t, "Hello", result)
}

func TestParseLiteralString(t *testing.T) {
	assert.Equal(t, "Hello", parseLiteralString("(Hello)"))
	assert.Equal(t, "Hello\nWorld", parseLiteralString("(Hello\\nWorld)"))
	assert.Equal(t, "Hello\\World", parseLiteralString("(Hello\\\\World)"))
	assert.Equal(t, "Hello)World", parseLiteralString("(Hello\\)World)"))
}

func TestParseHexString(t *testing.T) {
	result := parseHexString("<48656C6C6F>")
	assert.Equal(t, []byte("Hello"), result)

	result = parseHexString("<0049>")
	assert.Equal(t, []byte{0x00, 0x49}, result)
}

func TestSkipWS_Comment(t *testing.T) {
	s := &contentScanner{data: []byte("% this is a comment\nHello")}
	s.skipWS()
	assert.Equal(t, len("% this is a comment\n"), s.pos)
}

func TestNext_Operators(t *testing.T) {
	t.Run("gt", func(t *testing.T) {
		s := &contentScanner{data: []byte(">")}
		tok, ok := s.next()
		assert.True(t, ok)
		assert.Equal(t, byte(tokKw), tok.kind)
		assert.Equal(t, ">", tok.raw)
	})

	t.Run("brace", func(t *testing.T) {
		s := &contentScanner{data: []byte("{")}
		tok, ok := s.next()
		assert.True(t, ok)
		assert.Equal(t, byte(tokKw), tok.kind)
		assert.Equal(t, "{", tok.raw)
	})

	t.Run("escaped char in literal string", func(t *testing.T) {
		content := []byte("(Hello\\nWorld) Tj")
		text := textFromContentStream(content, nil)
		assert.Equal(t, "Hello\nWorld", text)
	})
}

func TestParseLiteralString_EdgeCases(t *testing.T) {
	t.Run("short", func(t *testing.T) {
		assert.Equal(t, "", parseLiteralString(""))
		assert.Equal(t, "a", parseLiteralString("a"))
	})

	t.Run("carriage return", func(t *testing.T) {
		assert.Equal(t, "\r", parseLiteralString("(\\r)"))
	})

	t.Run("tab", func(t *testing.T) {
		assert.Equal(t, "\t", parseLiteralString("(\\t)"))
	})

	t.Run("octal escape", func(t *testing.T) {
		assert.Equal(t, "\101", parseLiteralString("(\\101)"))
		assert.Equal(t, "\377", parseLiteralString("(\\377)"))
	})

	t.Run("octal escape overflow", func(t *testing.T) {
		assert.Equal(t, "", parseLiteralString("(\\777)"))
	})

	t.Run("default escape", func(t *testing.T) {
		assert.Equal(t, "x", parseLiteralString("(\\x)"))
	})
}

func TestParseHexString_EdgeCases(t *testing.T) {
	t.Run("short", func(t *testing.T) {
		assert.Nil(t, parseHexString(""))
		assert.Nil(t, parseHexString("<"))
	})

	t.Run("odd length", func(t *testing.T) {
		result := parseHexString("<48656C6C6>")
		assert.Equal(t, []byte{0x48, 0x65, 0x6C, 0x6C, 0x60}, result)
	})

	t.Run("invalid hex", func(t *testing.T) {
		assert.Nil(t, parseHexString("<ZZ>"))
	})
}

func TestToUnicodeMap_BfcharStringCode(t *testing.T) {
	cmap := `
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfchar
<0001> (I)
endbfchar
endcmap
end
end
`
	m := toUnicodeMap([]byte(cmap))
	assert.Equal(t, rune('I'), m[0x0001])
}

func TestToUnicodeMap_BfrangeContinuous(t *testing.T) {
	cmap := `
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfrange
<0001> <0003> <0041>
endbfrange
endcmap
end
end
`
	m := toUnicodeMap([]byte(cmap))
	assert.Equal(t, rune('A'), m[0x0001])
	assert.Equal(t, rune('B'), m[0x0002])
	assert.Equal(t, rune('C'), m[0x0003])
}

func TestDecodeText_SingleByteCID(t *testing.T) {
	cmap := map[uint16]rune{
		0x0041: 'A',
	}
	result := decodeText([]byte{0x41, 0x42, 0x43}, cmap)
	assert.Equal(t, "ABC", result)
}

func TestNext_NestedParen(t *testing.T) {
	s := &contentScanner{data: []byte("((Hello) World) Tj")}
	tok, ok := s.next()
	assert.True(t, ok)
	assert.Equal(t, byte(tokStr), tok.kind)
}

func TestToUnicodeMap_EdgeCases(t *testing.T) {
	t.Run("non-hex in bfchar", func(t *testing.T) {
		cmap := `
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
2 beginbfchar
/F1
<0001> <0041>
endbfchar
endcmap
`
		m := toUnicodeMap([]byte(cmap))
		assert.Equal(t, rune('A'), m[0x0001])
	})

	t.Run("bfchar eof after src", func(t *testing.T) {
		cmap := `
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfchar
<0001>
`
		m := toUnicodeMap([]byte(cmap))
		assert.Empty(t, m)
	})

	t.Run("non-hex in bfrange", func(t *testing.T) {
		cmap := `
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
2 beginbfrange
/F1
<0001> <0003> <0041>
endbfrange
endcmap
`
		m := toUnicodeMap([]byte(cmap))
		assert.Equal(t, rune('A'), m[0x0001])
	})

	t.Run("bfrange end cid not hex", func(t *testing.T) {
		cmap := `
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfrange
<0001> /F1 <0041>
endbfrange
endcmap
`
		m := toUnicodeMap([]byte(cmap))
		assert.Empty(t, m)
	})

	t.Run("bfrange eof after end cid", func(t *testing.T) {
		cmap := `
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfrange
<0001> <0003>
`
		m := toUnicodeMap([]byte(cmap))
		assert.Empty(t, m)
	})

	t.Run("bfrange non-array non-hex values", func(t *testing.T) {
		cmap := `
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfrange
<0001> <0003> /Something
endbfrange
endcmap
`
		m := toUnicodeMap([]byte(cmap))
		assert.Empty(t, m)
	})

	t.Run("bfrange list single-byte values", func(t *testing.T) {
		cmap := `
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfrange
<0001> <0003> [<41> <42> <43>]
endbfrange
endcmap
`
		m := toUnicodeMap([]byte(cmap))
		assert.Equal(t, rune('A'), m[0x0001])
		assert.Equal(t, rune('B'), m[0x0002])
		assert.Equal(t, rune('C'), m[0x0003])
	})
}

func TestToUnicodeMap_BfrangeContinuousSingleByte(t *testing.T) {
	cmap := `
/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfrange
<0001> <0003> <41>
endbfrange
endcmap
end
end
`
	m := toUnicodeMap([]byte(cmap))
	assert.Equal(t, rune('A'), m[0x0001])
	assert.Equal(t, rune('B'), m[0x0002])
	assert.Equal(t, rune('C'), m[0x0003])
}

func TestBuildFontCMaps_MissingFontObject(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	pdfCtx.Optimize.PageFonts[0][9999] = true
	m := buildFontCMaps(pdfCtx, 1)
	assert.NotNil(t, m)
}

func TestResolveToUnicode_NonIndirectRef(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	m := resolveToUnicode(pdfCtx, types.Integer(0))
	assert.Nil(t, m)
}

func TestCidToUnicode_DescendantFontsMissing(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	// forged-invoice.pdf uses F1/F2 simple fonts - no ToUnicode and no DescendantFonts
	// So cidToUnicode should return nil
	for _, fo := range pdfCtx.Optimize.FontObjects {
		m := cidToUnicode(pdfCtx, fo.FontDict)
		assert.Nil(t, m)
	}
}

func TestCidToUnicode_DescendantFontsBogusRef(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	fd := types.Dict{
		testDescendantFonts: types.Array{
			types.IndirectRef{ObjectNumber: 99999, GenerationNumber: 0},
		},
	}
	m := cidToUnicode(pdfCtx, fd)
	assert.Nil(t, m)
}

func TestCidToUnicode_DescendantFontsEmptyArray(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	fd := types.Dict{
		testDescendantFonts: types.Array{},
	}
	m := cidToUnicode(pdfCtx, fd)
	assert.Nil(t, m)
}

func TestCidToUnicode_DescendantFontsNonIndirectRef(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	fd := types.Dict{
		testDescendantFonts: types.Array{
			types.Integer(0),
		},
	}
	m := cidToUnicode(pdfCtx, fd)
	assert.Nil(t, m)
}

func TestResolveToUnicode_DerefError(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	m := resolveToUnicode(pdfCtx, types.IndirectRef{ObjectNumber: 99999, GenerationNumber: 0})
	assert.Nil(t, m)
}

func TestResolveToUnicode_ToNonStreamDict(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	// Point ToUnicode to object 3 (a font dict, not a stream)
	m := resolveToUnicode(pdfCtx, types.IndirectRef{ObjectNumber: 3, GenerationNumber: 0})
	assert.Nil(t, m)
}

func TestPDFTextExtract_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := PDFTextExtract(ctx, testdataDir+"/forged-invoice.pdf")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "canceled")
}

func TestBuildFontCMaps_InvalidPage(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/forged-invoice.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	t.Run("page 0", func(t *testing.T) {
		m := buildFontCMaps(pdfCtx, 0)
		assert.Empty(t, m)
	})

	t.Run("page > max", func(t *testing.T) {
		m := buildFontCMaps(pdfCtx, 9999)
		assert.Empty(t, m)
	})
}

func TestCidToUnicode_DescendantWithToUnicode(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/bsb-002-statement.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	t.Run("descendant without ToUnicode returns nil", func(t *testing.T) {
		for _, fo := range pdfCtx.Optimize.FontObjects {
			fd := make(types.Dict)
			for k, v := range fo.FontDict {
				fd[k] = v
			}
			delete(fd, "ToUnicode")

			m := cidToUnicode(pdfCtx, fd)
			assert.Nil(t, m)
		}
	})

	t.Run("descendant with ToUnicode returns cmap", func(t *testing.T) {
		// Font[11] has DescendantFonts → obj 34, ToUnicode → obj 35.
		// Add ToUnicode to the descendant dict (obj 34) and remove from font.
		dfEntry, found := pdfCtx.Table[34]
		require.True(t, found)

		df, ok := dfEntry.Object.(types.Dict)
		require.True(t, ok)

		df["ToUnicode"] = types.IndirectRef{ObjectNumber: 35, GenerationNumber: 0}

		fo := pdfCtx.Optimize.FontObjects[11]
		fd := make(types.Dict)
		for k, v := range fo.FontDict {
			fd[k] = v
		}
		delete(fd, "ToUnicode")

		m := cidToUnicode(pdfCtx, fd)
		assert.NotNil(t, m)
	})
}

func TestResolveToUnicode_DecodeCorruptedStream(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	f, err := os.Open(testdataDir + "/bsb-002-statement.pdf")
	require.NoError(t, err)
	defer f.Close()
	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	require.NoError(t, err)

	// Object 35 is the ToUnicode stream for font 11
	entry, found := pdfCtx.Table[35]
	require.True(t, found)

	entry.Object = types.StreamDict{
		Dict:    types.Dict{},
		Content: nil,
		Raw:     []byte{0xFF, 0xFE, 0xFD},
		FilterPipeline: []types.PDFFilter{
			{Name: "FlateDecode"},
		},
	}

	m := resolveToUnicode(pdfCtx, types.IndirectRef{ObjectNumber: 35, GenerationNumber: 0})
	assert.Nil(t, m)
}

func TestTextFromContentStream_WriteTextEdgeCases(t *testing.T) {
	t.Run("empty stack", func(t *testing.T) {
		content := []byte("Tj")
		text := textFromContentStream(content, nil)
		assert.Empty(t, text)
	})

	t.Run("non_string_token", func(t *testing.T) {
		content := []byte("/Name Tj")
		text := textFromContentStream(content, nil)
		assert.Empty(t, text)
	})
}

func TestTextFromContentStream_DQuote(t *testing.T) {
	t.Run("literal string", func(t *testing.T) {
		content := []byte("(Hello World)\"")
		text := textFromContentStream(content, nil)
		assert.Contains(t, text, "Hello World")
	})

	t.Run("hex string", func(t *testing.T) {
		content := []byte("<48656C6C6F>\"")
		text := textFromContentStream(content, nil)
		assert.Contains(t, text, "Hello")
	})
}
