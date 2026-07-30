// Copyright 2026 The Fileganizer Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pdftotext

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"fileganizer/logger"
)

const (
	tokName = 'n'
	tokStr  = 's'
	tokHex  = 'h'
	tokArr  = 'a'
	tokKw   = 'k'
	tokNum  = 'N'

	maxByte  = 255
	spaceCID = 32

	emScale       = 1000.0
	wordGapRatio  = 0.09
	lineGroupTol  = 2.0
	maxYDistLines = 4.0
)

type fontWidths [256]uint16

type pdfToken struct {
	kind byte
	raw  string
}

type positionedChar struct {
	x0, y0, x1, y1 float64
	text           string
}

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
	if s.pos == start {
		s.pos++
	}
	return pdfToken{kind: tokKw, raw: string(s.data[start:s.pos])}, true
}

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
			} else {
				logger.Get().Warn("invalid octal escape in PDF literal string, value exceeds byte range", "value", v, "octal", oct)
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return i
}

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
			cid := cidToUint16(srcCID)
			m[cid] = rune(cidToUint16(code))
		} else if len(srcCID) >= 2 && len(code) >= 1 {
			cid := cidToUint16(srcCID)
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
		lo := cidToUint16(startCID)
		hi := cidToUint16(endCID)
		var baseRune rune
		if len(val) >= 2 {
			baseRune = rune(cidToUint16(val))
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
		lo := cidToUint16(startCID)
		for i, val := range vals {
			cid := lo + uint16(i)
			if len(val) >= 2 {
				m[cid] = rune(cidToUint16(val))
			} else if len(val) >= 1 {
				m[cid] = rune(val[0])
			}
		}
	}
}

func cidToUint16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}

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
		b.WriteRune(rune(data[i]))
		i++
	}
	return b.String()
}

// multMatrices multiplies two 6-element PDF transformation matrices: a × b.
func multMatrices(a, b [6]float64) [6]float64 {
	return [6]float64{
		a[0]*b[0] + a[1]*b[2],
		a[0]*b[1] + a[1]*b[3],
		a[2]*b[0] + a[3]*b[2],
		a[2]*b[1] + a[3]*b[3],
		a[4]*b[0] + a[5]*b[2] + b[4],
		a[4]*b[1] + a[5]*b[3] + b[5],
	}
}

// applyTd translates the text matrix: Tm' = [1 0 0 1 tx ty] × Tm.
func applyTd(tm [6]float64, tx, ty float64) [6]float64 {
	return [6]float64{
		tm[0], tm[1], tm[2], tm[3],
		tx*tm[0] + ty*tm[2] + tm[4],
		tx*tm[1] + ty*tm[3] + tm[5],
	}
}

// textRenderPos returns the page-space position (from CTM × Tm).
func textRenderPos(ctm, tm [6]float64) (x, y float64) {
	m := multMatrices(ctm, tm)
	return m[4], m[5]
}

// charWidth returns the width of a glyph CID in text space units.
func charWidth(fw *fontWidths, cid byte, fontSize float64) float64 {
	if fw != nil && int(cid) < len(fw) && fw[cid] > 0 {
		return float64(fw[cid]) / emScale * fontSize
	}
	return fontSize * 0.5 //nolint:mnd
}

// groupCharsIntoLines groups positioned characters into lines by Y proximity.
func groupCharsIntoLines(chars []positionedChar, lineTol float64) [][]positionedChar {
	if len(chars) == 0 {
		return nil
	}

	sorted := make([]positionedChar, len(chars))
	copy(sorted, chars)
	sort.Slice(sorted, func(i, j int) bool {
		if math.Abs(sorted[i].y0-sorted[j].y0) > lineTol {
			return sorted[i].y0 > sorted[j].y0
		}
		return sorted[i].x0 < sorted[j].x0
	})

	var lines [][]positionedChar
	var cur []positionedChar
	curY := sorted[0].y0

	for _, c := range sorted {
		if math.Abs(c.y0-curY) > lineTol {
			lines = append(lines, cur)
			cur = nil
			curY = c.y0
		}
		cur = append(cur, c)
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}

	return lines
}

