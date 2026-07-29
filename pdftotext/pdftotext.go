// Copyright 2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pdftotext

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"fileganizer/logger"
)

// pdfToken kind constants.
const (
	tokName = 'n'
	tokStr  = 's'
	tokHex  = 'h'
	tokArr  = 'a'
	tokKw   = 'k'
	tokNum  = 'N'
)

// maxByte is the maximum value an uint8 can hold, used for bounds checking
// when converting parsed octal escape values from PDF literal strings.
const maxByte = 255

type pdfToken struct {
	kind byte
	raw  string
}

// contentScanner tokenizes a PDF content stream.
type contentScanner struct {
	data []byte
	pos  int
}

func (s *contentScanner) skipWS() {
	for s.pos < len(s.data) {
		b := s.data[s.pos]
		if b == '%' {
			for s.pos < len(s.data) && s.data[s.pos] != '\n' && s.data[s.pos] != '\r' {
				s.pos++
			}
			continue
		}
		if b == 0 || b == 9 || b == 10 || b == 12 || b == 13 || b == 32 {
			s.pos++
			continue
		}
		break
	}
}

func (s *contentScanner) next() (pdfToken, bool) {
	s.skipWS()
	if s.pos >= len(s.data) {
		return pdfToken{}, false
	}

	b := s.data[s.pos]

	switch {
	case b == '/':
		return s.readName()
	case b == '(':
		return s.readString()
	case b == '<':
		return s.readAngle()
	case b == '>':
		return s.readCloseAngle()
	case b == '[':
		s.pos++
		return pdfToken{kind: tokArr, raw: "["}, true
	case b == ']':
		s.pos++
		return pdfToken{kind: tokArr, raw: "]"}, true
	case b == '{' || b == '}':
		s.pos++
		return pdfToken{kind: tokKw, raw: string(b)}, true
	case b == '-' || b == '+' || b == '.' || (b >= '0' && b <= '9'):
		return s.readNumber()
	default:
		return s.readKeyword()
	}
}

func (s *contentScanner) readName() (pdfToken, bool) {
	start := s.pos
	s.pos++
	for s.pos < len(s.data) {
		c := s.data[s.pos]
		if c <= 32 || c == '/' || c == '<' || c == '>' || c == '(' || c == ')' || c == '[' || c == ']' || c == '{' || c == '}' {
			break
		}
		s.pos++
	}
	return pdfToken{kind: tokName, raw: string(s.data[start:s.pos])}, true
}

func (s *contentScanner) readString() (pdfToken, bool) {
	depth := 1
	start := s.pos
	s.pos++
	for s.pos < len(s.data) && depth > 0 {
		if s.data[s.pos] == '\\' {
			s.pos += 2
			continue
		}
		switch s.data[s.pos] {
		case '(':
			depth++
		case ')':
			depth--
		}
		s.pos++
	}
	return pdfToken{kind: tokStr, raw: string(s.data[start:s.pos])}, true
}

func (s *contentScanner) readAngle() (pdfToken, bool) {
	if s.pos+1 < len(s.data) && s.data[s.pos+1] == '<' {
		s.pos += 2
		return pdfToken{kind: tokKw, raw: "<<"}, true
	}
	start := s.pos
	s.pos++
	for s.pos < len(s.data) && s.data[s.pos] != '>' {
		s.pos++
	}
	if s.pos < len(s.data) {
		s.pos++
	}
	return pdfToken{kind: tokHex, raw: string(s.data[start:s.pos])}, true
}

func (s *contentScanner) readCloseAngle() (pdfToken, bool) {
	if s.pos+1 < len(s.data) && s.data[s.pos+1] == '>' {
		s.pos += 2
		return pdfToken{kind: tokKw, raw: ">>"}, true
	}
	s.pos++
	return pdfToken{kind: tokKw, raw: ">"}, true
}

