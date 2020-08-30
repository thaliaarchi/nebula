// Copyright 2016 The Go Authors. All rights reserved.
// Copyright 2020 Andrew Archibald. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements scanner, a lexical tokenizer for
// Go source. After initialization, consecutive calls of
// next advance the scanner one token at a time.
//
// This file, source.go, tokens.go, and token_string.go are self-contained
// (`go tool compile scanner.go source.go tokens.go token_string.go` compiles)
// and thus could be made into their own package.

package syntax

import (
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

// The mode flags below control which comments are reported
// by calling the error handler. If no flag is set, comments
// are ignored.
const (
	emitComments uint = 1 << iota // emit all comments
	semiComment
)

type scanner struct {
	source
	mode   uint
	nlsemi bool // if set '\n' and EOF translate to ';'

	// current token, valid after calling next()
	line, col uint
	blank     bool // line is blank up to col
	tok       token
	literal   string
	bad       bool // valid if tok is a literal, true if a syntax error occurred, literal may be malformed
}

func (s *scanner) init(src io.Reader, errh func(line, col uint, msg string), mode uint) {
	s.source.init(src, errh)
	s.mode = mode
	s.nlsemi = false
}

// errorf reports an error at the most recently read character position.
func (s *scanner) errorf(format string, args ...interface{}) {
	s.error(fmt.Sprintf(format, args...))
}

// errorAtf reports an error at a byte column offset relative to the current token start.
func (s *scanner) errorAtf(offset int, format string, args ...interface{}) {
	s.errh(s.line, s.col+uint(offset), fmt.Sprintf(format, args...))
}

// setLiteral sets the scanner state for a recognized literal token.
func (s *scanner) setLiteral(kind token, ok bool) {
	s.tok = kind
	s.literal = string(s.segment())
	s.bad = !ok
}

// next advances the scanner by reading the next token.
//
// If a read, source encoding, or lexical error occurs, next calls the
// installed error handler with the respective error position and
// message. The error message is guaranteed to be non-empty. The error
// handler must exist. Comments are only emitted if the scanner mode
// includes the comments flag.
func (s *scanner) next() {
	nlsemi := s.nlsemi
	s.nlsemi = false

redo:
	// skip white space
	s.stop()
	startLine, startCol := s.pos()
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\n' && !nlsemi || s.ch == '\r' {
		s.nextch()
	}

	// token start
	s.line, s.col = s.pos()
	s.blank = s.line > startLine || startCol == colbase
	s.start()

	switch s.ch {
	case -1:
		if nlsemi {
			s.literal = "EOF"
			s.tok = Semi
			break
		}
		s.tok = EOF

	case '\n':
		s.nextch()
		s.literal = "newline"
		s.tok = Semi

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		s.number(false)

	case '"':
		s.standardString()

	case '`':
		s.rawString()

	case '\'':
		s.rune()

	case ';':
		s.nextch()
		if s.mode&semiComment == 0 {
			s.literal = "semicolon"
			s.tok = Semi
			break
		}
		if !s.lineComment() {
			goto redo
		}

	case ':':
		s.nextch()
		s.tok = Colon

	case '.':
		s.nextch()
		if isDecimal(s.ch) {
			s.number(true)
			break
		}
		if s.atIdentChar(true) {
			s.nextch()
			s.ident()
			break
		}
		s.setLiteral(Ident, true)

	case '#':
		s.nextch()
		if !s.lineComment() {
			goto redo
		}

	case '/':
		s.nextch()
		if s.ch == '/' {
			s.nextch()
			if !s.lineComment() {
				goto redo
			}
		}
		if s.ch == '*' {
			s.nextch()
			keep := s.blockComment()
			if line, _ := s.pos(); line > s.line && nlsemi {
				// A multi-line comment acts like a newline;
				// it translates to a ';' if nlsemi is set.
				s.literal = "newline"
				s.tok = Semi
				break
			}
			if !keep {
				goto redo
			}
		}

	default:
		if s.atIdentChar(true) {
			s.nextch()
			s.ident()
			break
		}

		s.errorf("invalid character %#U", s.ch)
		s.nextch()
		goto redo
	}

	return
}

func (s *scanner) ident() {
	for s.atIdentChar(false) {
		s.nextch()
	}
	s.setLiteral(Ident, true)
}

func (s *scanner) atIdentChar(first bool) bool {
	switch {
	case unicode.IsLetter(s.ch) || s.ch == '_' || s.ch == '.' || s.ch == '~':
		// ok
	case unicode.IsDigit(s.ch):
		if first {
			s.errorf("identifier cannot begin with digit %#U", s.ch)
		}
	case s.ch >= utf8.RuneSelf:
		s.errorf("invalid character %#U in identifier", s.ch)
	default:
		return false
	}
	return true
}