func renderLines(lines [][]positionedChar, wordRatio float64) string {
	var out strings.Builder
	for li, line := range lines {
		if li > 0 {
			out.WriteByte('\n')
		}
		sort.Slice(line, func(i, j int) bool {
			return line[i].x0 < line[j].x0
		})
		for ci, c := range line {
			if ci > 0 {
				gap := c.x0 - line[ci-1].x1
				charSize := c.y0 - c.y1
				if charSize < 0 {
					charSize = -charSize
				}
				if gap > charSize*wordRatio {
					out.WriteByte(' ')
				}
			}
			out.WriteString(c.text)
		}
	}
	return out.String()
}

// textFromContentStream parses a PDF content stream and extracts text with
// position tracking, then groups characters into lines geometrically.
func textFromContentStream( //nolint:gocyclo,funlen
	content []byte, fontCMaps map[string]map[uint16]rune, fWidths map[string]*fontWidths,
) string {
	s := &contentScanner{data: content}
	var stack []pdfToken
	var currentFont string
	var fontSize float64

	ctm := [6]float64{1, 0, 0, 1, 0, 0}
	var ctmStack [][6]float64
	tm := [6]float64{1, 0, 0, 1, 0, 0}
	var textLeading float64
	var cursorX float64

	var chars []positionedChar

	pushChar := func(text string, curX float64) {
		px, py := textRenderPos(ctm, tm)
		x0 := px + math.Min(cursorX, curX)
		x1 := px + math.Max(cursorX, curX)
		if x1-x0 < 0.1 { //nolint:mnd
			x1 = x0 + fontSize
		}
		chars = append(chars, positionedChar{
			x0:   x0,
			y0:   py,
			x1:   x1,
			y1:   py - fontSize,
			text: text,
		})
	}

	decodeAndRender := func(data []byte, cmap map[uint16]rune) {
		fw := fWidths[currentFont]
		for i := 0; i < len(data); {
			var cid byte
			var r rune
			var advance float64

			if cmap != nil && i+1 < len(data) {
				cidPair := uint16(data[i])<<8 | uint16(data[i+1])
				if r2, ok := cmap[cidPair]; ok {
					r = r2
					cid = data[i]
					advance = charWidth(fw, cid, fontSize)
					pushChar(string(r), cursorX+advance)
					cursorX += advance
					i += 2
					continue
				}
			}
			cid = data[i]
			advance = charWidth(fw, cid, fontSize)
			if cmap != nil {
				if r2, ok := cmap[uint16(cid)]; ok {
					r = r2
				} else {
					r = rune(cid)
				}
			} else {
				r = rune(cid)
			}
			pushChar(string(r), cursorX+advance)
			cursorX += advance
			i++
		}
	}

	renderTJArray := func(seq []pdfToken) {
		cmap := fontCMaps[currentFont]
		for _, el := range seq {
			switch el.kind {
			case tokStr:
				data := []byte(parseLiteralString(el.raw))
				decodeAndRender(data, cmap)
			case tokHex:
				data := parseHexString(el.raw)
				decodeAndRender(data, cmap)
			case tokNum:
				if val, err := strconv.ParseFloat(el.raw, 64); err == nil {
					cursorX -= val * 0.001 * fontSize
				}
			}
		}
	}

	for {
		tok, ok := s.next()
		if !ok {
			break
		}

		if tok.kind == tokArr && tok.raw == "[" {
			var seq []pdfToken
			for {
				el, ok := s.next()
				if !ok || (el.kind == tokArr && el.raw == "]") {
					break
				}
				seq = append(seq, el)
			}
			renderTJArray(seq)
			continue
		}

		if tok.kind == tokKw {
			switch tok.raw {
			case "q":
				var cpy [6]float64
				copy(cpy[:], ctm[:])
				ctmStack = append(ctmStack, cpy)

			case "Q":
				if len(ctmStack) > 0 {
					ctm = ctmStack[len(ctmStack)-1]
					ctmStack = ctmStack[:len(ctmStack)-1]
				}

			case "cm":
				if len(stack) >= 6 { //nolint:mnd
					f := stack[len(stack)-1]
					e := stack[len(stack)-2]
					d := stack[len(stack)-3]
					c := stack[len(stack)-4]
					b := stack[len(stack)-5]
					a := stack[len(stack)-6]
					if a.kind == tokNum && b.kind == tokNum && c.kind == tokNum &&
						d.kind == tokNum && e.kind == tokNum && f.kind == tokNum {
						av, _ := strconv.ParseFloat(a.raw, 64)
						bv, _ := strconv.ParseFloat(b.raw, 64)
						cv, _ := strconv.ParseFloat(c.raw, 64)
						dv, _ := strconv.ParseFloat(d.raw, 64)
						ev, _ := strconv.ParseFloat(e.raw, 64)
						fv, _ := strconv.ParseFloat(f.raw, 64)
						ctm = multMatrices([6]float64{av, bv, cv, dv, ev, fv}, ctm)
					}
				}

			case "BT":
				tm = [6]float64{1, 0, 0, 1, 0, 0}
				cursorX = 0

			case "Tf":
				if len(stack) >= 2 && stack[len(stack)-2].kind == tokName {
					currentFont = strings.TrimPrefix(stack[len(stack)-2].raw, "/")
				}
				if len(stack) >= 1 && stack[len(stack)-1].kind == tokNum {
					fs, err := strconv.ParseFloat(stack[len(stack)-1].raw, 64)
					if err == nil {
						fontSize = fs
					}
				}

			case "Td", "TD":
				if len(stack) >= 2 {
					ty := stack[len(stack)-1]
					tx := stack[len(stack)-2]
					if tx.kind == tokNum && ty.kind == tokNum {
						txVal, _ := strconv.ParseFloat(tx.raw, 64)
						tyVal, _ := strconv.ParseFloat(ty.raw, 64)
						tm = applyTd(tm, txVal, tyVal)
						cursorX = 0
					}
				}
				if tok.raw == "TD" && len(stack) >= 1 {
					ty := stack[len(stack)-1]
					if ty.kind == tokNum {
						tyVal, _ := strconv.ParseFloat(ty.raw, 64)
						textLeading = -tyVal
					}
				}

			case "Tm":
				if len(stack) >= 6 { //nolint:mnd
					f := stack[len(stack)-1]
					e := stack[len(stack)-2]
					d := stack[len(stack)-3]
					c := stack[len(stack)-4]
					b := stack[len(stack)-5]
					a := stack[len(stack)-6]
					if a.kind == tokNum && b.kind == tokNum && c.kind == tokNum &&
						d.kind == tokNum && e.kind == tokNum && f.kind == tokNum {
						av, _ := strconv.ParseFloat(a.raw, 64)
						bv, _ := strconv.ParseFloat(b.raw, 64)
						cv, _ := strconv.ParseFloat(c.raw, 64)
						dv, _ := strconv.ParseFloat(d.raw, 64)
						ev, _ := strconv.ParseFloat(e.raw, 64)
						fv, _ := strconv.ParseFloat(f.raw, 64)
						tm = [6]float64{av, bv, cv, dv, ev, fv}
						cursorX = 0
					}
				}

			case "T*":
				tm = applyTd(tm, 0, -textLeading)
				cursorX = 0

			case "'":
				tm = applyTd(tm, 0, -textLeading)
				cursorX = 0
				if len(stack) >= 1 {
					last := stack[len(stack)-1]
					switch last.kind {
					case tokStr:
						data := []byte(parseLiteralString(last.raw))
						decodeAndRender(data, fontCMaps[currentFont])
					case tokHex:
						data := parseHexString(last.raw)
						decodeAndRender(data, fontCMaps[currentFont])
					}
				}

			case "\"":
				if len(stack) >= 3 {
					ac := stack[len(stack)-2]
					if ac.kind == tokNum {
						_, _ = strconv.ParseFloat(ac.raw, 64)
					}
				}
				if len(stack) >= 2 {
					aw := stack[len(stack)-3]
					if aw.kind == tokNum {
						awVal, _ := strconv.ParseFloat(aw.raw, 64)
						_ = awVal
					}
				}
				tm = applyTd(tm, 0, -textLeading)
				cursorX = 0
				if len(stack) >= 1 {
					last := stack[len(stack)-1]
					switch last.kind {
					case tokStr:
						data := []byte(parseLiteralString(last.raw))
						decodeAndRender(data, fontCMaps[currentFont])
					case tokHex:
						data := parseHexString(last.raw)
						decodeAndRender(data, fontCMaps[currentFont])
					}
				}

			case "Tj":
				if len(stack) >= 1 {
					last := stack[len(stack)-1]
					switch last.kind {
					case tokStr:
						data := []byte(parseLiteralString(last.raw))
						decodeAndRender(data, fontCMaps[currentFont])
					case tokHex:
						data := parseHexString(last.raw)
						decodeAndRender(data, fontCMaps[currentFont])
					}
				}

			case "TJ":
				// already handled via array path above

			case "TL":
				if len(stack) >= 1 && stack[len(stack)-1].kind == tokNum {
					lv, _ := strconv.ParseFloat(stack[len(stack)-1].raw, 64)
					textLeading = -lv
				}

			case "Tc":
				// character spacing - not used for layout
			case "Tw":
				// word spacing - not used for layout
			case "Tz":
				// horizontal scaling - not used for layout
			case "Ts":
				// text rise - not used for layout
			case "Tr":
				// text rendering mode - not used for layout
			}

			stack = stack[:0]
		} else {
			stack = append(stack, tok)
		}
	}

	lines := groupCharsIntoLines(chars, lineGroupTol)
	return renderLines(lines, wordGapRatio)
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
		fWidths := buildFontWidths(pdfCtx, pageNr)

		pageText := textFromContentStream(data, fontCMaps, fWidths)
		if pageText != "" {
			if text.Len() > 0 {
				text.WriteByte('\n')
			}
			text.WriteString(pageText)
		}
	}

	return text.String(), nil
}

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