func (s *contentScanner) readNumber() (pdfToken, bool) {
	start := s.pos
	if b := s.data[s.pos]; b == '-' || b == '+' {
		s.pos++
	}
	for s.pos < len(s.data) {
		c := s.data[s.pos]
		if (c >= '0' && c <= '9') || c == '.' {
			s.pos++
		} else {
			break
		}
	}
	return pdfToken{kind: tokNum, raw: string(s.data[start:s.pos])}, true
}

func (s *contentScanner) readKeyword() (pdfToken, bool) {
	start := s.pos
	for s.pos < len(s.data) {
		c := s.data[s.pos]
		if c <= 32 || c == '/' || c == '<' || c == '>' || c == '(' || c == ')' || c == '[' || c == ']' || c == '{' || c == '}' {
			break
		}
		s.pos++
	}
	return pdfToken{kind: tokKw, raw: string(s.data[start:s.pos])}, true
}

// parseLiteralString unescapes a PDF literal string.
func parseLiteralString(s string) string {
	if len(s) < 2 {
		return s
	}
	s = s[1 : len(s)-1]
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i = writeEscaped(&b, s, i)
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// writeEscaped handles a backslash escape sequence at position i in s,
// writing the decoded byte to b. It returns the updated index.
func writeEscaped(b *strings.Builder, s string, i int) int {
	i++
	switch s[i] {
	case 'n':
		b.WriteByte('\n')
	case 'r':
		b.WriteByte('\r')
	case 't':
		b.WriteByte('\t')
	case '\\', '(', ')':
		b.WriteByte(s[i])
	default:
		if s[i] >= '0' && s[i] <= '7' {
			oct := string(s[i])
			for j := 0; j < 2 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '7'; j++ {
				i++
				oct += string(s[i])
			}
			v, _ := strconv.ParseUint(oct, 8, 32)
			if v <= maxByte {
				b.WriteByte(byte(v))
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return i
}

// parseHexString decodes a PDF hex string.
func parseHexString(s string) []byte {
	if len(s) < 2 {
		return nil
	}
	h := strings.TrimSpace(s[1 : len(s)-1])
	if len(h)%2 != 0 {
		h += "0"
	}
	dst := make([]byte, hex.DecodedLen(len(h)))
	n, err := hex.Decode(dst, []byte(h))
	if err != nil {
		return nil
	}
	return dst[:n]
}

// toUnicodeMap parses a ToUnicode CMap and returns a CID->rune mapping.
func toUnicodeMap(data []byte) map[uint16]rune {
	m := make(map[uint16]rune)
	s := &contentScanner{data: data}

	for {
		tok, ok := s.next()
		if !ok {
			break
		}

		switch tok.raw {
		case "beginbfchar":
			parseBFCharInto(s, m)
		case "beginbfrange":
			parseBFRangeInto(s, m)
		}
	}

	return m
}

func parseBFCharInto(s *contentScanner, m map[uint16]rune) {
	for {
		tok, ok := s.next()
		if !ok || tok.raw == "endbfchar" {
			break
		}
		if tok.kind != tokHex {
			continue
		}
		srcCID := parseHexString(tok.raw)
		tok, ok = s.next()
		if !ok {
			break
		}
		var code []byte
		switch tok.kind {
		case tokHex:
			code = parseHexString(tok.raw)
		case tokStr:
			code = []byte(parseLiteralString(tok.raw))
		}
		if len(srcCID) >= 2 && len(code) >= 2 {
			cid := uint16(srcCID[0])<<8 | uint16(srcCID[1])
			m[cid] = rune(uint16(code[0])<<8 | uint16(code[1]))
		} else if len(srcCID) >= 2 && len(code) >= 1 {
			cid := uint16(srcCID[0])<<8 | uint16(srcCID[1])
			m[cid] = rune(code[0])
		}
	}
}

func parseBFRangeInto(s *contentScanner, m map[uint16]rune) {
	for {
		tok, ok := s.next()
		if !ok || tok.raw == "endbfrange" {
			break
		}
		if tok.kind != tokHex {
			continue
		}
		startCID := parseHexString(tok.raw)

		tok, ok = s.next()
		if !ok || tok.kind != tokHex {
			break
		}
		endCID := parseHexString(tok.raw)

		tok, ok = s.next()
		if !ok {
			break
		}

		if tok.kind == tokHex {
			val := parseHexString(tok.raw)
			addBFRangeContinuous(m, startCID, endCID, val)
		} else if tok.raw == "[" {
			addBFRangeList(s, m, startCID)
		}
	}
}

func addBFRangeContinuous(m map[uint16]rune, startCID, endCID, val []byte) {
	if len(startCID) >= 2 && len(endCID) >= 2 && len(val) >= 1 {
		lo := uint16(startCID[0])<<8 | uint16(startCID[1])
		hi := uint16(endCID[0])<<8 | uint16(endCID[1])
		var baseRune rune
		if len(val) >= 2 {
			baseRune = rune(uint16(val[0])<<8 | uint16(val[1]))
		} else {
			baseRune = rune(val[0])
		}
		for cid := lo; cid <= hi; cid++ {
			m[cid] = baseRune + rune(cid-lo)
		}
	}
}

func addBFRangeList(s *contentScanner, m map[uint16]rune, startCID []byte) {
	var vals [][]byte
	for {
		tok, ok := s.next()
		if !ok || tok.raw == "]" {
			break
		}
		if tok.kind == tokHex {
			vals = append(vals, parseHexString(tok.raw))
		}
	}
	if len(startCID) >= 2 {
		lo := uint16(startCID[0])<<8 | uint16(startCID[1])
		for i, val := range vals {
			cid := lo + uint16(i)
			if len(val) >= 2 {
				m[cid] = rune(uint16(val[0])<<8 | uint16(val[1]))
			} else if len(val) >= 1 {
				m[cid] = rune(val[0])
			}
		}
	}
}

// decodeText decodes CID bytes using a CID→rune map.
func decodeText(data []byte, cmap map[uint16]rune) string {
	var b strings.Builder
	for i := 0; i < len(data); {
		if cmap != nil && i+1 < len(data) {
			cid := uint16(data[i])<<8 | uint16(data[i+1])
			if r, ok := cmap[cid]; ok {
				b.WriteRune(r)
				i += 2
				continue
			}
		}
		if cmap != nil {
			if r, ok := cmap[uint16(data[i])]; ok {
				b.WriteRune(r)
				i++
				continue
			}
		}
		b.WriteByte(data[i])
		i++
	}
	return b.String()
}

// textFromContentStream parses a PDF content stream and extracts text strings
// using the provided font ToUnicode maps (fontResourceName -> CID→rune).
func textFromContentStream(content []byte, fontCMaps map[string]map[uint16]rune) string {
	s := &contentScanner{data: content}
	var stack []pdfToken
	var currentFont string
	var out strings.Builder

	for {
		tok, ok := s.next()
		if !ok {
			break
		}

		if tok.kind == tokArr && tok.raw == "[" {
			out.WriteString(collectTextFromArray(s, fontCMaps, currentFont))
			continue
		}

		if tok.kind == tokKw {
			switch tok.raw {
			case "Tf":
				if len(stack) >= 2 && stack[len(stack)-2].kind == tokName {
					currentFont = strings.TrimPrefix(stack[len(stack)-2].raw, "/")
				}

			case "Td", "TD":
				if len(stack) >= 2 {
					last := stack[len(stack)-1]
					if last.kind == tokNum && last.raw != "0" {
						out.WriteByte('\n')
					}
				}

			case "Tj", "'", "\"":
				out.WriteString(writeTextFromStack(stack, fontCMaps, currentFont))

			case "TJ":
			}

			stack = stack[:0]
		} else {
			stack = append(stack, tok)
		}
	}

	return out.String()
}

func collectTextFromArray(s *contentScanner, fontCMaps map[string]map[uint16]rune, currentFont string) string {
	var texts []string
	for {
		el, ok := s.next()
		if !ok || (el.kind == tokArr && el.raw == "]") {
			break
		}
		switch el.kind {
		case tokStr:
			texts = append(texts, parseLiteralString(el.raw))
		case tokHex:
			cmap := fontCMaps[currentFont]
			texts = append(texts, decodeText(parseHexString(el.raw), cmap))
		}
	}
	return strings.Join(texts, "")
}

func writeTextFromStack(stack []pdfToken, fontCMaps map[string]map[uint16]rune, currentFont string) string {
	if len(stack) < 1 {
		return ""
	}
	last := stack[len(stack)-1]
	switch last.kind {
	case tokStr:
		return parseLiteralString(last.raw)
	case tokHex:
		cmap := fontCMaps[currentFont]
		return decodeText(parseHexString(last.raw), cmap)
	}
	return ""
}

// PDFTextExtract uses pdfcpu to extract text from a PDF file.
func PDFTextExtract(ctx context.Context, filename string) (string, error) {
	l := logger.Get()
	l.Debug("PDFTextExtract", "file", filename)

	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	conf := model.NewDefaultConfiguration()

	pdfCtx, err := api.ReadValidateAndOptimize(f, conf)
	if err != nil {
		return "", err
	}

	var text strings.Builder

	for pageNr := 1; pageNr <= pdfCtx.PageCount; pageNr++ {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("canceled on page %d: %w", pageNr, err)
		}

		r, _ := pdfcpu.ExtractPageContent(pdfCtx, pageNr)

		data, _ := io.ReadAll(r)

		fontCMaps := buildFontCMaps(pdfCtx, pageNr)

		pageText := textFromContentStream(data, fontCMaps)
		if pageText != "" {
			text.WriteString(pageText)
			text.WriteByte('\n')
		}
	}

	return text.String(), nil
}

// buildFontCMaps builds font resource name → CID→rune maps for a given page.
func buildFontCMaps(ctx *model.Context, pageNr int) map[string]map[uint16]rune {
	result := make(map[string]map[uint16]rune)

	if pageNr < 1 || pageNr > len(ctx.Optimize.PageFonts) {
		return result
	}

	pageFonts := ctx.Optimize.PageFonts[pageNr-1]

	for objNr := range pageFonts {
		fo, ok := ctx.Optimize.FontObjects[objNr]
		if !ok {
			continue
		}

		cmap := cidToUnicode(ctx, fo.FontDict)
		if len(cmap) == 0 {
			continue
		}

		for _, resName := range fo.ResourceNames {
			result[resName] = cmap
		}
	}

	return result
}

// cidToUnicode extracts a CID→rune map from a font dictionary by reading its
// ToUnicode CMap. For Type0 CIDFonts, it also checks DescendantFonts.
func cidToUnicode(ctx *model.Context, fd types.Dict) map[uint16]rune {
	toUnicode, found := fd.Find("ToUnicode")
	if found && toUnicode != nil {
		if cmap := resolveToUnicode(ctx, toUnicode); len(cmap) > 0 {
			return cmap
		}
	}

	df, found := fd.Find("DescendantFonts")
	if !found || df == nil {
		return nil
	}
	arr, ok := df.(types.Array)
	if !ok || len(arr) == 0 {
		return nil
	}
	ir, ok := arr[0].(types.IndirectRef)
	if !ok {
		return nil
	}
	cfd, err := ctx.DereferenceDict(ir)
	if err != nil || cfd == nil {
		return nil
	}

	tu, found := cfd.Find("ToUnicode")
	if !found || tu == nil {
		return nil
	}
	return resolveToUnicode(ctx, tu)
}

// resolveToUnicode resolves a ToUnicode reference and parses the CMap.
func resolveToUnicode(ctx *model.Context, obj types.Object) map[uint16]rune {
	ir, ok := obj.(types.IndirectRef)
	if !ok {
		return nil
	}

	sd, _, err := ctx.DereferenceStreamDict(ir)
	if err != nil || sd == nil {
		return nil
	}
	if err := sd.Decode(); err != nil {
		return nil
	}

	return toUnicodeMap(sd.Content)
}
