package gospace

import (
	"bufio"
	"io"

	"github.com/icza/bitio"
)

type Token uint8

const (
	EOF Token = iota
	Space
	Tab
	LF
)

type SpaceLexer interface {
	Read() (Token, error)
	Pos() (int, int)
}

type Lexer struct {
	br   io.ByteReader
	line int
	col  int
}

func NewLexer(r io.Reader) *Lexer {
	var br io.ByteReader
	br, ok := r.(io.ByteReader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &Lexer{br, 1, 1}
}

func (l *Lexer) Next() (Token, error) {
	for {
		b, err := l.br.ReadByte()
		if err == io.EOF {
			return EOF, nil
		}
		if err != nil {
			return EOF, err
		}
		l.col++
		switch b {
		case ' ':
			return Space, nil
		case '\t':
			return Tab, nil
		case '\n':
			l.line++
			l.col = 1
			return LF, nil
		}
	}
}

func (l *Lexer) Pos() (int, int) {
	return l.line, l.col
}

type BitLexer struct {
	br  bitio.Reader
	pos int
}

func NewBitLexer(r io.Reader) *BitLexer {
	var br bitio.Reader
	br, ok := r.(bitio.Reader)
	if !ok {
		br = bitio.NewReader(r)
	}
	return &BitLexer{br, 0}
}

func (l *BitLexer) Next() (Token, error) {
	b, err := l.br.ReadBool()
	if err == io.EOF {
		return EOF, nil
	}
	if err != nil {
		return EOF, err
	}
	l.pos++
	if !b {
		return Space, nil
	}
	b, err = l.br.ReadBool()
	if err == io.EOF {
		return EOF, nil
	}
	if err != nil {
		return EOF, err
	}
	l.pos++
	if b {
		return LF, nil
	}
	return Tab, nil
}

func (l *BitLexer) Pos() (int, int) {
	return (l.pos / 8) + 1, (l.pos % 8) + 1
}