func lower(ch rune) rune     { return ('a' - 'A') | ch } // returns lower-case ch iff ch is ASCII letter
func isLetter(ch rune) bool  { return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_' }
func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }
func isHex(ch rune) bool     { return '0' <= ch && ch <= '9' || 'a' <= lower(ch) && lower(ch) <= 'f' }

// digits accepts the sequence { digit | '_' }.
// If base <= 10, digits accepts any decimal digit but records
// the index (relative to the literal start) of a digit >= base
// in *invalid, if *invalid < 0.
// digits returns a bitset describing whether the sequence contained
// digits (bit 0 is set), or separators '_' (bit 1 is set).
func (s *scanner) digits(base int, invalid *int) (digsep int) {
	if base <= 10 {
		max := rune('0' + base)
		for isDecimal(s.ch) || s.ch == '_' {
			ds := 1
			if s.ch == '_' {
				ds = 2
			} else if s.ch >= max && *invalid < 0 {
				_, col := s.pos()
				*invalid = int(col - s.col) // record invalid rune index
			}
			digsep |= ds
			s.nextch()
		}
	} else {
		for isHex(s.ch) || s.ch == '_' {
			ds := 1
			if s.ch == '_' {
				ds = 2
			}
			digsep |= ds
			s.nextch()
		}
	}
	return
}

func (s *scanner) number(seenPoint bool) {
	ok := true
	kind := Int
	base := 10        // number base
	prefix := rune(0) // one of 0 (decimal), '0' (0-octal), 'x', 'o', or 'b'
	digsep := 0       // bit 0: digit present, bit 1: '_' present
	invalid := -1     // index of invalid digit in literal, or < 0

	// integer part
	if !seenPoint {
		if s.ch == '0' {
			s.nextch()
			switch lower(s.ch) {
			case 'x':
				s.nextch()
				base, prefix = 16, 'x'
			case 'o':
				s.nextch()
				base, prefix = 8, 'o'
			case 'b':
				s.nextch()
				base, prefix = 2, 'b'
			default:
				base, prefix = 8, '0'
				digsep = 1 // leading 0
			}
		}
		digsep |= s.digits(base, &invalid)
		if s.ch == '.' {
			if prefix == 'o' || prefix == 'b' {
				s.errorf("invalid radix point in %s literal", baseName(base))
				ok = false
			}
			s.nextch()
			seenPoint = true
		}
	}

	// fractional part
	if seenPoint {
		kind = Float
		digsep |= s.digits(base, &invalid)
	}

	if digsep&1 == 0 && ok {
		s.errorf("%s literal has no digits", baseName(base))
		ok = false
	}

	// exponent
	if e := lower(s.ch); e == 'e' || e == 'p' {
		if ok {
			switch {
			case e == 'e' && prefix != 0 && prefix != '0':
				s.errorf("%q exponent requires decimal mantissa", s.ch)
				ok = false
			case e == 'p' && prefix != 'x':
				s.errorf("%q exponent requires hexadecimal mantissa", s.ch)
				ok = false
			}
		}
		s.nextch()
		kind = Float
		if s.ch == '+' || s.ch == '-' {
			s.nextch()
		}
		digsep = s.digits(10, nil) | digsep&2 // don't lose sep bit
		if digsep&1 == 0 && ok {
			s.errorf("exponent has no digits")
			ok = false
		}
	} else if prefix == 'x' && kind == Float && ok {
		s.errorf("hexadecimal mantissa requires a 'p' exponent")
		ok = false
	}

	s.setLiteral(kind, ok) // do this now so we can use s.literal below

	if kind == Int && invalid >= 0 && ok {
		s.errorAtf(invalid, "invalid digit %q in %s literal", s.literal[invalid], baseName(base))
		ok = false
	}

	if digsep&2 != 0 && ok {
		if i := invalidNumberSep(s.literal); i >= 0 {
			s.errorAtf(i, "'_' must separate successive digits")
			ok = false
		}
	}

	s.bad = !ok // correct s.bad
}

func baseName(base int) string {
	switch base {
	case 2:
		return "binary"
	case 8:
		return "octal"
	case 10:
		return "decimal"
	case 16:
		return "hexadecimal"
	}
	panic("invalid base")
}