func buildFontWidths(ctx *model.Context, pageNr int) map[string]*fontWidths {
	result := make(map[string]*fontWidths)

	if pageNr < 1 || pageNr > len(ctx.Optimize.PageFonts) {
		return result
	}

	pageFonts := ctx.Optimize.PageFonts[pageNr-1]

	for objNr := range pageFonts {
		fo, ok := ctx.Optimize.FontObjects[objNr]
		if !ok {
			continue
		}

		fw := fontWidthsFromDict(fo.FontDict)
		if fw == nil {
			continue
		}

		for _, resName := range fo.ResourceNames {
			result[resName] = fw
		}
	}

	return result
}

func fontWidthsFromDict(fd types.Dict) *fontWidths {
	w, found := fd.Find("Widths")
	if !found || w == nil {
		return stdFontWidthsFromBaseFont(fd)
	}
	arr, ok := w.(types.Array)
	if !ok || len(arr) == 0 {
		return nil
	}

	firstChar := 0
	if v, found := fd.Find("FirstChar"); found {
		if i, ok := v.(types.Integer); ok {
			firstChar = i.Value()
		}
	}

	var fw fontWidths
	for i, v := range arr {
		code := firstChar + i
		if code > maxByte {
			break
		}
		if code < 0 {
			continue
		}
		if iv, ok := v.(types.Integer); ok {
			val := iv.Value()
			if val >= 0 && val <= 65535 {
				fw[code] = uint16(val)
			}
		}
	}

	return &fw
}

func stdFontWidthsFromBaseFont(fd types.Dict) *fontWidths {
	bf, found := fd.Find("BaseFont")
	if !found || bf == nil {
		return nil
	}
	name, ok := bf.(types.Name)
	if !ok {
		return nil
	}
	baseName := string(name)
	fw, ok := stdFontWidths[baseName]
	if !ok {
		return nil
	}
	cp := *fw
	return &cp
}

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