// invalidNumberSep returns the index of the first invalid separator in x, or -1.
func invalidNumberSep(x string) int {
	x1 := ' ' // prefix char, we only care if it's 'x'
	d := '.'  // digit, one of '_', '0' (a digit), or '.' (anything else)
	i := 0

	// a prefix counts as a digit
	if len(x) >= 2 && x[0] == '0' {
		x1 = lower(rune(x[1]))
		if x1 == 'x' || x1 == 'o' || x1 == 'b' {
			d = '0'
			i = 2
		}
	}

	// mantissa and exponent
	for ; i < len(x); i++ {
		p := d // previous digit
		d = rune(x[i])
		switch {
		case d == '_':
			if p != '0' {
				return i
			}
		case isDecimal(d) || x1 == 'x' && isHex(d):
			d = '0'
		default:
			if p == '_' {
				return i - 1
			}
			d = '.'
		}
	}
	if d == '_' {
		return len(x) - 1
	}

	return -1
}

func (s *scanner) rune() {
	ok := true
	s.nextch()

	n := 0
	for ; ; n++ {
		if s.ch == '\'' {
			if ok {
				if n == 0 {
					s.errorf("empty rune literal or unescaped '")
					ok = false
				} else if n != 1 {
					s.errorAtf(0, "more than one character in rune literal")
					ok = false
				}
			}
			s.nextch()
			break
		}
		if s.ch == '\\' {
			s.nextch()
			if !s.escape('\'') {
				ok = false
			}
			continue
		}
		if s.ch == '\n' {
			if ok {
				s.errorf("newline in rune literal")
				ok = false
			}
			break
		}
		if s.ch < 0 {
			if ok {
				s.errorAtf(0, "rune literal not terminated")
				ok = false
			}
			break
		}
		s.nextch()
	}

	s.setLiteral(Rune, ok)
}

func (s *scanner) standardString() {
	ok := true
	s.nextch()

	for {
		if s.ch == '"' {
			s.nextch()
			break
		}
		if s.ch == '\\' {
			s.nextch()
			if !s.escape('"') {
				ok = false
			}
			continue
		}
		if s.ch == '\n' {
			s.errorf("newline in string")
			ok = false
			break
		}
		if s.ch < 0 {
			s.errorAtf(0, "string not terminated")
			ok = false
			break
		}
		s.nextch()
	}

	s.setLiteral(String, ok)
}

func (s *scanner) rawString() {
	ok := true
	s.nextch()

	for {
		if s.ch == '`' {
			s.nextch()
			break
		}
		if s.ch < 0 {
			s.errorAtf(0, "string not terminated")
			ok = false
			break
		}
		s.nextch()
	}
	// We leave CRs in the string since they are part of the
	// literal (even though they are not part of the literal
	// value).

	s.setLiteral(String, ok)
}

func (s *scanner) skipLine() {
	// don't consume '\n' - needed for nlsemi logic
	for s.ch >= 0 && s.ch != '\n' {
		s.nextch()
	}
}

func (s *scanner) lineComment() (keep bool) {
	// opening has already been consumed

	if s.mode&emitComments != 0 {
		s.skipLine()
		s.setLiteral(Comment, true)
		return true
	}

	s.stop()
	s.skipLine()
	return false
}

func (s *scanner) skipBlockComment() (ok bool) {
	for s.ch >= 0 {
		for s.ch == '*' {
			s.nextch()
			if s.ch == '/' {
				s.nextch()
				return true
			}
		}
		s.nextch()
	}
	s.errorAtf(0, "comment not terminated")
	return false
}

func (s *scanner) blockComment() (keep bool) {
	// opening has already been consumed

	if s.mode&emitComments != 0 {
		ok := s.skipBlockComment()
		s.setLiteral(Comment, ok)
		return true
	}

	s.stop()
	s.skipBlockComment()
	return false
}

func (s *scanner) escape(quote rune) (ok bool) {
	var n int
	var base, max uint32

	switch s.ch {
	case quote, 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\':
		s.nextch()
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.nextch()
		n, base, max = 2, 16, 255
	case 'u':
		s.nextch()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.nextch()
		n, base, max = 8, 16, unicode.MaxRune
	default:
		if s.ch < 0 {
			return true // complain in caller about EOF
		}
		s.errorf("unknown escape")
		return false
	}

	var x uint32
	for i := n; i > 0; i-- {
		if s.ch < 0 {
			return true // complain in caller about EOF
		}
		d := base
		if isDecimal(s.ch) {
			d = uint32(s.ch) - '0'
		} else if 'a' <= lower(s.ch) && lower(s.ch) <= 'f' {
			d = uint32(lower(s.ch)) - 'a' + 10
		}
		if d >= base {
			s.errorf("invalid character %q in %s escape", s.ch, baseName(int(base)))
			return false
		}
		// d < base
		x = x*base + d
		s.nextch()
	}

	if x > max && base == 8 {
		s.errorf("octal escape value %d > 255", x)
		return false
	}

	if x > max || 0xD800 <= x && x < 0xE000 /* surrogate range */ {
		s.errorf("escape is invalid Unicode code point %#U", x)
		return false
	}

	return true
}
